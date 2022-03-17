package provisioning

import (
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// RegisterAccessRequestDefinition - the function signature used when calling the NewAccessRequestBuilder function
type RegisterAccessRequestDefinition func(accessRequestDefinition *v1alpha1.AccessRequestDefinition) (*v1alpha1.AccessRequestDefinition, error)

type accessRequestDef struct {
	name         string
	schema       map[string]interface{}
	registerFunc RegisterAccessRequestDefinition
	err          error
}

// AccessRequestBuilder - aids in creating a new access request
type AccessRequestBuilder interface {
	SetName(name string) AccessRequestBuilder
	SetSchema(schema SchemaBuilder) AccessRequestBuilder
	Register() (*v1alpha1.AccessRequestDefinition, error)
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

// SetSchema - set the schema to be used for access requests
func (a *accessRequestDef) SetSchema(schema SchemaBuilder) AccessRequestBuilder {
	if a.err != nil {
		return a
	}

	if schema != nil {
		a.schema, a.err = schema.Build()
	} else {
		a.err = fmt.Errorf("expected a SchemaBuilder argument but received nil")
	}

	return a
}

// Register - create the access request defintion and send it to Central
func (a *accessRequestDef) Register() (*v1alpha1.AccessRequestDefinition, error) {
	if a.err != nil {
		return nil, a.err
	}

	if a.schema == nil {
		a.schema, _ = NewSchemaBuilder().Build()
	}

	ard := &v1alpha1.AccessRequestDefinition{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.AccessRequestDefinitionGVK(),
			Title:            a.name,
			Name:             a.name,
		},
		Spec: v1alpha1.AccessRequestDefinitionSpec{
			Schema: a.schema,
		},
	}

	return a.registerFunc(ard)
}
