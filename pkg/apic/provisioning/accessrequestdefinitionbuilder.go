package provisioning

import (
	"fmt"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
)

// RegisterAccessRequestDefinition - the function signature used when calling the NewAccessRequestBuilder function
type RegisterAccessRequestDefinition func(accessRequestDefinition *management.AccessRequestDefinition) (*management.AccessRequestDefinition, error)

type accessRequestDef struct {
	name                   string
	title                  string
	appProfDef             string
	provisionSchema        map[string]interface{}
	requestSchema          map[string]interface{}
	provisionEqualsRequest bool
	transferable           bool
	registerFunc           RegisterAccessRequestDefinition
	err                    error
}

// AccessRequestBuilder - aids in creating a new access request
type AccessRequestBuilder interface {
	SetName(name string) AccessRequestBuilder
	SetTitle(title string) AccessRequestBuilder
	SetRequestSchema(schema SchemaBuilder) AccessRequestBuilder
	SetProvisionSchema(schema SchemaBuilder) AccessRequestBuilder
	SetProvisionSchemaToRequestSchema() AccessRequestBuilder
	SetApplicationProfileDefinition(name string) AccessRequestBuilder
	IsTransferable() AccessRequestBuilder
	Register() (*management.AccessRequestDefinition, error)
}

// NewAccessRequestBuilder - called by the agent package and sends in the function that registers this access request
func NewAccessRequestBuilder(registerFunc RegisterAccessRequestDefinition) AccessRequestBuilder {
	return &accessRequestDef{
		registerFunc: registerFunc,
	}
}

// SetName - set the name of the access request
func (a *accessRequestDef) SetName(name string) AccessRequestBuilder {
	a.name = name
	return a
}

// SetTitle - set the title of the access request
func (a *accessRequestDef) SetTitle(title string) AccessRequestBuilder {
	a.title = title
	return a
}

// SetApplicationProfileDefinition - set the name of the application profile definition
func (a *accessRequestDef) SetApplicationProfileDefinition(name string) AccessRequestBuilder {
	a.appProfDef = name
	return a
}

// SetRequestSchema - set the schema to be used for access requests request data
func (a *accessRequestDef) SetRequestSchema(schema SchemaBuilder) AccessRequestBuilder {
	if a.err != nil {
		return a
	}

	if schema != nil {
		a.requestSchema, a.err = schema.Build()
	} else {
		a.err = fmt.Errorf("expected a SchemaBuilder argument but received nil")
	}

	return a
}

// SetProvisionSchemaToRequestSchema - set the schema to be used for access requests provisioning data
func (a *accessRequestDef) SetProvisionSchemaToRequestSchema() AccessRequestBuilder {
	if a.err != nil {
		return a
	}

	if a.provisionSchema != nil {
		a.err = fmt.Errorf("can't duplicate request schema as provisioning schema is set")
		return a
	}

	a.provisionEqualsRequest = true
	return a
}

// SetProvisionSchema - set the schema to be used for access requests provisioning data
func (a *accessRequestDef) SetProvisionSchema(schema SchemaBuilder) AccessRequestBuilder {
	if a.err != nil {
		return a
	}

	if schema != nil {
		a.provisionSchema, a.err = schema.Build()
	} else {
		a.err = fmt.Errorf("expected a SchemaBuilder argument but received nil")
	}

	return a
}

// IsTransferable - marks the access request transferable for plan migration
func (a *accessRequestDef) IsTransferable() AccessRequestBuilder {
	a.transferable = true
	return a
}

// Register - create the access request defintion and send it to Central
func (a *accessRequestDef) Register() (*management.AccessRequestDefinition, error) {
	if a.err != nil {
		return nil, a.err
	}

	if a.requestSchema == nil {
		a.requestSchema, _ = NewSchemaBuilder().Build()
	}

	if a.provisionSchema == nil {
		if a.provisionEqualsRequest {
			a.provisionSchema = util.MergeMapStringInterface(a.requestSchema)
		} else {
			a.provisionSchema, _ = NewSchemaBuilder().Build()
		}
	}

	if a.title == "" {
		a.title = a.name
	}

	spec := management.AccessRequestDefinitionSpec{
		Schema: a.requestSchema,
		Provision: &management.AccessRequestDefinitionSpecProvision{
			Schema: a.provisionSchema,
		},
	}

	if a.transferable {
		spec.Provision.Policies = management.AccessRequestDefinitionSpecProvisionPolicies{
			Transferable: true,
		}
	}

	hashInt, _ := util.ComputeHash(spec)

	// put back in spec the complete request schema
	spec.Schema = a.requestSchema

	if a.name == "" {
		a.name = util.ConvertUnitToString(hashInt)
	}

	ard := management.NewAccessRequestDefinition(a.name, "")
	ard.Title = a.title
	ard.Spec = spec

	if a.appProfDef != "" {
		ard.Applicationprofile = management.AccessRequestDefinitionApplicationprofile{
			Name: a.appProfDef,
		}
	}

	util.SetAgentDetailsKey(ard, definitions.AttrSpecHash, fmt.Sprintf("%v", hashInt))

	return a.registerFunc(ard)
}
