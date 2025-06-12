package provisioning

import (
	"fmt"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
)

// RegisterCredentialRequestDefinition - the function signature used when calling the NewCredentialRequestBuilder function
type RegisterCredentialRequestDefinition func(credentialRequestDefinition *management.CredentialRequestDefinition) (*management.CredentialRequestDefinition, error)

type credentialRequestDef struct {
	name            string
	title           string
	provisionSchema map[string]interface{}
	requestSchema   map[string]interface{}
	webhooks        []string
	actions         []string
	registerFunc    RegisterCredentialRequestDefinition
	err             error
	agentDetails    map[string]interface{}
	renewable       bool
	suspendable     bool
	period          int
	credType        string
}

// CredentialRequestBuilder - aids in creating a new credential request
type CredentialRequestBuilder interface {
	SetName(name string) CredentialRequestBuilder
	SetTitle(title string) CredentialRequestBuilder
	SetRequestSchema(schema SchemaBuilder) CredentialRequestBuilder
	SetProvisionSchema(schema SchemaBuilder) CredentialRequestBuilder
	SetWebhooks(webhooks []string) CredentialRequestBuilder
	AddWebhook(webhook string) CredentialRequestBuilder
	AddXAgentDetails(key string, value interface{}) CredentialRequestBuilder
	IsRenewable() CredentialRequestBuilder
	IsSuspendable() CredentialRequestBuilder
	SetExpirationDays(days int) CredentialRequestBuilder
	SetDeprovisionExpired() CredentialRequestBuilder
	SetType(crdType string) CredentialRequestBuilder
	Register() (*management.CredentialRequestDefinition, error)
}

// NewCRDBuilder - called by the agent package and sends in the function that registers this credential request
func NewCRDBuilder(registerFunc RegisterCredentialRequestDefinition) CredentialRequestBuilder {
	return &credentialRequestDef{
		webhooks:     make([]string, 0),
		registerFunc: registerFunc,
		actions:      make([]string, 0),
		agentDetails: map[string]interface{}{},
	}
}

// AddXAgentDetails - adds a key value pair to x-agent-details
func (c *credentialRequestDef) AddXAgentDetails(key string, value interface{}) CredentialRequestBuilder {
	c.agentDetails[key] = value
	return c
}

// SetName - set the name of the credential request
func (c *credentialRequestDef) SetName(name string) CredentialRequestBuilder {
	c.name = util.NormalizeNameForCentral(name)
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

// IsRenewable - the credential can be asked to be renewed
func (c *credentialRequestDef) IsRenewable() CredentialRequestBuilder {
	c.renewable = true
	return c
}

// IsSuspendable - the credential can be asked to be suspended
func (c *credentialRequestDef) IsSuspendable() CredentialRequestBuilder {
	c.suspendable = true
	return c
}

// SetExpirationDays - the number of days a credential of this type can live
func (c *credentialRequestDef) SetExpirationDays(days int) CredentialRequestBuilder {
	c.period = days
	return c
}

// SetDeprovisionExpired - when set the agent will remove expired credentials from the data plane
func (c *credentialRequestDef) SetDeprovisionExpired() CredentialRequestBuilder {
	c.actions = append(c.actions, "deprovision")
	return c
}

// SetType - set the credential type for the request
func (c *credentialRequestDef) SetType(credType string) CredentialRequestBuilder {
	c.credType = credType
	return c
}

// Register - create the credential request definition and send it to Central
func (c *credentialRequestDef) Register() (*management.CredentialRequestDefinition, error) {
	if c.err != nil {
		return nil, c.err
	}

	if c.requestSchema == nil {
		c.requestSchema, _ = NewSchemaBuilder().Build()
	}

	spec := management.CredentialRequestDefinitionSpec{
		Type:   c.credType,
		Schema: c.requestSchema,
		Provision: &management.CredentialRequestDefinitionSpecProvision{
			Schema: c.provisionSchema,
			Policies: management.CredentialRequestDefinitionSpecProvisionPolicies{
				Renewable:   c.renewable,
				Suspendable: c.suspendable,
			},
		},
	}

	if c.period > 0 {
		spec.Provision.Policies.Expiry = &management.CredentialRequestDefinitionSpecProvisionPoliciesExpiry{
			Period: int32(c.period),
		}
	}

	hashInt, _ := util.ComputeHash(spec)

	// put back in spec the complete request schema
	spec.Schema = c.requestSchema

	if c.title == "" {
		c.title = c.name
	}

	crd := management.NewCredentialRequestDefinition(c.name, "")
	crd.Title = c.title

	crd.Spec = spec

	util.SetAgentDetailsKey(crd, definitions.AttrSpecHash, fmt.Sprintf("%v", hashInt))

	d := util.GetAgentDetails(crd)
	for key, value := range c.agentDetails {
		d[key] = value
	}

	util.SetAgentDetails(crd, d)

	return c.registerFunc(crd)
}
