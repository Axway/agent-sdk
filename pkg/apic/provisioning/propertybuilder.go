package provisioning

import (
	"fmt"
	"sort"
)

// Supported data types
const (
	DataTypeString  = "string"
	DataTypeNumber  = "number"
	DataTypeInteger = "integer"
	DataTypeArray   = "array"
	DataTypeObject  = "object"
)

// anyOfPropertyDefinitions - used for items of propertyDefinition
type anyOfPropertyDefinitions struct {
	AnyOf []propertyDefinition `json:"anyOf,omitempty"`
}

// propertyDefinition -
type propertyDefinition struct {
	Type               string                        `json:"type"`
	Title              string                        `json:"title"`
	Description        string                        `json:"description"`
	Enum               []string                      `json:"enum,omitempty"`
	DefaultValue       interface{}                   `json:"default,omitempty"`
	ReadOnly           bool                          `json:"readOnly,omitempty"`
	Format             string                        `json:"format,omitempty"`
	Properties         map[string]propertyDefinition `json:"properties,omitempty"`
	RequiredProperties []string                      `json:"required,omitempty"`
	Items              *anyOfPropertyDefinitions     `json:"items,omitempty"`    // We use a pointer to avoid generating an empty struct if not set
	MinItems           *uint                         `json:"minItems,omitempty"` // We use a pointer to differentiate the "blank value" from a chosen 0 min value
	MaxItems           *uint                         `json:"maxItems,omitempty"` // We use a pointer to differentiate the "blank value" from a chosen 0 min value
	Minimum            *float64                      `json:"minimum,omitempty"`  // We use a pointer to differentiate the "blank value" from a chosen 0 min value
	Maximum            *float64                      `json:"maximum,omitempty"`  // We use a pointer to differentiate the "blank value" from a chosen 0 max value
	IsEncrypted        bool                          `json:"x-axway-encrypted,omitempty"`
	TextArea           string                        `json:"x-axway-widget,omitempty"`
	Name               string                        `json:"-"`
	Required           bool                          `json:"-"`
}

// PropertyBuilder - mandatory methods for all property builders
type PropertyBuilder interface {
	// Build - builds the property, this is called automatically by the schema builder
	Build() (*propertyDefinition, error)
}

// TypePropertyBuilder - common methods related to type property builders
type TypePropertyBuilder interface {
	// SetLabel - sets the label for the property
	SetLabel(label string) TypePropertyBuilder
	// SetName - sets the name of the property
	SetName(name string) TypePropertyBuilder
	// SetDescription - set the description of the property
	SetDescription(description string) TypePropertyBuilder
	// SetRequired - set the property as a required field in the schema
	SetRequired() TypePropertyBuilder
	// SetReadOnly - set the property as a read only property
	SetReadOnly() TypePropertyBuilder
	// SetHidden - set the property as a hidden property
	SetHidden() TypePropertyBuilder
	// IsString - Set the property to be of type string
	IsString() StringPropertyBuilder
	// IsInteger - Set the property to be of type integer
	IsInteger() IntegerPropertyBuilder
	// IsNumber - Set the property to be of type number
	IsNumber() NumberPropertyBuilder
	// IsArray - Set the property to be of type array
	IsArray() ArrayPropertyBuilder
	// IsObject - Set the property to be of type object
	IsObject() ObjectPropertyBuilder
	PropertyBuilder
}

// StringPropertyBuilder - specific methods related to the String property builders
type StringPropertyBuilder interface {
	// SetEnumValues - Set a list of valid values for the property
	SetEnumValues(values []string) StringPropertyBuilder
	// SetSortEnumValues - Sort the allowed values alphabetically in the schema
	SetSortEnumValues() StringPropertyBuilder
	// SetFirstEnumValue - Set the value that should appear first in the list
	SetFirstEnumValue(value string) StringPropertyBuilder
	// AddEnumValue - Add another value to the list of allowed values for the property
	AddEnumValue(value string) StringPropertyBuilder
	// IsEncrypted - Set that this field must be encrypted at rest
	IsEncrypted() StringPropertyBuilder
	// SetDefaultValue - Define the initial value for the property
	SetDefaultValue(value string) StringPropertyBuilder
	// SetAsTextArea - Set value to be rendered as a textarea box within the UI
	SetAsTextArea() StringPropertyBuilder
	PropertyBuilder
}

// NumberPropertyBuilder - specific methods related to the Number property builders
type NumberPropertyBuilder interface {
	// SetMinValue - Set the minimum allowed number value
	SetMinValue(min float64) NumberPropertyBuilder
	// SetMaxValue - Set the maximum allowed number value
	SetMaxValue(min float64) NumberPropertyBuilder
	// SetDefaultValue - Define the initial value for the property
	SetDefaultValue(value float64) NumberPropertyBuilder
	PropertyBuilder
}

