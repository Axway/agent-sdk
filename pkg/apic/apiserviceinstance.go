package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/util"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func buildAPIServiceInstanceSpec(
	serviceBody *ServiceBody,
	endPoints []mv1a.ApiServiceInstanceSpecEndpoint,
) mv1a.ApiServiceInstanceSpec {
	return mv1a.ApiServiceInstanceSpec{
		ApiServiceRevision: serviceBody.serviceContext.revisionName,
		Endpoint:           endPoints,
	}
}

func (c *ServiceClient) buildAPIServiceInstance(
	serviceBody *ServiceBody,
	name string,
	endpoints []mv1a.ApiServiceInstanceSpecEndpoint,
) *mv1a.APIServiceInstance {
	instance := &mv1a.APIServiceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1a.APIServiceInstanceGVK(),
			Name:             name,
			Title:            serviceBody.NameToPush,
			Attributes:       util.MergeMapStringString(serviceBody.ServiceAttributes, serviceBody.InstanceAttributes),
			Tags:             mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish()),
		},
		Spec:  buildAPIServiceInstanceSpec(serviceBody, endpoints),
		Owner: c.getOwnerObject(serviceBody, false),
	}

	instDetails := util.MergeMapStringInterface(serviceBody.ServiceAgentDetails, serviceBody.InstanceAgentDetails)
	details := buildAgentDetailsSubResource(serviceBody, false, instDetails)
	util.SetAgentDetails(instance, details)

	return instance
}

func (c *ServiceClient) updateAPIServiceInstance(
	serviceBody *ServiceBody,
	instance *mv1a.APIServiceInstance,
	endpoints []mv1a.ApiServiceInstanceSpecEndpoint,
) *mv1a.APIServiceInstance {
	instance.GroupVersionKind = mv1a.APIServiceInstanceGVK()
	instance.Metadata.ResourceVersion = ""
	instance.Title = serviceBody.NameToPush
	instance.Attributes = util.MergeMapStringString(serviceBody.ServiceAttributes, serviceBody.InstanceAttributes)
	instance.Tags = mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish())
	instance.Spec = buildAPIServiceInstanceSpec(serviceBody, endpoints)
	instance.Owner = c.getOwnerObject(serviceBody, false)

	details := util.MergeMapStringInterface(serviceBody.ServiceAgentDetails, serviceBody.InstanceAgentDetails)
	util.SetAgentDetails(instance, buildAgentDetailsSubResource(serviceBody, false, details))

	return instance
}

// processInstance - Creates or updates an API Service Instance based on the current API Service Revision.
func (c *ServiceClient) processInstance(serviceBody *ServiceBody) error {
	endpoints, err := createInstanceEndpoint(serviceBody.Endpoints)
	if err != nil {
		return err
	}

	var httpMethod string
	var instance *mv1a.APIServiceInstance

	instanceURL := c.cfg.GetInstancesURL()
	instancePrefix := getRevisionPrefix(serviceBody)
	instanceName := instancePrefix + "." + strconv.Itoa(serviceBody.serviceContext.revisionCount)

	if serviceBody.serviceContext.revisionAction == addAPI {
		httpMethod = http.MethodPost
		instance = c.buildAPIServiceInstance(serviceBody, instanceName, endpoints)
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
		instance = c.updateAPIServiceInstance(serviceBody, instances[0], endpoints)
	}

	buffer, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	_, err = c.apiServiceDeployAPI(httpMethod, instanceURL, buffer)
	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, rollbackErr := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				return errors.New(err.Error() + rollbackErr.Error())
			}
		}
		return err
	}

	if err == nil {
		err = c.CreateSubResourceScoped(
			mv1a.EnvironmentResourceName,
			c.cfg.GetEnvironmentName(),
			instance.PluralName(),
			instance.Name,
			instance.Group,
			instance.APIVersion,
			instance.SubResources,
		)
		if err != nil {
			_, rollbackErr := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				return errors.New(err.Error() + rollbackErr.Error())
			}
		}
	}

	serviceBody.serviceContext.instanceName = instanceName

	return err
}

func createInstanceEndpoint(endpoints []EndpointDefinition) ([]mv1a.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := make([]mv1a.ApiServiceInstanceSpecEndpoint, 0)
	var err error

	// To set your own endpoints call AddServiceEndpoint/SetServiceEndpoint on the ServiceBodyBuilder.
	// Any endpoints provided from the ServiceBodyBuilder will override the endpoints found in the spec.
	if len(endpoints) > 0 {
		for _, endpointDef := range endpoints {
			ep := mv1a.ApiServiceInstanceSpecEndpoint{
				Host:     endpointDef.Host,
				Port:     endpointDef.Port,
				Protocol: endpointDef.Protocol,
				Routing: mv1a.ApiServiceInstanceSpecRouting{
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

func (c *ServiceClient) getRevisionInstances(name, url string) ([]*mv1a.APIServiceInstance, error) {
	// Check if instances exist for the current revision.
	queryParams := map[string]string{
		"query": "name==" + name,
	}

	return c.GetAPIServiceInstances(queryParams, url)
}

// GetAPIServiceInstanceByName - Returns the API service instance for specified name
func (c *ServiceClient) GetAPIServiceInstanceByName(name string) (*mv1a.APIServiceInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetInstancesURL() + "/" + name,
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
	apiInstance := new(mv1a.APIServiceInstance)
	err = json.Unmarshal(response.Body, apiInstance)
	return apiInstance, err
}
