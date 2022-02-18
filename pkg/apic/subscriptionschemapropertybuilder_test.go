package apic

import (
	"strings"
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
	//assert.Len(t, schemaProperty.enums, 0)
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
	//prop, err = NewSubscriptionSchemaPropertyBuilder().
	//	SetName("name").
	//	IsString().
	//	Build()

	//assert.NotNil(t, err)
	//assert.Nil(t, prop)

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
	assert.Equal(t, "xxx", prop.Enum[0])
	assert.Equal(t, "a", prop.Enum[1])
}

func getFloat64Pointer(value float64) *float64 {
	return &value
}

func getUintPointer(value uint) *uint {
	return &value
}

func Test_SubscriptionPropertyBuilder_Build_with_valid_values(t *testing.T) {
	tests := []struct {
		name        string
		builder     PropertyBuilder
		expectedDef SubscriptionSchemaPropertyDefinition
	}{
		{"Minimal String property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				IsString(),
			SubscriptionSchemaPropertyDefinition{
				Name:  "TheName",
				Title: "TheName",
				Type:  DataTypeString,
			}},
		{"Full String property with unsorted enum and first value",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				SetAPICRefField("APICRefField").
				IsString().
				SetEnumValues([]string{"c", "a", "b"}).
				AddEnumValue("addedValue").
				SetFirstEnumValue("firstValue"),
			SubscriptionSchemaPropertyDefinition{
				Name:        "TheName",
				Title:       "TheName",
				Description: "TheDescription",
				Required:    true,
				Format:      "hidden",
				ReadOnly:    true,
				APICRef:     "APICRefField",
				Type:        DataTypeString,
				Enum:        []string{"firstValue", "c", "a", "b", "addedValue"},
			}},
		{"Full String property with sorted enum and first value",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				SetAPICRefField("APICRefField").
				IsString().
				SetEnumValues([]string{"c", "a", "b"}).
				AddEnumValue("addedValue").
				SetFirstEnumValue("firstValue").
				SetSortEnumValues(),
			SubscriptionSchemaPropertyDefinition{
				Name:        "TheName",
				Title:       "TheName",
				Description: "TheDescription",
				Required:    true,
				Format:      "hidden",
				ReadOnly:    true,
				APICRef:     "APICRefField",
				Type:        DataTypeString,
				Enum:        []string{"firstValue", "a", "addedValue", "b", "c"},
			}},
		{"Minimal Number property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				IsNumber(),
			SubscriptionSchemaPropertyDefinition{
				Name:  "TheName",
				Title: "TheName",
				Type:  DataTypeNumber,
			}},
		{"Full Number property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				SetAPICRefField("APICRefField").
				IsNumber().
				SetMinValue(0.0).
				SetMaxValue(100.5),
			SubscriptionSchemaPropertyDefinition{
				Name:        "TheName",
				Title:       "TheName",
				Description: "TheDescription",
				Required:    true,
				Format:      "hidden",
				ReadOnly:    true,
				APICRef:     "APICRefField",
				Type:        DataTypeNumber,
				Minimum:     getFloat64Pointer(0.0),
				Maximum:     getFloat64Pointer(100.5),
			}},
		{"Minimal Integer property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				IsInteger(),
			SubscriptionSchemaPropertyDefinition{
				Name:  "TheName",
				Title: "TheName",
				Type:  DataTypeInteger,
			}},
		{"Full Integer property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				SetAPICRefField("APICRefField").
				IsInteger().
				SetMinValue(0).
				SetMaxValue(100),
			SubscriptionSchemaPropertyDefinition{
				Name:        "TheName",
				Title:       "TheName",
				Description: "TheDescription",
				Required:    true,
				Format:      "hidden",
				ReadOnly:    true,
				APICRef:     "APICRefField",
				Type:        DataTypeInteger,
				Minimum:     getFloat64Pointer(0),
				Maximum:     getFloat64Pointer(100),
			}},
		{"Minimal Array property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				IsArray(),
			SubscriptionSchemaPropertyDefinition{
				Name:  "TheName",
				Title: "TheName",
				Type:  DataTypeArray,
			}},
		{"Full Array property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				SetAPICRefField("APICRefField").
				IsArray().
				AddItem(NewSubscriptionSchemaPropertyBuilder().
					SetName("ItemName").
					IsString()).
				SetMinItems(0).
				SetMaxItems(1),
			SubscriptionSchemaPropertyDefinition{
				Name:        "TheName",
				Title:       "TheName",
				Description: "TheDescription",
				Required:    true,
				Format:      "hidden",
				ReadOnly:    true,
				APICRef:     "APICRefField",
				Type:        DataTypeArray,
				Items: &AnyOfSubscriptionSchemaPropertyDefinitions{
					AnyOf: []SubscriptionSchemaPropertyDefinition{
						{
							Name:  "ItemName",
							Title: "ItemName",
							Type:  DataTypeString,
						},
					},
				},
				MinItems: getUintPointer(0),
				MaxItems: getUintPointer(1),
			}},
		{"Minimal Object property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				IsObject(),
			SubscriptionSchemaPropertyDefinition{
				Name:  "TheName",
				Title: "TheName",
				Type:  DataTypeObject,
			}},
		{"Full Object property",
			NewSubscriptionSchemaPropertyBuilder().
				SetName("TheName").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				SetAPICRefField("APICRefField").
				IsObject().
				AddProperty(NewSubscriptionSchemaPropertyBuilder().
					SetName("PropertyName").
					SetRequired().
					IsString()),
			SubscriptionSchemaPropertyDefinition{
				Name:        "TheName",
				Title:       "TheName",
				Description: "TheDescription",
				Required:    true,
				Format:      "hidden",
				ReadOnly:    true,
				APICRef:     "APICRefField",
				Type:        DataTypeObject,
				Properties: map[string]SubscriptionSchemaPropertyDefinition{
					"PropertyName": {
						Name:     "PropertyName",
						Title:    "PropertyName",
						Type:     DataTypeString,
						Required: true,
					},
				},
				RequiredProperties: []string{
					"PropertyName",
				},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := tt.builder.Build()
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedDef, *def)
		})
	}
}

