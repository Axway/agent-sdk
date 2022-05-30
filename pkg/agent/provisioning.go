package agent

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// credential request definitions
// createOrUpdateDefinition -
func createOrUpdateDefinition(data v1.Interface, marketplaceMigration migrate.Migrator) (*v1.ResourceInstance, error) {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil, nil
	}

	runMarketplaceMigration := willCreateOrUpdateResource(data)

	ri, err := agent.apicClient.CreateOrUpdateResource(data)
	if err != nil {
		return nil, err
	}

	if marketplaceMigration != nil && runMarketplaceMigration {
		_, err = migrateMarketPlace(marketplaceMigration, ri)
	}

	return ri, nil
}

// willCreateOrUpdateResource - future check to see if CreateOrUpdateResource will be executed
func willCreateOrUpdateResource(data v1.Interface) bool {

	// Check (only) credential request definition to see if it exists prior to CreateOrUpdateResource call
	ri, err := data.AsInstance()
	if err != nil {
		return false
	}

	if mv1a.CredentialRequestDefinitionGVK().Kind == ri.Kind {
		existingCRD, _ := agent.cacheManager.GetCredentialRequestDefinitionByName(ri.Name)
		if existingCRD == nil {
			log.Debugf("credential request definition %s needs to be created or updated using migration path", ri.Name)
			return true
		}
	} else {
		existingARD, _ := agent.cacheManager.GetAccessRequestDefinitionByName(ri.Name)
		if existingARD == nil {
			log.Debugf("access request definition %s needs to be created or updated using migration path", ri.Name)
			return true
		}
	}

	return false
}

// migrateMarketPlace -
func migrateMarketPlace(marketplaceMigration migrate.Migrator, ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	switch ri.Kind {
	case mv1a.AccessRequestDefinitionGVK().Kind:
		agent.cacheManager.AddAccessRequestDefinition(ri)
	case mv1a.CredentialRequestDefinitionGVK().Kind:
		agent.cacheManager.AddCredentialRequestDefinition(ri)
	}

	apiSvcResources := make([]*v1.ResourceInstance, 0)

	cache := agent.cacheManager.GetAPIServiceCache()

	for _, key := range cache.GetKeys() {
		item, _ := cache.Get(key)
		if item == nil {
			continue
		}

		svc, ok := item.(*v1.ResourceInstance)
		if ok {
			apiSvcResources = append(apiSvcResources, svc)
		}
	}

	for _, svc := range apiSvcResources {
		var err error
		log.Debugf("update apiserviceinstances with request definition %s: %s", ri.Kind, ri.Name)
		_, err = marketplaceMigration.Migrate(svc)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate service: %s", err)
		}
	}
	return ri, nil
}

// createOrUpdateCredentialRequestDefinition -
func createOrUpdateCredentialRequestDefinition(data *v1alpha1.CredentialRequestDefinition) (*v1alpha1.CredentialRequestDefinition, error) {
	ri, err := createOrUpdateDefinition(data, agent.marketplaceMigration)
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

// WithName - set that the name for the CRD
func WithName(name string) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.name = name
	}
}

// WithCRDOAuthSecret - set that the Oauth cred is secret based
func WithCRDOAuthSecret() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		if c.name == "" {
			c.name = provisioning.OAuthSecretCRD
		}
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
		if c.name == "" {
			c.name = provisioning.OAuthPublicKeyCRD
		}

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
	ri, err := createOrUpdateDefinition(data, agent.marketplaceMigration)
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

	if agent.cfg.GetAgentType() == config.DiscoveryAgent || agent.cfg.GetAgentType() == config.GovernanceAgent {
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
}
