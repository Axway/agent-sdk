package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func (c *ServiceClient) buildAPIServiceInstanceSpec(
	serviceBody *ServiceBody,
	endPoints []v1alpha1.ApiServiceInstanceSpecEndpoint,
) v1alpha1.ApiServiceInstanceSpec {
	return v1alpha1.ApiServiceInstanceSpec{
		ApiServiceRevision: serviceBody.serviceContext.revisionName,
		Endpoint:           endPoints,
	}
}

func (c *ServiceClient) buildAPIServiceInstanceResource(
	serviceBody *ServiceBody,
	instanceName string,
	instanceAttributes map[string]string,
	endPoints []v1alpha1.ApiServiceInstanceSpecEndpoint,
) *v1alpha1.APIServiceInstance {
	return &v1alpha1.APIServiceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceInstanceGVK(),
			Name:             instanceName,
			Title:            serviceBody.NameToPush,
			Attributes:       c.buildAPIResourceAttributes(serviceBody, instanceAttributes, false),
			Tags:             c.mapToTagsArray(serviceBody.Tags),
		},
		Spec:  c.buildAPIServiceInstanceSpec(serviceBody, endPoints),
		Owner: c.getOwnerObject(serviceBody, false),
	}
}

func (c *ServiceClient) updateInstanceResource(
	instance *v1alpha1.APIServiceInstance,
	serviceBody *ServiceBody,
	endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint,
) *v1alpha1.APIServiceInstance {
	instance.ResourceMeta.Metadata.ResourceVersion = ""
	instance.Title = serviceBody.NameToPush
	instance.Attributes = c.buildAPIResourceAttributes(serviceBody, instance.Attributes, false)
	instance.Tags = c.mapToTagsArray(serviceBody.Tags)
	instance.Spec = c.buildAPIServiceInstanceSpec(serviceBody, endpoints)
	instance.Owner = c.getOwnerObject(serviceBody, false)
	return instance
}

// processInstance - Creates or updates an API Service Instance based on the current API Service Revision.
func (c *ServiceClient) processInstance(serviceBody *ServiceBody) error {
	instanceEndpoints, err := c.createInstanceEndpoint(serviceBody.Endpoints)
	if err != nil {
		return err
	}

	var httpMethod string
	var instance *v1alpha1.APIServiceInstance

	instanceURL := c.cfg.GetInstancesURL()
	instancePrefix := c.getRevisionPrefix(serviceBody)
	instanceName := instancePrefix + "." + strconv.Itoa(serviceBody.serviceContext.revisionCount)

	if serviceBody.serviceContext.revisionAction == addAPI {
		httpMethod = http.MethodPost
		instanceAttributes := serviceBody.InstanceAttributes
		if instanceAttributes == nil {
			instanceAttributes = make(map[string]string)
		}
		instance = c.buildAPIServiceInstanceResource(serviceBody, instanceName, instanceAttributes, instanceEndpoints)
	}

	if serviceBody.serviceContext.revisionAction == updateAPI {
		httpMethod = http.MethodPut
		instances, err := c.getRevisionInstances(instanceName, instanceURL)
		if err != nil {
			return err
		}
		if len(instances) == 0 {
			return fmt.Errorf("no instance found named '%s' for revision '%s'", instanceName, serviceBody.serviceContext.revisionName)
		}
		instanceURL = instanceURL + "/" + instanceName
		instance = c.updateInstanceResource(instances[0], serviceBody, instanceEndpoints)
	}

	buffer, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	_, err = c.apiServiceDeployAPI(httpMethod, instanceURL, buffer)
	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, rollbackErr := c.rollbackAPIService(*serviceBody, serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				return errors.New(err.Error() + rollbackErr.Error())
			}
		}
		return err
	}

	serviceBody.serviceContext.instanceName = instanceName

	return err
}

func (c *ServiceClient) createInstanceEndpoint(endpoints []EndpointDefinition) ([]v1alpha1.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := make([]v1alpha1.ApiServiceInstanceSpecEndpoint, 0)
	var err error

	// To set your own endpoints call AddServiceEndpoint/SetServiceEndpoint on the ServiceBodyBuilder.
	// Any endpoints provided from the ServiceBodyBuilder will override the endpoints found in the spec.
	if len(endpoints) > 0 {
		for _, endpointDef := range endpoints {
			ep := v1alpha1.ApiServiceInstanceSpecEndpoint{
				Host:     endpointDef.Host,
				Port:     endpointDef.Port,
				Protocol: endpointDef.Protocol,
				Routing: v1alpha1.ApiServiceInstanceSpecRouting{
					BasePath: endpointDef.BasePath,
				},
			}
			endPoints = append(endPoints, ep)
		}
	} else {
		log.Debug("Processing API service instance with no endpoint")
	}

	if err != nil {
		return nil, err
	}

	return endPoints, nil
}

func (c *ServiceClient) getRevisionInstances(instanceName, url string) ([]*v1alpha1.APIServiceInstance, error) {
	// Check if instances exist for the current revision.
	queryParams := map[string]string{
		"query": "name==" + instanceName,
	}

	return c.GetAPIServiceInstances(queryParams, url)
}

// GetAPIServiceInstanceByName - Returns the API service instance for specified name
func (c *ServiceClient) GetAPIServiceInstanceByName(instanceName string) (*v1alpha1.APIServiceInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetInstancesURL() + "/" + instanceName,
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
	apiInstance := new(v1alpha1.APIServiceInstance)
	json.Unmarshal(response.Body, apiInstance)
	return apiInstance, nil
}
