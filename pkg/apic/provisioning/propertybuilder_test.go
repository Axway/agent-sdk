package provisioning

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchemaPropertyBuilder(t *testing.T) {
	builder := NewSchemaPropertyBuilder()
	assert.NotNil(t, builder)

	schemaProperty := builder.(*schemaProperty)

	// test all the default values
	assert.Nil(t, schemaProperty.err)
	assert.Empty(t, schemaProperty.name)
	assert.Empty(t, schemaProperty.description)
	//assert.Len(t, schemaProperty.enums, 0)
	assert.False(t, schemaProperty.required)
	assert.False(t, schemaProperty.readOnly)
	assert.Empty(t, schemaProperty.dataType)
}

func TestSubscriptionSchemaPropertyBuilderSetters(t *testing.T) {
	// No name
	prop, err := NewSchemaPropertyBuilder().Build()

	assert.NotNil(t, err)
	assert.Nil(t, prop)

	// No datatype
	prop, err = NewSchemaPropertyBuilder().
		SetName("name").
		Build()

	assert.NotNil(t, err)
	assert.Nil(t, prop)

	// Datatype twice
	//prop, err = NewSchemaPropertyBuilder().
	//	SetName("name").
	//	IsString().
	//	Build()

	//assert.NotNil(t, err)
	//assert.Nil(t, prop)

	// good path, no enums
	prop, err = NewSchemaPropertyBuilder().
		SetName("name").
		SetDescription("description").
		SetRequired().
		SetReadOnly().
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

	// good path, set enums
	prop, err = NewSchemaPropertyBuilder().
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

	// good path, add enums
	prop, err = NewSchemaPropertyBuilder().
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

	// good path, sort enums & add first item
	prop, err = NewSchemaPropertyBuilder().
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
		expectedDef propertyDefinition
	}{
		{"Minimal String property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				IsString(),
			propertyDefinition{
				Name:  "TheName",
				Title: "The Label",
				Type:  DataTypeString,
			}},
		{"Full String property with unsorted enum and first value",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				IsString().
				IsEncrypted().
				SetAsTextArea().
				SetEnumValues([]string{"c", "a", "b"}).
				AddEnumValue("addedValue").
				SetFirstEnumValue("firstValue").
				SetDefaultValue("a"),
			propertyDefinition{
				Name:         "TheName",
				Title:        "The Label",
				Description:  "TheDescription",
				Required:     true,
				Format:       "hidden",
				ReadOnly:     true,
				IsEncrypted:  true,
				Widget:       "textArea",
				Type:         DataTypeString,
				Enum:         []string{"firstValue", "c", "a", "b", "addedValue"},
				DefaultValue: "a",
			}},
		{"Full String property with sorted enum and first value",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				IsString().
				IsEncrypted().
				SetAsTextArea().
				SetEnumValues([]string{"c", "a", "b"}).
				AddEnumValue("addedValue").
				SetFirstEnumValue("firstValue").
				SetSortEnumValues().
				SetDefaultValue("a"),
			propertyDefinition{
				Name:         "TheName",
				Title:        "The Label",
				Description:  "TheDescription",
				Required:     true,
				Format:       "hidden",
				ReadOnly:     true,
				IsEncrypted:  true,
				Widget:       "textArea",
				Type:         DataTypeString,
				Enum:         []string{"firstValue", "a", "addedValue", "b", "c"},
				DefaultValue: "a",
			}},
		{"Minimal Number property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				IsNumber(),
			propertyDefinition{
				Name:  "TheName",
				Title: "The Label",
				Type:  DataTypeNumber,
			}},
		{"Full Number property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				IsNumber().
				SetMinValue(0.0).
				SetMaxValue(100.5).
				SetDefaultValue(50.5),
			propertyDefinition{
				Name:         "TheName",
				Title:        "The Label",
				Description:  "TheDescription",
				Required:     true,
				Format:       "hidden",
				ReadOnly:     true,
				Type:         DataTypeNumber,
				Minimum:      getFloat64Pointer(0.0),
				Maximum:      getFloat64Pointer(100.5),
				DefaultValue: getFloat64Pointer(50.5),
			}},
		{"Minimal Integer property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				IsInteger(),
			propertyDefinition{
				Name:  "TheName",
				Title: "The Label",
				Type:  DataTypeInteger,
			}},
		{"Full Integer property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				IsInteger().
				SetMinValue(0).
				SetMaxValue(100).
				SetDefaultValue(50),
			propertyDefinition{
				Name:         "TheName",
				Title:        "The Label",
				Description:  "TheDescription",
				Required:     true,
				Format:       "hidden",
				ReadOnly:     true,
				Type:         DataTypeInteger,
				Minimum:      getFloat64Pointer(0),
				Maximum:      getFloat64Pointer(100),
				DefaultValue: getFloat64Pointer(50),
			}},
		{"Minimal Array property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				IsArray(),
			propertyDefinition{
				Name:  "TheName",
				Title: "The Label",
				Type:  DataTypeArray,
			}},
		{"Full Array property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				IsArray().
				AddItem(NewSchemaPropertyBuilder().
					SetName("ItemName").
					SetLabel("Item Label").
					IsString()).
				SetMinItems(0).
				SetMaxItems(1),
			propertyDefinition{
				Name:        "TheName",
				Title:       "The Label",
				Description: "TheDescription",
				Required:    true,
				Format:      "hidden",
				ReadOnly:    true,
				Type:        DataTypeArray,
				Items: &anyOfPropertyDefinitions{
					AnyOf: []propertyDefinition{
						{
							Name:  "ItemName",
							Title: "Item Label",
							Type:  DataTypeString,
						},
					},
				},
				MinItems:    getUintPointer(0),
				MaxItems:    getUintPointer(1),
				UniqueItems: true,
			}},
		{"Minimal Object property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				IsObject(),
			propertyDefinition{
				Name: "TheName",
				Type: DataTypeObject,
			}},
		{"Full Object property",
			NewSchemaPropertyBuilder().
				SetName("TheName").
				SetLabel("The Label").
				SetDescription("TheDescription").
				SetRequired().
				SetHidden().
				SetReadOnly().
				IsObject().
				AddProperty(NewSchemaPropertyBuilder().
					SetName("PropertyName").
					SetLabel("Property Label").
					SetRequired().
					IsString()),
			propertyDefinition{
				Name:        "TheName",
				Title:       "The Label",
				Description: "TheDescription",
				Required:    true,
				Format:      "hidden",
				ReadOnly:    true,
				Type:        DataTypeObject,
				Properties: map[string]propertyDefinition{
					"PropertyName": {
						Name:     "PropertyName",
						Title:    "Property Label",
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
		{"String property without name", NewSchemaPropertyBuilder().
			IsString(), "without a name"},
		{"String property with default value not present in enum list", NewSchemaPropertyBuilder().
			SetName("aString").
			IsString().
			SetEnumValues([]string{"a", "b"}).
			SetDefaultValue("z"), "must be present in the enum list"},
		{"Number property without name", NewSchemaPropertyBuilder().
			IsNumber(), "without a name"},
		{"Number property with min greater than max", NewSchemaPropertyBuilder().
			SetName("aNumber").
			IsNumber().
			SetMinValue(2).
			SetMaxValue(1), "greater than"},
		{"Number property with default value greater than max", NewSchemaPropertyBuilder().
			SetName("aNumber").
			IsNumber().
			SetMaxValue(1).
			SetDefaultValue(2), "must be equal or lower than max value"},
		{"Number property with default value lower than min", NewSchemaPropertyBuilder().
			SetName("aNumber").
			IsNumber().
			SetMinValue(2).
			SetDefaultValue(1), "must be equal or greater than min value"},
		{"Integer property without name", NewSchemaPropertyBuilder().
			IsInteger(),
			"without a name"},
		{"Integer property with min greater than max", NewSchemaPropertyBuilder().
			SetName("anInteger").
			IsInteger().
			SetMinValue(2).
			SetMaxValue(1), "greater than"},
		{"Integer property with default value greater than max", NewSchemaPropertyBuilder().
			SetName("aNumber").
			IsInteger().
			SetMaxValue(1).
			SetDefaultValue(2), "must be equal or lower than max value"},
		{"Integer property with default value lower than min", NewSchemaPropertyBuilder().
			SetName("aNumber").
			IsInteger().
			SetMinValue(2).
			SetDefaultValue(1), "must be equal or greater than min value"},
		{"Array property without name", NewSchemaPropertyBuilder().
			IsArray(), "without a name"},
		{"Array property with min items greater than max items", NewSchemaPropertyBuilder().
			SetName("anArray").
			IsArray().
			SetMinItems(2).
			SetMaxItems(1), "greater than"},
		{"Array property with wrong max items", NewSchemaPropertyBuilder().
			SetName("anArray").
			IsArray().
			SetMaxItems(0), "greater than 0"},
		{"Array property with error on added item", NewSchemaPropertyBuilder().
			SetName("anArray").
			IsArray().
			AddItem(NewSchemaPropertyBuilder()), "without a name"},
		{"Object property without name", NewSchemaPropertyBuilder().
			IsObject(), "without a name"},
		{"Object property with error on added property", NewSchemaPropertyBuilder().
			SetName("anObject").
			IsObject().
			AddProperty(NewSchemaPropertyBuilder()), "without a name"},
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
