package apic

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/util"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func buildAPIServiceSpec(serviceBody *ServiceBody) mv1a.ApiServiceSpec {
	if serviceBody.Image != "" {
		return mv1a.ApiServiceSpec{
			Description: serviceBody.Description,
			Icon: mv1a.ApiServiceSpecIcon{
				ContentType: serviceBody.ImageContentType,
				Data:        serviceBody.Image,
			},
		}
	}
	return mv1a.ApiServiceSpec{
		Description: serviceBody.Description,
	}
}

func (c *ServiceClient) buildAPIService(serviceBody *ServiceBody) *mv1a.APIService {
	svc := &mv1a.APIService{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1a.APIServiceGVK(),
			Title:            serviceBody.NameToPush,
			Attributes:       util.CheckEmptyMapStringString(serviceBody.ServiceAttributes),
			Tags:             mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish()),
		},
		Spec:  buildAPIServiceSpec(serviceBody),
		Owner: c.getOwnerObject(serviceBody, true),
	}

	svcDetails := buildAgentDetailsSubResource(serviceBody, true, serviceBody.ServiceAgentDetails)
	util.SetAgentDetails(svc, svcDetails)

	return svc
}

func (c *ServiceClient) getOwnerObject(serviceBody *ServiceBody, warning bool) *v1.Owner {
	if id, found := c.getTeamFromCache(serviceBody.TeamName); found {
		return &v1.Owner{
			Type: v1.TeamOwner,
			ID:   id,
		}
	} else if warning {
		// warning is only true when creating service, revision and instance will not print it
		log.Warnf("A team named %s does not exist on Amplify, not setting an owner of the API Service for %s", serviceBody.TeamName, serviceBody.APIName)
	}
	return nil
}

func (c *ServiceClient) updateAPIService(serviceBody *ServiceBody, svc *mv1a.APIService) {
	svc.GroupVersionKind = mv1a.APIServiceGVK()
	svc.Metadata.ResourceVersion = ""
	svc.Title = serviceBody.NameToPush
	svc.Tags = mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish())
	svc.Spec.Description = serviceBody.Description
	svc.Owner = c.getOwnerObject(serviceBody, true)
	svc.Attributes = util.CheckEmptyMapStringString(serviceBody.ServiceAttributes)

	svcDetails := buildAgentDetailsSubResource(serviceBody, true, serviceBody.ServiceAgentDetails)
	util.SetAgentDetails(svc, svcDetails)

	if serviceBody.Image != "" {
		svc.Spec.Icon = mv1a.ApiServiceSpecIcon{
			ContentType: serviceBody.ImageContentType,
			Data:        serviceBody.Image,
		}
	}
}

// processService -
func (c *ServiceClient) processService(serviceBody *ServiceBody) (*v1alpha1.APIService, error) {
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

	// spec needs to adhere to environment schema

	buffer, err := json.Marshal(svc)
	if err != nil {
		return nil, err
	}
	serviceBody.serviceContext.serviceName, err = c.apiServiceDeployAPI(httpMethod, serviceURL, buffer)
	if err != nil {
		return nil, err
	}
	svc.Name = serviceBody.serviceContext.serviceName

	if len(svc.SubResources) > 0 {
		err = c.CreateSubResourceScoped(
			mv1a.EnvironmentResourceName,
			c.cfg.GetEnvironmentName(),
			svc.PluralName(),
			svc.Name,
			svc.Group,
			svc.APIVersion,
			svc.SubResources,
		)

		if err != nil {
			_, e := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
			if e != nil {
				return nil, errors.New(err.Error() + e.Error())
			}
		}
	}

	return svc, err
}

func (c *ServiceClient) getAPIServiceByExternalAPIID(apiID string) (*mv1a.APIService, error) {
	ri := c.caches.GetAPIServiceWithAPIID(apiID)
	if ri == nil {
		return nil, nil
	}
	apiSvc := &mv1a.APIService{}
	err := apiSvc.FromInstance(ri)
	return apiSvc, err
}

func (c *ServiceClient) getAPIServiceByPrimaryKey(primaryKey string) (*mv1a.APIService, error) {
	ri := c.caches.GetAPIServiceWithPrimaryKey(primaryKey)
	if ri == nil {
		return nil, nil
	}
	apiSvc := &mv1a.APIService{}
	err := apiSvc.FromInstance(ri)
	return apiSvc, err
}

func (c *ServiceClient) getAPIServiceFromCache(serviceBody *ServiceBody) (*mv1a.APIService, error) {
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
func (c *ServiceClient) GetAPIServiceByName(name string) (*mv1a.APIService, error) {
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
	apiService := new(mv1a.APIService)
	err = json.Unmarshal(response.Body, apiService)
	return apiService, err
}
