package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func (c *ServiceClient) buildAPIServiceInstanceSpec(serviceBody *ServiceBody, endPoints []v1alpha1.ApiServiceInstanceSpecEndpoint) v1alpha1.ApiServiceInstanceSpec {
	return v1alpha1.ApiServiceInstanceSpec{
		ApiServiceRevision: serviceBody.serviceContext.currentRevision,
		Endpoint:           endPoints,
	}
}

func (c *ServiceClient) buildAPIServiceInstanceResource(serviceBody *ServiceBody, instanceName string,
	instanceAttributes map[string]string, endPoints []v1alpha1.ApiServiceInstanceSpecEndpoint) *v1alpha1.APIServiceInstance {
	return &v1alpha1.APIServiceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceInstanceGVK(),
			Name:             instanceName,
			Title:            serviceBody.NameToPush,
			Attributes:       c.buildAPIResourceAttributes(serviceBody, instanceAttributes, false),
			Tags:             c.mapToTagsArray(serviceBody.Tags),
		},
		Spec: c.buildAPIServiceInstanceSpec(serviceBody, endPoints),
	}
}

func (c *ServiceClient) updateInstanceResource(instance *v1alpha1.APIServiceInstance, serviceBody *ServiceBody, endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint) {
	instance.ResourceMeta.Metadata.ResourceVersion = ""
	instance.Title = serviceBody.NameToPush
	instance.Attributes = c.buildAPIResourceAttributes(serviceBody, instance.Attributes, false)
	instance.Tags = c.mapToTagsArray(serviceBody.Tags)
	instance.Spec = c.buildAPIServiceInstanceSpec(serviceBody, endpoints)
}

//processInstance -
func (c *ServiceClient) processInstance(serviceBody *ServiceBody) error {
	endPoints, err := c.processEndPoints(serviceBody)
	if err != nil {
		return err
	}

	httpMethod := http.MethodPost
	instanceURL := c.cfg.GetInstancesURL()
	instancePrefix := c.getRevisionPrefix(serviceBody)
	instanceName := instancePrefix + "." + strconv.Itoa(serviceBody.serviceContext.instanceCount+1)
	apiInstance := serviceBody.serviceContext.previousInstance

	if serviceBody.serviceContext.instanceAction == updateAPI {
		instanceName = serviceBody.serviceContext.previousInstance.Name
		httpMethod = http.MethodPut
		instanceURL += "/" + instanceName
		c.updateInstanceResource(apiInstance, serviceBody, endPoints)
	} else {
		instanceAttributes := make(map[string]string)
		if serviceBody.serviceContext.previousInstance != nil {
			instanceAttributes[AttrPreviousAPIServiceInstanceID] = serviceBody.serviceContext.previousInstance.Metadata.ID
		}
		apiInstance = c.buildAPIServiceInstanceResource(serviceBody, instanceName, instanceAttributes, endPoints)
	}

	buffer, err := json.Marshal(apiInstance)
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

	serviceBody.serviceContext.currentInstance = instanceName

	return err
}

func (c *ServiceClient) processEndPoints(serviceBody *ServiceBody) ([]v1alpha1.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := make([]v1alpha1.ApiServiceInstanceSpecEndpoint, 0)
	var err error

	// To set your own endpoints call AddServiceEndpoint/SetServiceEndpoint on the ServiceBodyBuilder.
	// Any endpoints provided from the ServiceBodyBuilder will override the endpoints found in the spec.
	if len(serviceBody.Endpoints) > 0 {
		for _, endpointDef := range serviceBody.Endpoints {
			ep := v1alpha1.ApiServiceInstanceSpecEndpoint{
				Host:     endpointDef.Host,
				Port:     endpointDef.Port,
				Protocol: endpointDef.Protocol,
				Routing:  v1alpha1.ApiServiceInstanceSpecRouting{BasePath: endpointDef.BasePath},
			}
			endPoints = append(endPoints, ep)
		}
	} else {
		log.Debug("Processing API service instance with no endpoint")
	}

	err = c.setInstanceAction(serviceBody, endPoints)
	if err != nil {
		return nil, err
	}

	return endPoints, nil
}

func (c *ServiceClient) setInstanceAction(serviceBody *ServiceBody, endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint) error {
	// If service is created in the chain, then set action to create instance
	serviceBody.serviceContext.instanceAction = addAPI
	// If service is updated, identify the action based on the existing instance
	if serviceBody.serviceContext.serviceAction == updateAPI && serviceBody.serviceContext.previousRevision != nil {
		// Get instances for the existing revision and use the latest one as last reference
		queryParams := map[string]string{
			"query": "metadata.references.name==" + serviceBody.serviceContext.previousRevision.Name,
			"sort":  "metadata.audit.createTimestamp,DESC",
		}

		instances, err := c.GetAPIServiceInstances(queryParams, c.cfg.GetInstancesURL())
		if err != nil {
			return err
		}

		if len(instances) > 0 {
			err = c.updateServiceContext(instances, endpoints, serviceBody)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *ServiceClient) updateServiceContext(instances []*v1alpha1.APIServiceInstance, endpoints []v1alpha1.ApiServiceInstanceSpecEndpoint, serviceBody *ServiceBody) error {
	splitName := strings.Split(instances[0].Name, ".")
	countStr := splitName[len(splitName)-1]
	instanceCount, err := strconv.Atoi(countStr)
	if err != nil {
		return fmt.Errorf("failed to convert instance count to an int: %s", err)
	}
	serviceBody.serviceContext.instanceCount = instanceCount
	serviceBody.serviceContext.previousInstance = instances[0]
	// if the endpoints are same update the current instance otherwise create new instance
	if c.compareEndpoints(endpoints, serviceBody.serviceContext.previousInstance.Spec.Endpoint) {
		serviceBody.serviceContext.instanceAction = updateAPI
	}

	return nil
}

func (c *ServiceClient) compareEndpoints(endPointsSrc, endPointsTarget []v1alpha1.ApiServiceInstanceSpecEndpoint) bool {
	if endPointsSrc == nil || endPointsTarget == nil {
		return false
	}
	if len(endPointsSrc) != len(endPointsTarget) {
		return false
	}
	matchedCount := 0
	for _, epSrc := range endPointsSrc {
		itemMatched := false
		for _, epTarget := range endPointsTarget {
			itemMatched = c.compareEndpoint(epSrc, epTarget)
			if itemMatched {
				break
			}
		}
		if !itemMatched {
			break
		}
		matchedCount++
	}
	return matchedCount == len(endPointsSrc)
}

func (c *ServiceClient) compareEndpoint(endPointSrc, endPointTarget v1alpha1.ApiServiceInstanceSpecEndpoint) bool {
	return endPointSrc.Host == endPointTarget.Host &&
		endPointSrc.Port == endPointTarget.Port &&
		endPointSrc.Protocol == endPointTarget.Protocol &&
		endPointSrc.Routing.BasePath == endPointTarget.Routing.BasePath
}

// getAPIServiceInstanceByName - Returns the API service instance for specified name
func (c *ServiceClient) getAPIServiceInstanceByName(instanceName string) (*v1alpha1.APIServiceInstance, error) {
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
