package provisioning

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	axwayOrder = "x-axway-order"
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
	schemaMap1, err := NewSchemaBuilder().
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
	assert.NotNil(t, schemaMap1)
	assert.NotEmpty(t, schemaMap1[axwayOrder])

	// assert that properties in property order takes precedence over added property appended order
	propertyOrder, _ := schemaMap1[axwayOrder].([]interface{})
	assert.Equal(t, propertyOrder[0].(string), "name3")
	assert.Equal(t, propertyOrder[1].(string), "name2")
	assert.Equal(t, propertyOrder[2].(string), "name1")

	// do no set property order.  property order appended as each property is added
	schemaMap2, err := NewSchemaBuilder().
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
	assert.NotNil(t, schemaMap2)
	assert.NotEmpty(t, schemaMap2[axwayOrder])

	// assert that appended properties exist
	assert.Contains(t, schemaMap2[axwayOrder], "name5")
	assert.Contains(t, schemaMap2[axwayOrder], "name3")
	assert.Contains(t, schemaMap2[axwayOrder], "name1")

	// set property order, however, no properties were added to match any properties in property order
	schemaMap3, _ := NewSchemaBuilder().
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

	assert.NotNil(t, schemaMap3)
	assert.NotEmpty(t, schemaMap3[axwayOrder])

	// assert that properties in property order weren't added
	propertyOrder1, _ := schemaMap3[axwayOrder].([]interface{})
	for _, item := range propertyOrder1 {
		assert.NotEqual(t, item.(string), "name5")
	}
}

func TestSchemaBuilderWithDependenciesProperties(t *testing.T) {
	// set dependent property - dependent property definition error
	_, err := NewSchemaBuilder().
		SetName("sch").
		AddProperty(NewSchemaPropertyBuilder().
			SetName("dep").
			IsString().
			SetEnumValues([]string{"a", "b", "c"}).
			AddDependency("a", NewSchemaPropertyBuilder().
				SetName("dep"))).
		Build()

	assert.NotNil(t, err)

	// set dependent property - good path
	s, err := NewSchemaBuilder().
		SetName("sch").
		AddProperty(NewSchemaPropertyBuilder().
			SetName("prop").
			IsString().
			SetEnumValues([]string{"a", "b", "c"}).
			AddDependency("a", NewSchemaPropertyBuilder().
				SetName("a-prop").
				IsString())).
		Build()
	assert.Nil(t, err)
	schema := &jsonSchema{}
	buf, _ := json.Marshal(s)
	json.Unmarshal(buf, schema)
	assert.NotNil(t, schema.Dependencies)

}
