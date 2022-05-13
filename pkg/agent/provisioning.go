package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// credential request definitions
// createOrUpdateDefinition -
func createOrUpdateDefinition(data v1.Interface) (*v1.ResourceInstance, error) {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil, nil
	}
	ri, err := agent.apicClient.CreateOrUpdateResource(data)
	if err != nil {
		return nil, err
	}

	if ri.Kind == mv1a.CredentialRequestDefinitionGVK().Kind {
		resources := make([]*v1.ResourceInstance, 0)
		cache := agent.cacheManager.GetAPIServiceCache()

		for _, key := range cache.GetKeys() {
			item, _ := cache.Get(key)
			if item == nil {
				continue
			}

			apiSvc, ok := item.(*v1.ResourceInstance)
			if ok {
				resources = append(resources, apiSvc)
			}
		}

		for _, svcInst := range resources {
			var err error
			_, err = migrateDefinitions(svcInst, ri, agent)
			if err != nil {
				return nil, fmt.Errorf("failed to migrate service: %s", err)
			}

		}
	}

	return ri, nil
}

// createOrUpdateCredentialRequestDefinition -
func createOrUpdateCredentialRequestDefinition(data *v1alpha1.CredentialRequestDefinition) (*v1alpha1.CredentialRequestDefinition, error) {
	ri, err := createOrUpdateDefinition(data)
	if ri == nil || err != nil {
		return nil, err
	}
	err = data.FromInstance(ri)
	return data, err
}

type crdBuilderOptions struct {
	name      string
	provProps []provisioning.PropertyBuilder
	reqProps  []provisioning.PropertyBuilder
}

// NewCredentialRequestBuilder - called by the agents to build and register a new credential reqest definition
func NewCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	thisCred := &crdBuilderOptions{
		provProps: make([]provisioning.PropertyBuilder, 0),
		reqProps:  make([]provisioning.PropertyBuilder, 0),
	}
	for _, o := range options {
		o(thisCred)
	}

	provSchema := provisioning.NewSchemaBuilder()
	for _, provProp := range thisCred.provProps {
		provSchema.AddProperty(provProp)
	}

	reqSchema := provisioning.NewSchemaBuilder()
	for _, props := range thisCred.reqProps {
		reqSchema.AddProperty(props)
	}

	return provisioning.NewCRDBuilder(createOrUpdateCredentialRequestDefinition).
		SetName(thisCred.name).
		SetProvisionSchema(provSchema).
		SetRequestSchema(reqSchema)
}

// withCRDName - set another name for the CRD
func withCRDName(name string) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.name = name
	}
}

// WithCRDProvisionSchemaProperty - add more provisioning properties
func WithCRDProvisionSchemaProperty(prop provisioning.PropertyBuilder) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.provProps = append(c.provProps, prop)
	}
}

// WithCRDRequestSchemaProperty - add more request properties
func WithCRDRequestSchemaProperty(prop provisioning.PropertyBuilder) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.reqProps = append(c.reqProps, prop)
	}
}

// WithCRDOAuthSecret - set that the Oauth cred is secret based
func WithCRDOAuthSecret() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.name = provisioning.OAuthSecretCRD
		c.provProps = append(c.provProps,
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.OauthClientSecret).
				SetLabel("Client Secret").
				SetRequired().
				IsString().
				IsEncrypted())
	}
}

// WithCRDOAuthPublicKey - set that the Oauth cred is key based
func WithCRDOAuthPublicKey() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.name = provisioning.OAuthPublicKeyCRD
		c.reqProps = append(c.reqProps,
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.OauthPublicKey).
				SetLabel("Public Key").
				SetRequired().
				IsString())
	}
}

// NewAPIKeyCredentialRequestBuilder - add api key base properties for provisioning schema
func NewAPIKeyCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	apiKeyOptions := []func(*crdBuilderOptions){
		withCRDName(provisioning.APIKeyCRD),
		WithCRDProvisionSchemaProperty(
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.APIKey).
				SetLabel("API Key").
				SetRequired().
				IsString().
				IsEncrypted()),
	}

	apiKeyOptions = append(apiKeyOptions, options...)

	return NewCredentialRequestBuilder(apiKeyOptions...)
}

// NewOAuthCredentialRequestBuilder - add oauth base properties for provisioning schema
func NewOAuthCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	oauthOptions := []func(*crdBuilderOptions){
		WithCRDProvisionSchemaProperty(
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.OauthClientID).
				SetLabel("Client ID").
				SetRequired().
				IsString()),
	}

	oauthOptions = append(oauthOptions, options...)

	return NewCredentialRequestBuilder(oauthOptions...)
}

// access request definitions

// createOrUpdateAccessRequestDefinition -
func createOrUpdateAccessRequestDefinition(data *v1alpha1.AccessRequestDefinition) (*v1alpha1.AccessRequestDefinition, error) {
	ri, err := createOrUpdateDefinition(data)
	if ri == nil || err != nil {
		return nil, err
	}
	err = data.FromInstance(ri)
	return data, err
}

