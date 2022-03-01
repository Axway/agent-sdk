package provisioning

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchemaBuilder(t *testing.T) {
	builder := NewSchemaBuilder()
	assert.NotNil(t, builder)

	schemaBuilderProps := builder.(*schemaBuilder)

	// test all the default values
	assert.Nil(t, schemaBuilderProps.err)
	assert.Empty(t, schemaBuilderProps.name)
	assert.Empty(t, schemaBuilderProps.properties)
	assert.Len(t, schemaBuilderProps.uniqueKeys, 0)
}

func TestSubscriptionSchemaBuilderSetters(t *testing.T) {
	_, err := NewSchemaBuilder().
		SetName("name").
		AddUniqueKey("key").
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name").
			SetDescription("description").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		Build()

	assert.Nil(t, err)

	_, err = NewSchemaBuilder().
		SetName("name1").
		SetDescription("description1").
		AddUniqueKey("key").
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name").
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		AddProperty(NewSchemaPropertyBuilder().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		Build()

	assert.NotNil(t, err)
}
