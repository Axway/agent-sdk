package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Axway/agent-sdk/pkg/util"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func buildAPIServiceSpec(serviceBody *ServiceBody) management.ApiServiceSpec {
	if serviceBody.Image != "" {
		return management.ApiServiceSpec{
			Description: serviceBody.Description,
			Icon: management.ApiServiceSpecIcon{
				ContentType: serviceBody.ImageContentType,
				Data:        serviceBody.Image,
			},
		}
	}
	return management.ApiServiceSpec{
		Description: serviceBody.Description,
	}
}

func (c *ServiceClient) buildAPIService(serviceBody *ServiceBody) *management.APIService {
	owner, ownerErr := c.getOwnerObject(serviceBody, true)
	svc := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.APIServiceGVK(),
			Title:            serviceBody.NameToPush,
			Attributes:       util.CheckEmptyMapStringString(serviceBody.ServiceAttributes),
			Tags:             mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish()),
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Kind: management.EnvironmentGVK().Kind,
					Name: c.cfg.GetEnvironmentName(),
				},
			},
		},
		Spec:   buildAPIServiceSpec(serviceBody),
		Owner:  owner,
		Status: buildAPIServiceStatusSubResource(&apiv1.ResourceStatus{}, ownerErr),
	}

	buildAPIServiceSourceSubResource(svc, serviceBody)
	svcDetails := buildAgentDetailsSubResource(serviceBody, true, serviceBody.ServiceAgentDetails)
	c.setMigrationFlags(svcDetails)

	util.SetAgentDetails(svc, svcDetails)

	return svc
}

func (c *ServiceClient) setMigrationFlags(svcDetails map[string]interface{}) {
	svcDetails[defs.MarketplaceMigration] = defs.MigrationCompleted
}

func (c *ServiceClient) getOwnerObject(serviceBody *ServiceBody, warning bool) (*apiv1.Owner, error) {
	if id, found := c.getTeamFromCache(serviceBody.TeamName); found {
		return &apiv1.Owner{
			Type: apiv1.TeamOwner,
			ID:   id,
		}, nil
	} else if warning {
		// warning is only true when creating service, revision and instance will not print it
		warnMsg := fmt.Sprintf("A team named %s does not exist on Amplify, not setting an owner of the API Service for %s", serviceBody.TeamName, serviceBody.APIName)
		log.Warnf("%s", warnMsg)
		return nil, errors.New(warnMsg)
	}
	return nil, nil
}

func (c *ServiceClient) updateAPIService(serviceBody *ServiceBody, svc *management.APIService) {
	owner, ownerErr := c.getOwnerObject(serviceBody, true)

	svc.GroupVersionKind = management.APIServiceGVK()
	svc.Metadata.ResourceVersion = ""
	svc.Title = serviceBody.NameToPush
	svc.Tags = mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish())
	svc.Spec.Description = serviceBody.Description
	svc.Owner = owner
	svc.Attributes = util.CheckEmptyMapStringString(serviceBody.ServiceAttributes)
	svc.Status = buildAPIServiceStatusSubResource(svc.Status, ownerErr)
	buildAPIServiceSourceSubResource(svc, serviceBody)

	svcDetails := buildAgentDetailsSubResource(serviceBody, true, serviceBody.ServiceAgentDetails)
	newSVCDetails := util.MergeMapStringInterface(util.GetAgentDetails(svc), svcDetails)
	util.SetAgentDetails(svc, newSVCDetails)

	// get the specHashes from the existing service
	if revDetails, found := newSVCDetails[specHashes]; found {
		if specHashes, ok := revDetails.(map[string]interface{}); ok {
			serviceBody.specHashes = specHashes
		}
	}

	if serviceBody.Image != "" {
		svc.Spec.Icon = management.ApiServiceSpecIcon{
			ContentType: serviceBody.ImageContentType,
			Data:        serviceBody.Image,
		}
	}
}

func buildAPIServiceSourceSubResource(svc *management.APIService, serviceBody *ServiceBody) {
	serviceBody.serviceContext.updateServiceSource = false

	source := svc.Source
	if source == nil {
		svc.Source = &management.ApiServiceSource{}
		source = svc.Source
	}

	dataplaneType := serviceBody.GetDataplaneType()
	if dataplaneType != "" {
		if source.DataplaneType == nil {
			source.DataplaneType = &management.ApiServiceSourceDataplaneType{}
		}
		if serviceBody.IsDesignDataplane() {
			if source.DataplaneType.Design != dataplaneType.String() {
				source.DataplaneType.Design = dataplaneType.String()
				serviceBody.serviceContext.updateServiceSource = true
			}
		} else if source.DataplaneType.Managed != dataplaneType.String() {
			source.DataplaneType.Managed = dataplaneType.String()
			serviceBody.serviceContext.updateServiceSource = true
		}
	}

	referencedSvc := serviceBody.GetReferencedServiceName()
	if referencedSvc != "" {
		if source.References == nil {
			source.References = &management.ApiServiceSourceReferences{}
		}
		if source.References.ApiService != referencedSvc {
			source.References.ApiService = serviceBody.GetReferencedServiceName()
			serviceBody.serviceContext.updateServiceSource = true
		}
	}
}