func Test_SubscriptionPropertyBuilder_Build_with_error(t *testing.T) {
	tests := []struct {
		name         string
		builder      PropertyBuilder
		errorPattern string
	}{
		{"String property without name", NewSubscriptionSchemaPropertyBuilder().
			IsString(), "without a name"},
		{"Number property without name", NewSubscriptionSchemaPropertyBuilder().
			IsNumber(), "without a name"},
		{"Number property with min greater than max", NewSubscriptionSchemaPropertyBuilder().
			SetName("aNumber").
			IsNumber().
			SetMinValue(2).
			SetMaxValue(1), "greater than"},
		{"Integer property without name", NewSubscriptionSchemaPropertyBuilder().
			IsInteger(),
			"without a name"},
		{"Integer property with min greater than max", NewSubscriptionSchemaPropertyBuilder().
			SetName("anInteger").
			IsInteger().
			SetMinValue(2).
			SetMaxValue(1), "greater than"},
		{"Array property without name", NewSubscriptionSchemaPropertyBuilder().
			IsArray(), "without a name"},
		{"Array property with min items greater than max items", NewSubscriptionSchemaPropertyBuilder().
			SetName("anArray").
			IsArray().
			SetMinItems(2).
			SetMaxItems(1), "greater than"},
		{"Array property with wrong max items", NewSubscriptionSchemaPropertyBuilder().
			SetName("anArray").
			IsArray().
			SetMaxItems(0), "greater than 0"},
		{"Array property with error on added item", NewSubscriptionSchemaPropertyBuilder().
			SetName("anArray").
			IsArray().
			AddItem(NewSubscriptionSchemaPropertyBuilder()), "without a name"},
		{"Object property without name", NewSubscriptionSchemaPropertyBuilder().
			IsObject(), "without a name"},
		{"Object property with error on added property", NewSubscriptionSchemaPropertyBuilder().
			SetName("anObject").
			IsObject().
			AddProperty(NewSubscriptionSchemaPropertyBuilder()), "without a name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop, err := tt.builder.Build()
			assert.NotNil(t, err)
			assert.True(t, strings.Contains(err.Error(), tt.errorPattern))
			assert.Nil(t, prop)
		})
	}
}
