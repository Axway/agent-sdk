package agent

import (
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
)

// credential request definitions

// createOrUpdateCredentialRequestDefinition -
func createOrUpdateCredentialRequestDefinition(data *v1alpha1.CredentialRequestDefinition) (*v1alpha1.CredentialRequestDefinition, error) {
	if !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil, nil
	}
	crdRI, _ := agent.cacheManager.GetCredentialRequestDefinitionByName(data.Name)
	if crdRI == nil {
		return agent.apicClient.RegisterCredentialRequestDefinition(data, false)
	}
	if data.SubResources[definitions.AttrSpecHash] != crdRI.SubResources[definitions.AttrSpecHash] {
		return agent.apicClient.RegisterCredentialRequestDefinition(data, true)
	}
	err := data.FromInstance(crdRI)
	if err != nil {
		return nil, err
	}
	return data, nil
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
				SetName("secret").
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
				SetName("public-key").
				SetLabel("Public Key").
				SetRequired().
				IsString())
	}
}

// NewAPIKeyCredentialRequestBuilder - add api key base properties for provisioning schema
func NewAPIKeyCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	if _, err := agent.cacheManager.GetAccessRequestDefinitionByName(provisioning.APIKeyARD); err != nil {
		NewAccessRequestBuilder().SetName(provisioning.APIKeyARD).Register()
	}

	apiKeyOptions := []func(*crdBuilderOptions){
		withCRDName(provisioning.APIKeyCRD),
		WithCRDProvisionSchemaProperty(
			provisioning.NewSchemaPropertyBuilder().
				SetName("key").
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
				SetName("client-id").
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
	if !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil, nil
	}
	ardRI, _ := agent.cacheManager.GetAccessRequestDefinitionByName(data.Name)
	if ardRI == nil {
		return agent.apicClient.RegisterAccessRequestDefinition(data, false)
	}
	if data.SubResources[definitions.AttrSpecHash] != ardRI.SubResources[definitions.AttrSpecHash] {
		return agent.apicClient.RegisterAccessRequestDefinition(data, true)
	}
	err := data.FromInstance(ardRI)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// NewAccessRequestBuilder - called by the agents to build and register a new access request definition
func NewAccessRequestBuilder() provisioning.AccessRequestBuilder {
	return provisioning.NewAccessRequestBuilder(createOrUpdateAccessRequestDefinition)
}

// provisioner

// RegisterProvisioner - allow the agent to register a provisioner
func RegisterProvisioner(provisioner provisioning.Provisioning) {
	if !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
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