func buildAPIServiceStatusSubResource(status *apiv1.ResourceStatus, ownerErr error) *apiv1.ResourceStatus {
	// only set status if ownerErr != nil
	if ownerErr == nil || status == nil {
		return nil
	}

	// if the top reason is the same then return the existing status
	if len(status.Reasons) > 0 && status.Reasons[0].Detail == ownerErr.Error() {
		return status
	}

	// get current time
	activityTime := time.Now()
	newV1Time := apiv1.Time(activityTime)
	message := ownerErr.Error()
	level := "Error"
	return &apiv1.ResourceStatus{
		Level: level,
		Reasons: []apiv1.ResourceStatusReason{
			{
				Type:      level,
				Detail:    message,
				Timestamp: newV1Time,
			},
		},
	}
}

// processService -
func (c *ServiceClient) processService(serviceBody *ServiceBody) (*management.APIService, error) {
	// Default action to create service
	serviceURL := c.cfg.GetServicesURL()
	httpMethod := http.MethodPost
	serviceBody.serviceContext.serviceAction = addAPI
	serviceBody.specHashes = map[string]interface{}{}

	// If service exists, update existing service
	svc, err := c.getAPIServiceFromCache(serviceBody)
	if err != nil {
		return nil, err
	}

	if svc != nil {
		serviceBody.serviceContext.serviceAction = updateAPI
		httpMethod = http.MethodPut
		serviceURL += "/" + svc.Name
		c.updateAPIService(serviceBody, svc)
	} else {
		svc = c.buildAPIService(serviceBody)
	}

	buffer, err := json.Marshal(svc)
	if err != nil {
		return nil, err
	}

	ri, err := c.apiServiceDeployAPI(httpMethod, serviceURL, buffer)
	if err != nil {
		return nil, err
	}

	if ri != nil {
		serviceBody.serviceContext.serviceName = ri.Name
		serviceBody.serviceContext.serviceID = ri.Metadata.ID
	}

	svc.Name = serviceBody.serviceContext.serviceName
	err = c.updateAPIServiceSubresources(svc, serviceBody.serviceContext.updateServiceSource)
	if err != nil && serviceBody.serviceContext.serviceAction == addAPI {
		_, e := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
		if e != nil {
			return nil, errors.New(err.Error() + e.Error())
		}
	}

	ri, _ = svc.AsInstance()
	c.caches.AddAPIService(ri)
	return svc, err
}

func (c *ServiceClient) updateAPIServiceSubresources(svc *management.APIService, updateSource bool) error {
	subResources := make(map[string]interface{})
	if svc.Status != nil {
		subResources[management.ApiServiceStatusSubResourceName] = svc.Status
	}

	if len(svc.SubResources) > 0 {
		if xAgentDetail, ok := svc.SubResources[defs.XAgentDetails]; ok {
			subResources[defs.XAgentDetails] = xAgentDetail
		}
	}

	if updateSource && svc.Source != nil {
		subResources[management.ApiServiceSourceSubResourceName] = svc.Source
	}

	if len(subResources) > 0 {
		return c.CreateSubResource(svc.ResourceMeta, subResources)
	}
	return nil
}

func (c *ServiceClient) getAPIServiceByExternalAPIID(apiID string) (*management.APIService, error) {
	ri := c.caches.GetAPIServiceWithAPIID(apiID)
	if ri == nil {
		return nil, nil
	}
	apiSvc := &management.APIService{}
	err := apiSvc.FromInstance(ri)
	return apiSvc, err
}

func (c *ServiceClient) getAPIServiceByPrimaryKey(primaryKey string) (*management.APIService, error) {
	ri := c.caches.GetAPIServiceWithPrimaryKey(primaryKey)
	if ri == nil {
		return nil, nil
	}
	apiSvc := &management.APIService{}
	err := apiSvc.FromInstance(ri)
	return apiSvc, err
}

func (c *ServiceClient) getAPIServiceFromCache(serviceBody *ServiceBody) (*management.APIService, error) {
	apiService, err := c.getAPIServiceByExternalAPIID(serviceBody.RestAPIID)
	if apiService != nil && err == nil {
		return apiService, nil
	}

	if serviceBody.PrimaryKey != "" {
		apiService, err = c.getAPIServiceByPrimaryKey(serviceBody.PrimaryKey)
	}
	return apiService, err
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (c *ServiceClient) rollbackAPIService(name string) (*apiv1.ResourceInstance, error) {
	return c.apiServiceDeployAPI(http.MethodDelete, c.cfg.DeleteServicesURL()+"/"+name, nil)
}

// GetAPIServiceByName - Returns the API service based on its name
func (c *ServiceClient) GetAPIServiceByName(name string) (*management.APIService, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}
	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetServicesURL() + "/" + name,
		Headers: headers,
	}
	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		if response.Code != http.StatusNotFound {
			responseErr := readResponseErrors(response.Code, response.Body)
			return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
		}
		return nil, nil
	}
	apiService := new(management.APIService)
	err = json.Unmarshal(response.Body, apiService)
	return apiService, err
}
