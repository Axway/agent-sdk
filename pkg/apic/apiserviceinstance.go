package apic

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/util"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func buildAPIServiceInstanceSpec(
	serviceBody *ServiceBody,
	endpoints []mv1a.ApiServiceInstanceSpecEndpoint,
) mv1a.ApiServiceInstanceSpec {
	return mv1a.ApiServiceInstanceSpec{
		ApiServiceRevision: serviceBody.serviceContext.revisionName,
		Endpoint:           endpoints,
	}
}

func buildAPIServiceInstanceMarketplaceSpec(
	serviceBody *ServiceBody,
	endpoints []mv1a.ApiServiceInstanceSpecEndpoint,
	knownCRDs []string,
) mv1a.ApiServiceInstanceSpec {
	return mv1a.ApiServiceInstanceSpec{
		ApiServiceRevision:           serviceBody.serviceContext.revisionName,
		Endpoint:                     endpoints,
		CredentialRequestDefinitions: knownCRDs,
		AccessRequestDefinition:      serviceBody.ardName,
	}
}

func (c *ServiceClient) checkCredentialRequestDefinitions(serviceBody *ServiceBody) []string {
	crds := serviceBody.GetCredentialRequestDefinitions()

	// remove any crd not in the cache
	knownCRDs := make([]string, 0)
	for _, crd := range crds {
		if def, err := c.caches.GetCredentialRequestDefinitionByName(crd); err == nil && def != nil {
			knownCRDs = append(knownCRDs, crd)
		}
	}

	return knownCRDs
}

func (c *ServiceClient) checkAccessRequestDefinition(serviceBody *ServiceBody) {
	ard := serviceBody.ardName

	if def, err := c.caches.GetAccessRequestDefinitionByName(ard); err == nil && def != nil {
		return
	}

	serviceBody.ardName = ""
}

func (c *ServiceClient) buildAPIServiceInstance(
	serviceBody *ServiceBody,
	name string,
	endpoints []mv1a.ApiServiceInstanceSpecEndpoint,
) *mv1a.APIServiceInstance {
	finalizer := make([]v1.Finalizer, 0)
	if serviceBody.uniqueARD {
		finalizer = append(finalizer, v1.Finalizer{
			Name:        AccessRequestDefinitionFinalizer,
			Description: serviceBody.ardName,
		})
	}

	spec := buildAPIServiceInstanceSpec(serviceBody, endpoints)
	if c.cfg.IsMarketplaceSubsEnabled() {
		c.checkAccessRequestDefinition(serviceBody)
		spec = buildAPIServiceInstanceMarketplaceSpec(serviceBody, endpoints, c.checkCredentialRequestDefinitions(serviceBody))
	}

	owner, _ := c.getOwnerObject(serviceBody, false) // owner, _ := at this point, we don't need to validate error on getOwnerObject.  This is used for subresource status update
	instance := &mv1a.APIServiceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1a.APIServiceInstanceGVK(),
			Name:             name,
			Title:            serviceBody.NameToPush,
			Attributes:       util.CheckEmptyMapStringString(serviceBody.InstanceAttributes),
			Tags:             mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish()),
			Finalizers:       finalizer,
			Metadata: v1.Metadata{
				Scope: v1.MetadataScope{
					Kind: mv1a.EnvironmentGVK().Kind,
					Name: c.cfg.GetEnvironmentName(),
				},
			},
		},
		Spec:  spec,
		Owner: owner,
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
	owner, _ := c.getOwnerObject(serviceBody, false)
	instance.GroupVersionKind = mv1a.APIServiceInstanceGVK()
	instance.Metadata.ResourceVersion = ""
	instance.Title = serviceBody.NameToPush
	instance.Attributes = util.CheckEmptyMapStringString(serviceBody.InstanceAttributes)
	instance.Tags = mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish())
	instance.Spec = buildAPIServiceInstanceSpec(serviceBody, endpoints)
	if c.cfg.IsMarketplaceSubsEnabled() {
		c.checkAccessRequestDefinition(serviceBody)
		instance.Spec = buildAPIServiceInstanceMarketplaceSpec(serviceBody, endpoints, c.checkCredentialRequestDefinitions(serviceBody))
	}
	instance.Owner = owner

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
	instanceName := getRevisionPrefix(serviceBody)

	// creating new instance
	httpMethod = http.MethodPost
	instance = c.buildAPIServiceInstance(serviceBody, instanceName, endpoints)

	if serviceBody.serviceContext.serviceAction == updateAPI {
		instances, err := c.getInstances(instanceName, instanceURL)
		if err != nil {
			return err
		}

		if len(instances) > 0 {
			instanceURL = instanceURL + "/" + instanceName

			// updating existing instance
			httpMethod = http.MethodPut
			instance = c.updateAPIServiceInstance(serviceBody, instances[0], endpoints)
		}
	}

	buffer, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	ri, err := c.executeAPIServiceAPI(httpMethod, instanceURL, buffer)
	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, rollbackErr := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				return errors.New(err.Error() + rollbackErr.Error())
			}
		}
		return err
	}

	if err == nil && len(instance.SubResources) > 0 {
		ri.SubResources = instance.SubResources // add the subresources to the instance that will be cached
		if xAgentDetail, ok := instance.SubResources[defs.XAgentDetails]; ok {
			subResources := map[string]interface{}{
				defs.XAgentDetails: xAgentDetail,
			}
			err = c.CreateSubResource(instance.ResourceMeta, subResources)
			if err != nil {
				_, rollbackErr := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
				if rollbackErr != nil {
					return errors.New(err.Error() + rollbackErr.Error())
				}
			}
		}
	}

	c.caches.AddAPIServiceInstance(ri)
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

func (c *ServiceClient) getInstances(name, url string) ([]*mv1a.APIServiceInstance, error) {
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
