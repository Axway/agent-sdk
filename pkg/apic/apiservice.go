package apic

import (
	"encoding/json"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/google/uuid"
)

func buildAPIServiceSpec(serviceBody *ServiceBody) v1alpha1.ApiServiceSpec {
	if serviceBody.Image != "" {
		return v1alpha1.ApiServiceSpec{
			Description: serviceBody.Description,
			Icon: v1alpha1.ApiServiceSpecIcon{
				ContentType: serviceBody.ImageContentType,
				Data:        serviceBody.Image,
			},
		}
	}
	return v1alpha1.ApiServiceSpec{
		Description: serviceBody.Description,
	}
}

func (c *ServiceClient) buildAPIServiceResource(serviceBody *ServiceBody, serviceName string) *v1alpha1.APIService {
	svc := &v1alpha1.APIService{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             serviceName,
			Title:            serviceBody.NameToPush,
			Attributes:       buildAPIResourceAttributes(serviceBody, nil),
			Tags:             mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish()),
		},
		Spec:  buildAPIServiceSpec(serviceBody),
		Owner: c.getOwnerObject(serviceBody, true),
	}

	svc.SetSubResource(definitions.XAgentDetails, buildAgentDetailsSubResource(serviceBody, true))

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

func (c *ServiceClient) updateAPIServiceResource(svc *v1alpha1.APIService, serviceBody *ServiceBody) {
	svc.ResourceMeta.Metadata.ResourceVersion = ""
	svc.Title = serviceBody.NameToPush
	svc.ResourceMeta.Tags = mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish())
	svc.Spec.Description = serviceBody.Description
	svc.Owner = c.getOwnerObject(serviceBody, true)
	svc.ResourceMeta.Attributes = buildAPIResourceAttributes(serviceBody, svc.ResourceMeta.Attributes)
	svc.SetSubResource(definitions.XAgentDetails, buildAgentDetailsSubResource(serviceBody, true))

	if serviceBody.Image != "" {
		svc.Spec.Icon = v1alpha1.ApiServiceSpecIcon{
			ContentType: serviceBody.ImageContentType,
			Data:        serviceBody.Image,
		}
	}
}

// processService -
func (c *ServiceClient) processService(serviceBody *ServiceBody) (*v1alpha1.APIService, error) {
	uuid, _ := uuid.NewUUID()
	serviceName := uuid.String()

	// Default action to create service
	serviceURL := c.cfg.GetServicesURL()
	httpMethod := http.MethodPost
	serviceBody.serviceContext.serviceAction = addAPI

	// If service exists, update existing service
	apiService, err := c.getAPIServiceFromCache(serviceBody)
	if err != nil {
		return nil, err
	}

	if apiService != nil {
		serviceName = apiService.Name
		serviceBody.serviceContext.serviceAction = updateAPI
		httpMethod = http.MethodPut
		serviceURL += "/" + serviceName
		c.updateAPIServiceResource(apiService, serviceBody)
	} else {
		apiService = c.buildAPIServiceResource(serviceBody, serviceName)
	}

	// spec needs to adhere to environment schema

	buffer, err := json.Marshal(apiService)
	if err != nil {
		return nil, err
	}
	_, err = c.apiServiceDeployAPI(httpMethod, serviceURL, buffer)
	if err == nil {
		serviceBody.serviceContext.serviceName = serviceName
	}

	if err == nil {
		if len(apiService.SubResources) > 0 {
			inst, err := apiService.AsInstance()
			if err != nil {
				return apiService, err
			}
			err = c.CreateSubResourceScoped(v1alpha1.EnvironmentResourceName, c.cfg.GetEnvironmentName(), v1alpha1.APIServiceResourceName, inst)
			if err != nil {
				return apiService, err
			}
		}
	}

	return apiService, err
}

func (c *ServiceClient) getAPIServiceByExternalAPIID(apiID string) (*v1alpha1.APIService, error) {
	ri := c.caches.GetAPIServiceWithAPIID(apiID)
	if ri == nil {
		return nil, nil
	}
	apiSvc := &v1alpha1.APIService{}
	err := apiSvc.FromInstance(ri)
	return apiSvc, err
}

func (c *ServiceClient) getAPIServiceByPrimaryKey(primaryKey string) (*v1alpha1.APIService, error) {
	ri := c.caches.GetAPIServiceWithPrimaryKey(primaryKey)
	if ri == nil {
		return nil, nil
	}
	apiSvc := &v1alpha1.APIService{}
	err := apiSvc.FromInstance(ri)
	return apiSvc, err
}

func (c *ServiceClient) getAPIServiceFromCache(serviceBody *ServiceBody) (*v1alpha1.APIService, error) {
	if serviceBody.PrimaryKey != "" {
		return c.getAPIServiceByPrimaryKey(serviceBody.PrimaryKey)
	}
	return c.getAPIServiceByExternalAPIID(serviceBody.RestAPIID)
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (c *ServiceClient) rollbackAPIService(name string) (string, error) {
	return c.apiServiceDeployAPI(http.MethodDelete, c.cfg.DeleteServicesURL()+"/"+name, nil)
}

// GetAPIServiceByName - Returns the API service based on its name
func (c *ServiceClient) GetAPIServiceByName(name string) (*v1alpha1.APIService, error) {
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
	apiService := new(v1alpha1.APIService)
	json.Unmarshal(response.Body, apiService)
	return apiService, nil
}
