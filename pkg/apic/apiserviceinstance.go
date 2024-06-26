package apic

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/util"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func buildAPIServiceInstanceSpec(
	serviceBody *ServiceBody,
	endpoints []management.ApiServiceInstanceSpecEndpoint,
) management.ApiServiceInstanceSpec {
	return management.ApiServiceInstanceSpec{
		ApiServiceRevision: serviceBody.serviceContext.revisionName,
		Endpoint:           endpoints,
	}
}

func buildAPIServiceInstanceMarketplaceSpec(
	serviceBody *ServiceBody,
	endpoints []management.ApiServiceInstanceSpecEndpoint,
	knownCRDs []string,
) management.ApiServiceInstanceSpec {
	return management.ApiServiceInstanceSpec{
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
	// Check if request definitions are allowed. False would indicate the service is Unpublished
	if serviceBody.requestDefinitionsAllowed {
		for _, crd := range crds {
			if def, err := c.caches.GetCredentialRequestDefinitionByName(crd); err == nil && def != nil {
				knownCRDs = append(knownCRDs, crd)
			}
		}
	} else {
		log.Warnf("removed existing credential request definitions for instance %s. Contact your system administrator for further assistance", serviceBody.APIName)
	}

	return knownCRDs
}

func (c *ServiceClient) checkAccessRequestDefinition(serviceBody *ServiceBody) {
	ard := serviceBody.ardName

	// Check if request definitions are allowed. False would indicate the service is Unpublished
	if serviceBody.requestDefinitionsAllowed {
		if def, err := c.caches.GetAccessRequestDefinitionByName(ard); err == nil && def != nil {
			return
		}
	} else {
		log.Warnf("removed existing access request definitions for instance %s. Contact your system administrator for further assistance", serviceBody.APIName)
	}

	serviceBody.ardName = ""
}

func (c *ServiceClient) buildAPIServiceInstance(
	serviceBody *ServiceBody,
	name string,
	endpoints []management.ApiServiceInstanceSpecEndpoint,
) *management.APIServiceInstance {

	spec := buildAPIServiceInstanceSpec(serviceBody, endpoints)
	if c.cfg.IsMarketplaceSubsEnabled() {
		c.checkAccessRequestDefinition(serviceBody)
		spec = buildAPIServiceInstanceMarketplaceSpec(serviceBody, endpoints, c.checkCredentialRequestDefinitions(serviceBody))
	}

	owner, _ := c.getOwnerObject(serviceBody, false) // owner, _ := at this point, we don't need to validate error on getOwnerObject.  This is used for subresource status update
	instance := &management.APIServiceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.APIServiceInstanceGVK(),
			Name:             name,
			Title:            serviceBody.NameToPush,
			Attributes:       util.CheckEmptyMapStringString(serviceBody.InstanceAttributes),
			Tags:             mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish()),
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Kind: management.EnvironmentGVK().Kind,
					Name: c.cfg.GetEnvironmentName(),
				},
			},
		},
		Spec:  spec,
		Owner: owner,
	}
	buildAPIServiceInstanceSourceSubResource(instance, serviceBody)

	instDetails := util.MergeMapStringInterface(serviceBody.ServiceAgentDetails, serviceBody.InstanceAgentDetails)
	details := buildAgentDetailsSubResource(serviceBody, false, instDetails)
	util.SetAgentDetails(instance, details)

	return instance
}

func (c *ServiceClient) updateAPIServiceInstance(
	serviceBody *ServiceBody,
	instance *management.APIServiceInstance,
	endpoints []management.ApiServiceInstanceSpecEndpoint,
) *management.APIServiceInstance {
	owner, _ := c.getOwnerObject(serviceBody, false)
	instance.GroupVersionKind = management.APIServiceInstanceGVK()
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
	buildAPIServiceInstanceSourceSubResource(instance, serviceBody)

	details := util.MergeMapStringInterface(serviceBody.ServiceAgentDetails, serviceBody.InstanceAgentDetails)
	util.SetAgentDetails(instance, buildAgentDetailsSubResource(serviceBody, false, details))

	return instance
}

