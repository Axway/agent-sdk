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
		Status: buildAPIServiceStatusSubResource(ownerErr),
	}

	svcDetails := buildAgentDetailsSubResource(serviceBody, true, serviceBody.ServiceAgentDetails)
	c.setMigrationFlags(svcDetails)

	util.SetAgentDetails(svc, svcDetails)

	return svc
}

func (c *ServiceClient) setMigrationFlags(svcDetails map[string]interface{}) {
	svcDetails[defs.InstanceMigration] = defs.MigrationCompleted

	if c.cfg.IsMarketplaceSubsEnabled() {
		svcDetails[defs.MarketplaceMigration] = defs.MigrationCompleted
	}
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
		log.Warnf(warnMsg)
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
	svc.Status = buildAPIServiceStatusSubResource(ownerErr)

	svcDetails := buildAgentDetailsSubResource(serviceBody, true, serviceBody.ServiceAgentDetails)
	sub := util.MergeMapStringInterface(util.GetAgentDetails(svc), svcDetails)
	util.SetAgentDetails(svc, sub)

	if serviceBody.Image != "" {
		svc.Spec.Icon = management.ApiServiceSpecIcon{
			ContentType: serviceBody.ImageContentType,
			Data:        serviceBody.Image,
		}
	}
}

func buildAPIServiceStatusSubResource(ownerErr error) *apiv1.ResourceStatus {
	// only set status if ownerErr != nil
	if ownerErr != nil {
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
	return nil
}

// processService -
func (c *ServiceClient) processService(serviceBody *ServiceBody) (*management.APIService, error) {
	// Default action to create service
	serviceURL := c.cfg.GetServicesURL()
	httpMethod := http.MethodPost
	serviceBody.serviceContext.serviceAction = addAPI

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

	serviceBody.serviceContext.serviceName, err = c.apiServiceDeployAPI(httpMethod, serviceURL, buffer)
	if err != nil {
		return nil, err
	}

	svc.Name = serviceBody.serviceContext.serviceName
	err = c.updateAPIServiceSubresources(svc)
	if err != nil && serviceBody.serviceContext.serviceAction == addAPI {
		_, e := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
		if e != nil {
			return nil, errors.New(err.Error() + e.Error())
		}
	}

	ri, _ := svc.AsInstance()
	c.caches.AddAPIService(ri)
	return svc, err
}

func (c *ServiceClient) updateAPIServiceSubresources(svc *management.APIService) error {
	subResources := make(map[string]interface{})
	if svc.Status != nil {
		subResources["status"] = svc.Status
	}

	if len(svc.SubResources) > 0 {
		if xAgentDetail, ok := svc.SubResources[defs.XAgentDetails]; ok {
			subResources[defs.XAgentDetails] = xAgentDetail
		}
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
	if serviceBody.PrimaryKey != "" {
		apiService, err := c.getAPIServiceByPrimaryKey(serviceBody.PrimaryKey)
		if apiService != nil && err == nil {
			return apiService, err
		}
	}
	return c.getAPIServiceByExternalAPIID(serviceBody.RestAPIID)
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (c *ServiceClient) rollbackAPIService(name string) (string, error) {
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