// NewAccessRequestBuilder - called by the agents to build and register a new access request definition
func NewAccessRequestBuilder() provisioning.AccessRequestBuilder {
	return provisioning.NewAccessRequestBuilder(createOrUpdateAccessRequestDefinition)
}

// NewAPIKeyAccessRequestBuilder - called by the agents
func NewAPIKeyAccessRequestBuilder() provisioning.AccessRequestBuilder {
	return NewAccessRequestBuilder().SetName(provisioning.APIKeyARD)
}

// provisioner

// RegisterProvisioner - allow the agent to register a provisioner
func RegisterProvisioner(provisioner provisioning.Provisioning) {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return
	}
	agent.provisioner = provisioner
	agent.proxyResourceHandler.RegisterTargetHandler(
		"accessrequesthandler",
		handler.NewAccessRequestHandler(agent.provisioner, agent.cacheManager, agent.apicClient),
	)
	agent.proxyResourceHandler.RegisterTargetHandler(
		"managedappHandler",
		handler.NewManagedApplicationHandler(agent.provisioner, agent.cacheManager, agent.apicClient),
	)
	agent.proxyResourceHandler.RegisterTargetHandler(
		"credentialHandler",
		handler.NewCredentialHandler(agent.provisioner, agent.apicClient),
	)
}

type migrateFunc func(svcInst, requestDefinition *v1.ResourceInstance, agent agentData) error

func migrateDefinitions(svcInst, requestDefinition *v1.ResourceInstance, agent agentData) (*v1.ResourceInstance, error) {
	if svcInst.Kind != mv1a.APIServiceGVK().Kind {
		return svcInst, fmt.Errorf("expected resource instance kind to be api service")
	}

	funcs := []migrateFunc{
		updateInst,
	}

	errCh := make(chan error, len(funcs))
	wg := &sync.WaitGroup{}

	for _, f := range funcs {
		wg.Add(1)

		go func(fun migrateFunc) {
			defer wg.Done()

			err := fun(svcInst, requestDefinition, agent)
			errCh <- err
		}(f)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return svcInst, e
		}
	}

	log.Debugf("finished migrating marketplace provisioning for service: %s", svcInst.Name)

	return svcInst, nil
}

// updateInst gets a list of instances for the service and updates their request definitions.
func updateInst(svcInst, requestDefinition *v1.ResourceInstance, agent agentData) error {
	revURL := agent.cfg.GetRevisionsURL()

	q := map[string]string{}

	revs, err := agent.apicClient.GetAPIV1ResourceInstancesWithPageSize(q, revURL, 100)
	if err != nil {
		return err
	}

	errCh := make(chan error, len(revs))
	wg := &sync.WaitGroup{}

	// query for api service instances by reference to a revision name
	for _, rev := range revs {
		wg.Add(1)

		go func(r *v1.ResourceInstance) {
			defer wg.Done()

			specDefintionType := r.Spec["definition"].(map[string]interface{})["type"]
			specDefinitionValue := r.Spec["definition"].(map[string]interface{})["value"].(string)

			q := map[string]string{
				"query": queryFunc(r.Name),
			}
			url := agent.cfg.GetInstancesURL()
			err := updateInstResources(requestDefinition, url, q, specDefinitionValue, specDefintionType.(string))
			errCh <- err
		}(rev)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return e
		}
	}

	return nil
}