func buildAPIServiceInstanceSourceSubResource(instance *management.APIServiceInstance, serviceBody *ServiceBody) *management.ApiServiceInstanceSource {
	serviceBody.serviceContext.updateInstanceSource = false

	source := instance.Source
	if source == nil {
		instance.Source = &management.ApiServiceInstanceSource{}
		source = instance.Source
	}

	dataplaneType := serviceBody.GetDataplaneType()
	if dataplaneType != "" {
		if source.DataplaneType == nil {
			source.DataplaneType = &management.ApiServiceInstanceSourceDataplaneType{}
		}
		if serviceBody.IsDesignDataplane() {
			if source.DataplaneType.Design != dataplaneType.String() {
				source.DataplaneType.Design = dataplaneType.String()
				serviceBody.serviceContext.updateInstanceSource = true
			}
		} else if source.DataplaneType.Managed != dataplaneType.String() {
			source.DataplaneType.Managed = dataplaneType.String()
			serviceBody.serviceContext.updateInstanceSource = true
		}
	}

	referencedInstance := serviceBody.GetReferenceInstanceName()
	if referencedInstance != "" {
		if source.References == nil {
			source.References = &management.ApiServiceInstanceSourceReferences{}
		}
		if source.References.ApiServiceInstance != referencedInstance {
			source.References.ApiServiceInstance = serviceBody.GetReferenceInstanceName()
			serviceBody.serviceContext.updateInstanceSource = true
		}
	}
	return nil
}

// processInstance - Creates or updates an API Service Instance based on the current API Service Revision.
func (c *ServiceClient) processInstance(serviceBody *ServiceBody) error {
	endpoints, err := createInstanceEndpoint(serviceBody.Endpoints)
	if err != nil {
		return err
	}

	// creating new instance
	instance := c.buildAPIServiceInstance(serviceBody, getRevisionPrefix(serviceBody), endpoints)

	if serviceBody.serviceContext.serviceAction == updateAPI {
		prevInst, err := c.getInstance(serviceBody, c.createAPIServerURL(instance.GetKindLink()))
		if err != nil {
			return err
		}

		if prevInst != nil {
			// updating existing instance
			instance = c.updateAPIServiceInstance(serviceBody, prevInst, endpoints)
		}
	}

	addSpecHashToResource(instance)

	ri, err := c.CreateOrUpdateResource(instance)
	if err == nil {
		err = c.updateAPIServiceInstanceSubresources(ri, instance, serviceBody)
	}

	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, rollbackErr := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				return errors.New(err.Error() + rollbackErr.Error())
			}
		}
		return err
	}

	c.caches.AddAPIServiceInstance(ri)
	serviceBody.serviceContext.instanceName = instance.Name

	return err
}

func (c *ServiceClient) updateAPIServiceInstanceSubresources(ri apiv1.Interface, instance *management.APIServiceInstance, serviceBody *ServiceBody) error {
	subResources := make(map[string]interface{})
	if serviceBody.serviceContext.updateInstanceSource && instance.Source != nil {
		subResources[management.ApiServiceInstanceSourceSubResourceName] = instance.Source
	}

	if len(subResources) > 0 {
		inst, _ := ri.AsInstance()
		return c.CreateSubResource(inst.ResourceMeta, subResources)
	}
	return nil
}

func createInstanceEndpoint(endpoints []EndpointDefinition) ([]management.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := make([]management.ApiServiceInstanceSpecEndpoint, 0)

	// To set your own endpoints call AddServiceEndpoint/SetServiceEndpoint on the ServiceBodyBuilder.
	// Any endpoints provided from the ServiceBodyBuilder will override the endpoints found in the spec.
	if len(endpoints) > 0 {
		for _, endpointDef := range endpoints {
			endPoints = append(endPoints, management.ApiServiceInstanceSpecEndpoint{
				Host:     endpointDef.Host,
				Port:     endpointDef.Port,
				Protocol: endpointDef.Protocol,
				Routing: management.ApiServiceInstanceSpecRouting{
					BasePath: endpointDef.BasePath,
					Details:  endpointDef.Details,
				},
			})
		}
	} else {
		log.Debug("Processing API service instance with no endpoint")
	}

	return endPoints, nil
}

func (c *ServiceClient) getInstance(serviceBody *ServiceBody, url string) (*management.APIServiceInstance, error) {
	queryParams := map[string]string{
		"query": "metadata.references.name==" + serviceBody.serviceContext.revisionName,
	}
	instances, err := c.GetAPIServiceInstances(queryParams, url)
	if err != nil {
		return nil, err
	}
	if len(instances) == 1 {
		// return only instance
		return instances[0], nil
	}

	// check the instance for the stage agent details
	for _, i := range instances {
		stage, err := util.GetAgentDetailsValue(i, defs.AttrExternalAPIStage)
		if err != nil {
			continue
		}
		if stage == serviceBody.Stage {
			// found the stage match
			return i, nil
		}
	}

	// if no instance found
	return nil, nil
}

// GetAPIServiceInstanceByName - Returns the API service instance for specified name
func (c *ServiceClient) GetAPIServiceInstanceByName(name string) (*management.APIServiceInstance, error) {
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
	apiInstance := new(management.APIServiceInstance)
	err = json.Unmarshal(response.Body, apiInstance)
	return apiInstance, err
}
