package provisioning

import (
	"fmt"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
)

// RegisterApplicationProfileDefinition - the function signature used when calling the NewApplicationProfileBuilder function
type RegisterApplicationProfileDefinition func(applicationProfileDefinition *management.ApplicationProfileDefinition) (*management.ApplicationProfileDefinition, error)

type applicationProfileDef struct {
	name          string
	title         string
	requestSchema map[string]interface{}
	registerFunc  RegisterApplicationProfileDefinition
	err           error
}

// ApplicationProfileBuilder - aids in creating a new access request
type ApplicationProfileBuilder interface {
	SetName(name string) ApplicationProfileBuilder
	SetTitle(title string) ApplicationProfileBuilder
	SetRequestSchema(schema SchemaBuilder) ApplicationProfileBuilder
	Register() (*management.ApplicationProfileDefinition, error)
}

// NewApplicationProfileBuilder - called by the agent package and sends in the function that registers this access request
func NewApplicationProfileBuilder(registerFunc RegisterApplicationProfileDefinition) ApplicationProfileBuilder {
	return &applicationProfileDef{
		registerFunc: registerFunc,
	}
}

// SetName - set the name of the access request
func (a *applicationProfileDef) SetName(name string) ApplicationProfileBuilder {
	a.name = name
	return a
}

// SetTitle - set the title of the access request
func (a *applicationProfileDef) SetTitle(title string) ApplicationProfileBuilder {
	a.title = title
	return a
}

// SetRequestSchema - set the schema to be used for access requests request data
func (a *applicationProfileDef) SetRequestSchema(schema SchemaBuilder) ApplicationProfileBuilder {
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

// Register - create the access request defintion and send it to Central
func (a *applicationProfileDef) Register() (*management.ApplicationProfileDefinition, error) {
	if a.err != nil {
		return nil, a.err
	}

	if a.requestSchema == nil {
		a.requestSchema, _ = NewSchemaBuilder().Build()
	}

	if a.title == "" {
		a.title = a.name
	}

	spec := management.ApplicationProfileDefinitionSpec{
		Schema: a.requestSchema,
	}

	hashInt, _ := util.ComputeHash(spec)

	if a.name == "" {
		a.name = util.ConvertUnitToString(hashInt)
	}

	ard := management.NewApplicationProfileDefinition(a.name, "")
	ard.Title = a.title
	ard.Spec = spec

	util.SetAgentDetailsKey(ard, definitions.AttrSpecHash, fmt.Sprintf("%v", hashInt))

	return a.registerFunc(ard)
}