// IntegerPropertyBuilder - specific methods related to the Integer property builders
type IntegerPropertyBuilder interface {
	// SetMinValue - Set the minimum allowed integer value
	SetMinValue(min int64) IntegerPropertyBuilder
	// SetMaxValue - Set the maximum allowed integer value
	SetMaxValue(min int64) IntegerPropertyBuilder
	// SetDefaultValue - Define the initial value for the property
	SetDefaultValue(value int64) IntegerPropertyBuilder
	PropertyBuilder
}

// ObjectPropertyBuilder - specific methods related to the Object property builders
type ObjectPropertyBuilder interface {
	// AddProperty - Add a property in the object property
	AddProperty(property PropertyBuilder) ObjectPropertyBuilder
	PropertyBuilder
}

// ArrayPropertyBuilder - specific methods related to the Array property builders
type ArrayPropertyBuilder interface {
	// AddItem - Add an item property in the array property
	AddItem(item PropertyBuilder) ArrayPropertyBuilder
	// SetMinItems - Set the minimum number of items in the array property
	SetMinItems(min uint) ArrayPropertyBuilder
	// SetMaxItems - Set the maximum number of items in the array property
	SetMaxItems(max uint) ArrayPropertyBuilder
	PropertyBuilder
}

// schemaProperty - holds basic info needed to create a subscription schema property
type schemaProperty struct {
	err         error
	name        string
	label       string
	description string
	required    bool
	readOnly    bool
	hidden      bool
	dataType    string
	PropertyBuilder
}

// NewSchemaPropertyBuilder - Creates a new subscription schema property builder
func NewSchemaPropertyBuilder() TypePropertyBuilder {
	return &schemaProperty{}
}

// SetName - sets the name of the property
func (p *schemaProperty) SetName(name string) TypePropertyBuilder {
	p.name = name
	return p
}

// SetLabel - sets the label of the property
func (p *schemaProperty) SetLabel(label string) TypePropertyBuilder {
	p.label = label
	return p
}

// SetDescription - set the description of the property
func (p *schemaProperty) SetDescription(description string) TypePropertyBuilder {
	p.description = description
	return p
}

// SetRequired - set the property as a required field in the schema
func (p *schemaProperty) SetRequired() TypePropertyBuilder {
	p.required = true
	return p
}

// SetReadOnly - set the property as a read only property
func (p *schemaProperty) SetReadOnly() TypePropertyBuilder {
	p.readOnly = true
	return p
}

// SetHidden - set the property as a hidden property
func (p *schemaProperty) SetHidden() TypePropertyBuilder {
	p.hidden = true
	return p
}

// IsString - Set the property to be of type string
func (p *schemaProperty) IsString() StringPropertyBuilder {
	p.dataType = DataTypeString
	return &stringSchemaProperty{
		schemaProperty: p,
	}
}

// IsNumber - Set the property to be of type number
func (p *schemaProperty) IsNumber() NumberPropertyBuilder {
	p.dataType = DataTypeNumber
	return &numberSchemaProperty{
		schemaProperty: p,
	}
}

// IsInteger - Set the property to be of type integer
func (p *schemaProperty) IsInteger() IntegerPropertyBuilder {
	p.dataType = DataTypeInteger
	return &integerSchemaProperty{
		numberSchemaProperty{
			schemaProperty: p,
		},
	}
}

// IsArray - Set the property to be of type array
func (p *schemaProperty) IsArray() ArrayPropertyBuilder {
	p.dataType = DataTypeArray
	return &arraySchemaProperty{
		schemaProperty: p,
	}
}

// IsObject - Set the property to be of type object
func (p *schemaProperty) IsObject() ObjectPropertyBuilder {
	p.dataType = DataTypeObject
	return &objectSchemaProperty{
		schemaProperty: p,
	}
}

// Build - create a string propertyDefinition for use in the subscription schema builder
func (p *schemaProperty) Build() (*propertyDefinition, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.name == "" {
		return nil, fmt.Errorf("cannot add a schema property without a name")
	}

	if p.dataType == "" {
		return nil, fmt.Errorf("schema property named %s must have a data type", p.name)
	}

	prop := &propertyDefinition{
		Name:        p.name,
		Title:       p.label,
		Type:        p.dataType,
		Description: p.description,
		ReadOnly:    p.readOnly,
		Required:    p.required,
	}

	if p.hidden {
		prop.Format = "hidden"
	}

	return prop, nil
}

