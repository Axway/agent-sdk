package provisioning

import (
	"fmt"

	v1alpha1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
)

// RegisterCredentialRequestDefinition - the function signature used when calling the NewCredentialRequestBuilder function
type RegisterCredentialRequestDefinition func(credentialRequestDefinition *v1alpha1.CredentialRequestDefinition) (*v1alpha1.CredentialRequestDefinition, error)

type credentialRequestDef struct {
	name            string
	title           string
	provisionSchema map[string]interface{}
	requestSchema   map[string]interface{}
	webhooks        []string
	registerFunc    RegisterCredentialRequestDefinition
	err             error
}

// CredentialRequestBuilder - aids in creating a new credential request
type CredentialRequestBuilder interface {
	SetName(name string) CredentialRequestBuilder
	SetTitle(title string) CredentialRequestBuilder
	SetRequestSchema(schema SchemaBuilder) CredentialRequestBuilder
	SetProvisionSchema(schema SchemaBuilder) CredentialRequestBuilder
	SetWebhooks(webhooks []string) CredentialRequestBuilder
	AddWebhook(webhook string) CredentialRequestBuilder
	Register() (*v1alpha1.CredentialRequestDefinition, error)
}

// NewCRDBuilder - called by the agent package and sends in the function that registers this credential request
func NewCRDBuilder(registerFunc RegisterCredentialRequestDefinition) CredentialRequestBuilder {
	return &credentialRequestDef{
		webhooks:     make([]string, 0),
		registerFunc: registerFunc,
	}
}

// SetName - set the name of the credential request
func (c *credentialRequestDef) SetName(name string) CredentialRequestBuilder {
	c.name = name
	return c
}

// SetTitle - set the title of the credential request
func (c *credentialRequestDef) SetTitle(title string) CredentialRequestBuilder {
	c.title = title
	return c
}

// SetRequestSchema - set the schema to be used for credential requests
func (c *credentialRequestDef) SetRequestSchema(schema SchemaBuilder) CredentialRequestBuilder {
	if c.err != nil {
		return c
	}

	if schema != nil {
		c.requestSchema, c.err = schema.Build()
	} else {
		c.err = fmt.Errorf("expected a SchemaBuilder argument but received nil")
	}

	return c
}

// SetProvisionSchema - set the schema to be used when provisioning credentials
func (c *credentialRequestDef) SetProvisionSchema(schema SchemaBuilder) CredentialRequestBuilder {
	if c.err != nil {
		return c
	}

	if schema != nil {
		c.provisionSchema, c.err = schema.Build()
	} else {
		c.err = fmt.Errorf("expected a SchemaBuilder argument but received nil")
	}

	return c
}

// SetWebhooks - set a list of webhooks to be invoked when credential of this type created
func (c *credentialRequestDef) SetWebhooks(webhooks []string) CredentialRequestBuilder {
	if webhooks != nil {
		c.webhooks = webhooks
	}
	return c
}

// AddWebhook - add a webhook to the list of webhooks to be invoked when a credential of this type is requested
func (c *credentialRequestDef) AddWebhook(webhook string) CredentialRequestBuilder {
	c.webhooks = append(c.webhooks, webhook)
	return c
}

// Register - create the credential request defintion and send it to Central
func (c *credentialRequestDef) Register() (*v1alpha1.CredentialRequestDefinition, error) {
	if c.err != nil {
		return nil, c.err
	}

	if c.requestSchema == nil {
		c.requestSchema, _ = NewSchemaBuilder().Build()
	}

	spec := v1alpha1.CredentialRequestDefinitionSpec{
		Schema: c.requestSchema,
		Provision: &v1alpha1.CredentialRequestDefinitionSpecProvision{
			Schema: c.provisionSchema,
		},
	}

	hashInt, _ := util.ComputeHash(spec)

	if c.title == "" {
		c.title = c.name
	}

	crd := v1alpha1.NewCredentialRequestDefinition(c.name, "")
	crd.Title = c.title
	crd.Spec = spec

	util.SetAgentDetailsKey(crd, definitions.AttrSpecHash, fmt.Sprintf("%v", hashInt))

	return c.registerFunc(crd)
}
