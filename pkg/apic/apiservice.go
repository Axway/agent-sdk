package apic

import (
	"encoding/json"
	"errors"
	"net/http"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/google/uuid"
)

func (c *ServiceClient) buildAPIServiceSpec(serviceBody *ServiceBody) v1alpha1.ApiServiceSpec {
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
	return &v1alpha1.APIService{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             serviceName,
			Title:            serviceBody.NameToPush,
			Attributes:       c.buildAPIResourceAttributes(serviceBody, nil, true),
			Tags:             c.mapToTagsArray(serviceBody.Tags),
		},
		Spec: c.buildAPIServiceSpec(serviceBody),
	}
}

func (c *ServiceClient) updateAPIServiceResource(apiSvc *v1alpha1.APIService, serviceBody *ServiceBody) {
	apiSvc.ResourceMeta.Metadata.ResourceVersion = ""
	apiSvc.Title = serviceBody.NameToPush
	apiSvc.ResourceMeta.Attributes = c.buildAPIResourceAttributes(serviceBody, apiSvc.ResourceMeta.Attributes, true)
	apiSvc.ResourceMeta.Tags = c.mapToTagsArray(serviceBody.Tags)
	apiSvc.Spec.Description = serviceBody.Description
	if serviceBody.Image != "" {
		apiSvc.Spec.Icon = v1alpha1.ApiServiceSpecIcon{
			ContentType: serviceBody.ImageContentType,
			Data:        serviceBody.Image,
		}
	}
}

//processService -
func (c *ServiceClient) processService(serviceBody *ServiceBody) (*v1alpha1.APIService, error) {
	uuid, _ := uuid.NewUUID()
	serviceName := uuid.String()

	// Default action to create service
	serviceURL := c.cfg.GetServicesURL()
	httpMethod := http.MethodPost
	serviceBody.serviceContext.serviceAction = addAPI

	// If service exists, update existing service
	apiService, err := c.getAPIServiceByExternalAPIID(serviceBody)
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
	return apiService, err
}

// deleteService
func (c *ServiceClient) deleteServiceByAPIID(externalAPIID string) error {
	svc, err := c.getAPIServiceByAttribute(externalAPIID, "")
	if err != nil {
		return err
	}
	if svc == nil {
		return errors.New("no API Service found for externalAPIID " + externalAPIID)
	}
	_, err = c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetServicesURL()+"/"+svc.Name, nil)
	if err != nil {
		return err
	}
	return nil
}

// getAPIServiceByExternalAPIID - Returns the API service based on external api identification
func (c *ServiceClient) getAPIServiceByExternalAPIID(serviceBody *ServiceBody) (*v1alpha1.APIService, error) {
	if serviceBody.PrimaryKey != "" {
		apiService, err := c.getAPIServiceByAttribute(serviceBody.RestAPIID, serviceBody.PrimaryKey)
		if apiService != nil || err != nil {
			return apiService, err
		}
	}
	return c.getAPIServiceByAttribute(serviceBody.RestAPIID, "")
}

// getAPIServiceByAttribute - Returns the API service based on attribute
func (c *ServiceClient) getAPIServiceByAttribute(externalAPIID, primaryKey string) (*v1alpha1.APIService, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}
	query := map[string]string{}
	if primaryKey != "" {
		query["query"] = "attributes." + AttrExternalAPIPrimaryKey + "==\"" + primaryKey + "\""
	} else {
		query["query"] = "attributes." + AttrExternalAPIID + "==\"" + externalAPIID + "\""
	}

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetServicesURL(),
		Headers:     headers,
		QueryParams: query,
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
	apiServices := make([]v1alpha1.APIService, 0)
	json.Unmarshal(response.Body, &apiServices)
	if len(apiServices) > 0 {
		return &apiServices[0], nil
	}
	return nil, nil
}

// rollbackAPIService - if the process to add api/revision/instance fails, delete the api that was created
func (c *ServiceClient) rollbackAPIService(serviceBody ServiceBody, name string) (string, error) {
	return c.apiServiceDeployAPI(http.MethodDelete, c.cfg.DeleteServicesURL()+"/"+name, nil)
}