/**
  string property datatype
*/
// stringSchemaProperty - adds specific info needed for a string schema property
type stringSchemaProperty struct {
	schemaProperty *schemaProperty
	isEncrypted    bool
	sortEnums      bool
	firstEnumValue string
	enums          []string
	propertyOrder  int
	textArea       string
	defaultValue   string
	StringPropertyBuilder
}

// SetEnumValues - add a list of enum values to the property
func (p *stringSchemaProperty) SetEnumValues(values []string) StringPropertyBuilder {
	dict := make(map[string]bool, 0)

	// use a temp map to filter out any duplicate values from the input
	for _, value := range values {
		if _, ok := dict[value]; !ok {
			dict[value] = true
			p.enums = append(p.enums, value)
		}
	}

	return p
}

// SetSortEnumValues - indicates to sort the enums
func (p *stringSchemaProperty) SetSortEnumValues() StringPropertyBuilder {
	p.sortEnums = true
	return p
}

// SetFirstEnumValue - Sets a first item for enums. Only needed for sorted enums if you want a specific
// item first in the list
func (p *stringSchemaProperty) SetFirstEnumValue(value string) StringPropertyBuilder {
	p.firstEnumValue = value
	return p
}

func (p *stringSchemaProperty) enumContains(str string) bool {
	for _, v := range p.enums {
		if v == str {
			return true
		}
	}
	return false
}

// AddEnumValue - Add another value to the list of allowed values for the property
func (p *stringSchemaProperty) AddEnumValue(value string) StringPropertyBuilder {
	if !p.enumContains(value) {
		p.enums = append(p.enums, value)
	}
	return p
}

// SetDefaultValue - Define the initial value for the property
func (p *stringSchemaProperty) SetDefaultValue(value string) StringPropertyBuilder {
	p.defaultValue = value
	return p
}

// SetPropertyOrder - add a list of enum values to the property
func (p *stringSchemaProperty) SetPropertyOrder(propertyOrder int) StringPropertyBuilder {
	p.propertyOrder = propertyOrder
	return p
}

// SetAsTextArea - set the field to be rendered as a textarea box within the UI
func (p *stringSchemaProperty) SetAsTextArea() StringPropertyBuilder {
	p.textArea = "textArea"
	return p
}

// IsEncrypted - Sets that this field needs to be encrypted at rest
func (p *stringSchemaProperty) IsEncrypted() StringPropertyBuilder {
	p.isEncrypted = true
	return p
}

// Build - create a string propertyDefinition for use in the subscription schema builder
func (p *stringSchemaProperty) Build() (def *propertyDefinition, err error) {

	def, err = p.schemaProperty.Build()
	if err != nil {
		return
	}

	// sort if specified to do so
	if p.sortEnums {
		sort.Strings(p.enums)
	}

	// append item to start if specified
	if p.firstEnumValue != "" {
		p.enums = append([]string{p.firstEnumValue}, p.enums...)
	}
	def.Enum = p.enums

	// set default value
	if len(p.defaultValue) > 0 {
		if len(p.enums) > 0 {
			// Check validity for defaultValue
			isDefaultValueValid := false
			for _, x := range p.enums {
				if x == p.defaultValue {
					isDefaultValueValid = true
					break
				}
			}
			if isDefaultValueValid == false {
				return nil, fmt.Errorf("Default value (%s) must be present in the enum list (%s)", p.defaultValue, p.enums)
			}
		}
		def.DefaultValue = p.defaultValue
	}

	// set if the property is encrypted at rest
	def.IsEncrypted = p.isEncrypted

	// set field to be rendered as a textarea box within the UI
	def.TextArea = p.textArea

	return def, err
}

/**
  number property datatype builder
*/
// numberSchemaProperty - adds specific info needed for a number schema property
type numberSchemaProperty struct {
	schemaProperty *schemaProperty
	minValue       *float64 // We use a pointer to differentiate the "blank value" from a chosen 0 min value
	maxValue       *float64 // We use a pointer to differentiate the "blank value" from a chosen 0 max value
	defaultValue   *float64
	PropertyBuilder
}

// SetMinValue - set the minimum allowed value
func (p *numberSchemaProperty) SetMinValue(min float64) NumberPropertyBuilder {
	p.minValue = &min
	return p
}

// SetMaxValue - set the maximum allowed value
func (p *numberSchemaProperty) SetMaxValue(max float64) NumberPropertyBuilder {
	p.maxValue = &max
	return p
}

// SetDefaultValue - Define the initial value for the property
func (p *numberSchemaProperty) SetDefaultValue(value float64) NumberPropertyBuilder {
	p.defaultValue = &value
	return p
}

