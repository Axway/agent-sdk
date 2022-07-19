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
	assert.Len(t, schemaBuilderProps.propertyOrder, 0)
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

	// set property order - property order takes precedence
	_, err = NewSchemaBuilder().
		SetName("name").
		AddUniqueKey("key").
		SetPropertyOrder([]string{"name3", "name2", "name1"}).
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name1").
			SetDescription("description").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name2").
			SetDescription("description").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name3").
			SetDescription("description").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		Build()

	assert.Nil(t, err)

	// do no set property order.  property order appended as each property is added
	_, err = NewSchemaBuilder().
		SetName("name").
		AddUniqueKey("key").
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name5").
			SetDescription("description").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name3").
			SetDescription("description").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name1").
			SetDescription("description").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		Build()

	assert.Nil(t, err)

	// set property order, however, no properties were added to match any properties in property order
	_, err = NewSchemaBuilder().
		SetName("name").
		AddUniqueKey("key").
		SetPropertyOrder([]string{"name3", "name2", "name1"}).
		AddProperty(NewSchemaPropertyBuilder().
			SetName("name5").
			SetDescription("description").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		Build()

	assert.Nil(t, err)
}