func updateInstResources(requestDefinition *v1.ResourceInstance, resourceURL string, query map[string]string, specDefinitionValue, specDefintionType string) error {
	resources, err := agent.apicClient.GetAPIV1ResourceInstancesWithPageSize(query, resourceURL, 100)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error, len(resources))

	for _, resource := range resources {
		wg.Add(1)

		go func(ri *v1.ResourceInstance) {
			defer wg.Done()

			apiSvcInst := mv1a.NewAPIServiceInstance(ri.Name, ri.Metadata.Scope.Name)
			apiSvcInst.FromInstance(ri)

			specDefinition, _ := base64.StdEncoding.DecodeString(specDefinitionValue)

			specParser := apic.NewSpecResourceParser(specDefinition, specDefintionType)
			err := specParser.Parse()
			if err != nil {
				errCh <- err
				return
			}

			specProcessor := specParser.GetSpecProcessor()
			endPoints, err := specProcessor.GetEndpoints()
			instanceSpecEndPoints, err := createInstanceEndpoint(endPoints)
			if err != nil {
				errCh <- err
				return
			}

			var ardRIName string
			var credentialRequestPolicies []string

			var i interface{} = specProcessor

			if val, ok := i.(apic.OasSpecProcessor); ok {
				val.ParseAuthInfo()

				// get the auth policy from the spec
				authPolicies := val.GetAuthPolicies()

				// get the apikey info
				apiKeyInfo := val.GetAPIKeyInfo()

				// get oauth scopes
				oauthScopes := val.GetOAuthScopes()

				if requestDefinition.Kind == mv1a.AccessRequestDefinitionGVK().Kind {
					// Check if ARD exists
					if apiSvcInst.Spec.AccessRequestDefinition == "" {
						log.Debugf("apiservice %s does not have any access request definitions", apiSvcInst.Name)
						ardRIName, err = migrateAccessRequestDefinitions(apiKeyInfo, oauthScopes, ri)
						if err != nil {
							errCh <- err
							return
						}

						if ardRIName == "" {
							ardRIName = "api-key"
						}

						log.Debugf("adding the following access request definition %s", ardRIName)
					}
				} else if requestDefinition.Kind == mv1a.CredentialRequestDefinitionGVK().Kind {
					// Check if CRD exists
					if len(apiSvcInst.Spec.CredentialRequestDefinitions) == 0 {
						log.Debugf("apiservice %s does not have any credential request definitions", apiSvcInst.Name)
						credentialRequestPolicies, err = migrateCredentialRequestDefinitions(authPolicies, ri)

						// Find only the known CRD's

						log.Debugf("check to see if credential request definitions exist for %s", credentialRequestPolicies)
						// remove any crd not in the cache
						knownCRDs := make([]string, 0)
						for _, credentialRequestPolicy := range credentialRequestPolicies {
							if def, err := agent.cacheManager.GetCredentialRequestDefinitionByName(credentialRequestPolicy); err == nil && def != nil {
								knownCRDs = append(knownCRDs, credentialRequestPolicy)
							}
						}

						if len(knownCRDs) > 0 {
							log.Debugf("attempt to add the following credential request definitions %s, to apiservice %s", knownCRDs, apiSvcInst.Name)
						} else {
							log.Debug("did not find any credential request definitions in cache")
							return
						}
					}
				}

				newSpec := mv1a.ApiServiceInstanceSpec{
					Endpoint:                     instanceSpecEndPoints,
					ApiServiceRevision:           ri.Name,
					CredentialRequestDefinitions: credentialRequestPolicies,
					AccessRequestDefinition:      ardRIName,
				}

				// convert to set ri.Spec
				var inInterface map[string]interface{}
				in, _ := json.Marshal(newSpec)
				json.Unmarshal(in, &inInterface)

				ri.Spec = inInterface
			}

			url := fmt.Sprintf("%s/%s", resourceURL, ri.Name)
			err = updateRI(url, ri)
			errCh <- err
		}(resource)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return e
		}
	}

	return nil

}

func migrateCredentialRequestDefinitions(authPolicies []string, ri *v1.ResourceInstance) ([]string, error) {
	var credentialRequestPolicies []string

	for _, policy := range authPolicies {
		if policy == apic.Apikey {
			credentialRequestPolicies = append(credentialRequestPolicies, provisioning.APIKeyCRD)
		}
		if policy == apic.Oauth {
			credentialRequestPolicies = append(credentialRequestPolicies, []string{provisioning.OAuthPublicKeyCRD, provisioning.OAuthSecretCRD}...)
		}
	}

	return credentialRequestPolicies, nil

}

func migrateAccessRequestDefinitions(apiKeyInfo []apic.APIKeyInfo, oauthScopes map[string]string, ri *v1.ResourceInstance) (string, error) {

	scopes := make([]string, 0)
	for scope := range oauthScopes {
		scopes = append(scopes, scope)
	}

	if len(scopes) > 0 {
		ardRI, err := provisioning.NewAccessRequestBuilder(setAccessRequestDefintion).
			SetSchema(
				provisioning.NewSchemaBuilder().
					AddProperty(
						provisioning.NewSchemaPropertyBuilder().
							SetName("scopes").
							SetLabel("Scopes").
							IsArray().
							AddItem(
								provisioning.NewSchemaPropertyBuilder().
									SetName("scope").
									IsString().
									SetEnumValues(scopes)))).Register()
		if err != nil {
			return "", err
		}
		return ardRI.Name, nil
	}
	return "", nil
}

func setAccessRequestDefintion(accessRequestDefinition *mv1a.AccessRequestDefinition) (*mv1a.AccessRequestDefinition, error) {
	accessRequestDefinition = accessRequestDefinition
	return accessRequestDefinition, nil
}

// updateRI updates the resource, and the sub resource
func updateRI(url string, ri *v1.ResourceInstance) error {
	_, err := agent.apicClient.UpdateResourceInstance(ri)
	if err != nil {
		return err
	}

	return nil
}

func createInstanceEndpoint(endpoints []apic.EndpointDefinition) ([]mv1a.ApiServiceInstanceSpecEndpoint, error) {
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

func queryFunc(name string) string {
	return fmt.Sprintf("metadata.references.name==%s", name)
}

func checkCredentialRequestDefinitions(credentialRequestPolicies []string) []string {
	log.Debugf("check to see if credential request definitions exist for %s", credentialRequestPolicies)
	// remove any crd not in the cache
	knownCRDs := make([]string, 0)
	for _, credentialRequestPolicy := range credentialRequestPolicies {
		if def, err := agent.cacheManager.GetCredentialRequestDefinitionByName(credentialRequestPolicy); err == nil && def != nil {
			knownCRDs = append(knownCRDs, credentialRequestPolicy)
		}
	}

	return knownCRDs
}