// Build - create the propertyDefinition for use in the subscription schema builder
func (p *numberSchemaProperty) Build() (def *propertyDefinition, err error) {
	def, err = p.schemaProperty.Build()
	if err != nil {
		return
	}

	if p.minValue != nil && p.maxValue != nil && *p.minValue > *p.maxValue {
		return nil, fmt.Errorf("max value (%f) must be greater than min value (%f)", *p.maxValue, *p.minValue)
	}

	if p.defaultValue != nil {
		if p.minValue != nil && *p.defaultValue < *p.minValue {
			return nil, fmt.Errorf("Default value (%f) must be equal or greater than min value (%f)", *p.defaultValue, *p.minValue)
		}
		if p.maxValue != nil && *p.defaultValue > *p.maxValue {
			return nil, fmt.Errorf("Default value (%f) must be equal or lower than max value (%f)", *p.defaultValue, *p.maxValue)
		}
		def.DefaultValue = p.defaultValue
	}

	def.Minimum = p.minValue
	def.Maximum = p.maxValue
	return def, err
}

/**
  integer property datatype builder
*/
// integerSchemaProperty - adds specific info needed for an integer schema property
type integerSchemaProperty struct {
	numberSchemaProperty
}

// SetMinValue - set the minimum allowed value
func (p *integerSchemaProperty) SetMinValue(min int64) IntegerPropertyBuilder {
	minimum := float64(min)
	p.minValue = &minimum
	return p
}

// SetMaxValue - set the maximum allowed value
func (p *integerSchemaProperty) SetMaxValue(max int64) IntegerPropertyBuilder {
	maximum := float64(max)
	p.maxValue = &maximum
	return p
}

// SetDefaultValue - Define the initial value for the property
func (p *integerSchemaProperty) SetDefaultValue(value int64) IntegerPropertyBuilder {
	defaultValue := float64(value)
	p.defaultValue = &defaultValue
	return p
}

/**
  array property datatype builder
*/
// arraySchemaProperty - adds specific info needed for an array schema property
type arraySchemaProperty struct {
	schemaProperty *schemaProperty
	items          []propertyDefinition
	minItems       *uint
	maxItems       *uint
	PropertyBuilder
}

// AddItem - add an item in the array property
func (p *arraySchemaProperty) AddItem(item PropertyBuilder) ArrayPropertyBuilder {
	def, err := item.Build()
	if err == nil {
		p.items = append(p.items, *def)
	} else {
		p.schemaProperty.err = err
	}
	return p
}

// SetMinItems - set the minimum items in the property array
func (p *arraySchemaProperty) SetMinItems(min uint) ArrayPropertyBuilder {
	p.minItems = &min
	return p
}

// SetMaxItems - set the maximum items in the property array
func (p *arraySchemaProperty) SetMaxItems(max uint) ArrayPropertyBuilder {
	if max < 1 {
		p.schemaProperty.err = fmt.Errorf("max array items must be greater than 0")
	} else {
		p.maxItems = &max
	}
	return p
}

// Build - create the propertyDefinition for use in the subscription schema builder
func (p *arraySchemaProperty) Build() (def *propertyDefinition, err error) {
	def, err = p.schemaProperty.Build()
	if err != nil {
		return
	}

	var anyOfItems *anyOfPropertyDefinitions
	if p.items != nil {
		anyOfItems = &anyOfPropertyDefinitions{p.items}
	}

	if p.minItems != nil && p.maxItems != nil && *p.minItems > *p.maxItems {
		return nil, fmt.Errorf("max array items (%d) must be greater than min array items (%d)", p.maxItems, p.minItems)
	}

	def.Items = anyOfItems
	def.MinItems = p.minItems
	def.MaxItems = p.maxItems
	return def, err
}

/**
  object property datatype builder
*/
// objectSchemaProperty - adds specific info needed for an object schema property
type objectSchemaProperty struct {
	schemaProperty *schemaProperty
	properties     map[string]propertyDefinition
	PropertyBuilder
}

// AddProperty - Add a property in the object property
func (p *objectSchemaProperty) AddProperty(property PropertyBuilder) ObjectPropertyBuilder {
	def, err := property.Build()
	if err == nil {
		if p.properties == nil {
			p.properties = make(map[string]propertyDefinition, 0)
		}
		p.properties[def.Name] = *def
	} else {
		p.schemaProperty.err = err
	}
	return p
}

// Build - create the propertyDefinition for use in the subscription schema builder
func (p *objectSchemaProperty) Build() (def *propertyDefinition, err error) {
	def, err = p.schemaProperty.Build()
	if err != nil {
		return
	}

	var requiredProperties []string
	if p.properties != nil {
		for _, property := range p.properties {
			if property.Required {
				requiredProperties = append(requiredProperties, property.Name)
			}
		}
	}

	def.Properties = p.properties
	def.RequiredProperties = requiredProperties
	return def, err
}
