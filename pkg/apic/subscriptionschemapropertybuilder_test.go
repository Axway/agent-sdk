package apic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSubscriptionSchemaPropertyBuilder(t *testing.T) {
	builder := NewSubscriptionSchemaPropertyBuilder()
	assert.NotNil(t, builder)

	schemaProperty := builder.(*schemaProperty)

	// test all the default values
	assert.Nil(t, schemaProperty.err)
	assert.Empty(t, schemaProperty.name)
	assert.Empty(t, schemaProperty.description)
	assert.Empty(t, schemaProperty.apicRefField)
	assert.Len(t, schemaProperty.enums, 0)
	assert.False(t, schemaProperty.required)
	assert.False(t, schemaProperty.readOnly)
	assert.Empty(t, schemaProperty.dataType)
}

func TestSubscriptionSchemaPropertyBuilderSetters(t *testing.T) {
	// No name
	prop, err := NewSubscriptionSchemaPropertyBuilder().Build()

	assert.NotNil(t, err)
	assert.Nil(t, prop)

	// No datatype
	prop, err = NewSubscriptionSchemaPropertyBuilder().
		SetName("name").
		Build()

	assert.NotNil(t, err)
	assert.Nil(t, prop)

	// Datatype twice
	prop, err = NewSubscriptionSchemaPropertyBuilder().
		SetName("name").
		IsString().
		IsString().
		Build()

	assert.NotNil(t, err)
	assert.Nil(t, prop)

	// good path, no enums
	prop, err = NewSubscriptionSchemaPropertyBuilder().
		SetName("name").
		SetDescription("description").
		SetRequired().
		SetReadOnly().
		SetAPICRefField("refField").
		SetHidden().
		IsString().
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, prop)
	assert.Len(t, prop.Enum, 0)
	assert.Equal(t, "name", prop.Name)
	assert.Equal(t, "description", prop.Description)
	assert.True(t, prop.Required)
	assert.True(t, prop.ReadOnly)
	assert.Equal(t, prop.Format, "hidden")
	assert.Equal(t, "refField", prop.APICRef)

	// good path, set enums
	prop, err = NewSubscriptionSchemaPropertyBuilder().
		SetName("name").
		IsString().
		SetEnumValues([]string{"a", "b", "c", "c"}).
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, prop)
	assert.Len(t, prop.Enum, 3)
	assert.Equal(t, "name", prop.Name)
	assert.Equal(t, "", prop.Description)
	assert.False(t, prop.Required)
	assert.False(t, prop.ReadOnly)
	assert.Equal(t, prop.Format, "")
	assert.Equal(t, "", prop.APICRef)

	// good path, add enums
	prop, err = NewSubscriptionSchemaPropertyBuilder().
		SetName("name").
		IsString().
		AddEnumValue("a").
		AddEnumValue("b").
		AddEnumValue("c").
		AddEnumValue("c").
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, prop)
	assert.Len(t, prop.Enum, 3)
	assert.Equal(t, "name", prop.Name)
	assert.Equal(t, "", prop.Description)
	assert.False(t, prop.Required)
	assert.False(t, prop.ReadOnly)
	assert.Equal(t, prop.Format, "")
	assert.Equal(t, "", prop.APICRef)
	assert.False(t, prop.SortEnums)
	assert.Equal(t, "", prop.FirstEnumItem)

	// good path, sort enums & add first item
	prop, err = NewSubscriptionSchemaPropertyBuilder().
		SetName("name").
		IsString().
		AddEnumValue("c").
		AddEnumValue("a").
		AddEnumValue("b").
		SetSortEnumValues().
		SetFirstEnumValue("xxx").
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, prop)
	assert.Len(t, prop.Enum, 4)
	assert.Equal(t, "name", prop.Name)
	assert.Equal(t, "", prop.Description)
	assert.False(t, prop.Required)
	assert.False(t, prop.ReadOnly)
	assert.Equal(t, prop.Format, "")
	assert.Equal(t, "", prop.APICRef)
	assert.True(t, prop.SortEnums)
	assert.Equal(t, "xxx", prop.FirstEnumItem)
	assert.Equal(t, "xxx", prop.Enum[0])
	assert.Equal(t, "a", prop.Enum[1])
}
