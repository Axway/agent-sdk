package agent

import (
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
)

const (
	apikeyARD = "api-key"
	apikeyCRD = "api-key"
	oauthCRD  = "oauth"
)

// credential request definitions

// createOrUpdateCredentialRequestDefinition -
func createOrUpdateCredentialRequestDefinition(data *v1alpha1.CredentialRequestDefinition) (*v1alpha1.CredentialRequestDefinition, error) {
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

// NewCredentialRequestBuilder - called by the agents to build and register a new credential reqest definition
func NewCredentialRequestBuilder() provisioning.CredentialRequestBuilder {
	return provisioning.NewCRDBuilder(createOrUpdateCredentialRequestDefinition)
}

// NewAPIKeyCredentialRequestBuilder - add api key base properties for provisioning schema
func NewAPIKeyCredentialRequestBuilder() provisioning.CredentialRequestBuilder {
	if _, err := agent.cacheManager.GetAccessRequestDefinitionByName(apikeyARD); err != nil {
		NewAccessRequestBuilder().SetName(apikeyARD).Register()
	}
	return NewCredentialRequestBuilder().
		SetName(apikeyCRD).
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
		SetName(oauthCRD).
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

// NewAccessRequestBuilder - called by the agents to build and register a new access reqest definition
func NewAccessRequestBuilder() provisioning.AccessRequestBuilder {
	return provisioning.NewAccessRequestBuilder(createOrUpdateAccessRequestDefinition)
}

// RegisterProvisioner - allow the agent to register a provisioner
func RegisterProvisioner(provisioner provisioning.Provisioning) {
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
