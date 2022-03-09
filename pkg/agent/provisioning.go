package agent

import (
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
)

// credential request definitions

// createOrUpdateCredentialRequestDefinition -
func createOrUpdateCredentialRequestDefinition(data *v1alpha1.CredentialRequestDefinition) (*v1alpha1.CredentialRequestDefinition, error) {
	// TODO - check cache for credential request, update if needed
	return agent.apicClient.RegisterCredentialRequestDefinition(data, true)
}

// NewCredentialRequestBuilder - called by the agents to build and register a new credential reqest definition
func NewCredentialRequestBuilder() provisioning.CredentialRequestBuilder {
	return provisioning.NewCRDBuilder(createOrUpdateCredentialRequestDefinition)
}

// NewAPIKeyCredentialRequestBuilder - add api key base properties for provisioning schema
func NewAPIKeyCredentialRequestBuilder() provisioning.CredentialRequestBuilder {
	NewAccessRequestBuilder().SetName("api-key").Register()
	return NewCredentialRequestBuilder().
		SetName("api-key").
		SetProvisionSchema(provisioning.NewSchemaBuilder().
			AddProperty(
				provisioning.NewSchemaPropertyBuilder().
					SetName("key").
					SetLabel("API Key").
					SetRequired().
					IsString().
					IsEncrypted()))
}

// NewOAuthCredentialRequestBuilder - add oauth base properties for provisioning schema
func NewOAuthCredentialRequestBuilder() provisioning.CredentialRequestBuilder {
	return NewCredentialRequestBuilder().
		SetName("oauth").
		SetProvisionSchema(provisioning.NewSchemaBuilder().
			AddProperty(
				provisioning.NewSchemaPropertyBuilder().
					SetName("id").
					SetLabel("Client ID").
					SetRequired().
					IsString()).
			AddProperty(
				provisioning.NewSchemaPropertyBuilder().
					SetName("secret").
					SetLabel("Client Secret").
					SetRequired().
					IsString().
					IsEncrypted()))
}

// access request definitions

// createOrUpdateAccessRequestDefinition -
func createOrUpdateAccessRequestDefinition(data *v1alpha1.AccessRequestDefinition) (*v1alpha1.AccessRequestDefinition, error) {
	// TODO - check cache for access request, update if needed
	return agent.apicClient.RegisterAccessRequestDefinition(data, true)
}

// NewAccessRequestBuilder - called by the agents to build and register a new access reqest definition
func NewAccessRequestBuilder() provisioning.AccessRequestBuilder {
	return provisioning.NewAccessRequestBuilder(createOrUpdateAccessRequestDefinition)
}

func RegisterProvisioner(provisioner provisioning.Provisioning) {
	agent.provisioner = provisioner
	agent.proxyResourceHandler.RegisterTargetHandler("accessrequesthandler",
		handler.NewAccessRequestHandler(agent.provisioner, agent.cacheManager, agent.apicClient))
	agent.proxyResourceHandler.RegisterTargetHandler("managedappHandler",
		handler.NewManagedApplicationHandler(agent.provisioner, agent.cacheManager, agent.apicClient))
	agent.proxyResourceHandler.RegisterTargetHandler("credentialHandler",
		handler.NewCredentialHandler(agent.provisioner, agent.apicClient))
}
