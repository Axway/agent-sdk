package apic

import "fmt"

// SubscriptionPropertyBuilder - used to build a subscription schmea property
type SubscriptionPropertyBuilder interface {
	SetName(name string) SubscriptionPropertyBuilder
	SetDescription(description string) SubscriptionPropertyBuilder
	SetEnumValues(values []string) SubscriptionPropertyBuilder
	AddEnumValue(value string) SubscriptionPropertyBuilder
	SetRequired() SubscriptionPropertyBuilder
	SetReadOnly() SubscriptionPropertyBuilder
	SetHidden() SubscriptionPropertyBuilder
	SetAPICRefField(field string) SubscriptionPropertyBuilder
	IsString() SubscriptionPropertyBuilder
	Build() (*SubscriptionSchemaPropertyDefinition, error)
}

// schemaProperty - holds all the info needed to create a subscrition schema property
type schemaProperty struct {
	SubscriptionPropertyBuilder
	err          error
	name         string
	description  string
	apicRefField string
	enums        map[string]bool
	required     bool
	readOnly     bool
	hidden       bool
	dataType     string
}

// NewSubscriptionSchemaPropertyBuilder - Creates a new subscription schema property builder
func NewSubscriptionSchemaPropertyBuilder() SubscriptionPropertyBuilder {
	return &schemaProperty{
		enums: make(map[string]bool),
	}
}

// SetName - sets the name of the property
func (p *schemaProperty) SetName(name string) SubscriptionPropertyBuilder {
	p.name = name
	return p
}

// SetDescription - set the description of the property
func (p *schemaProperty) SetDescription(description string) SubscriptionPropertyBuilder {
	p.description = description
	return p
}

// SetEnumValues - add a list of enum values to the property
func (p *schemaProperty) SetEnumValues(values []string) SubscriptionPropertyBuilder {
	for _, value := range values {
		p.enums[value] = true
	}
	return p
}

// AddEnumValue - add a new value to the enum list
func (p *schemaProperty) AddEnumValue(value string) SubscriptionPropertyBuilder {
	if _, ok := p.enums[value]; !ok {
		p.enums[value] = true
	}
	return p
}

// SetRequired - set the property as a required field in the schema
func (p *schemaProperty) SetRequired() SubscriptionPropertyBuilder {
	p.required = true
	return p
}

// SetReadOnly - set the property as a read only property
func (p *schemaProperty) SetReadOnly() SubscriptionPropertyBuilder {
	p.readOnly = true
	return p
}

// SetHidden - set the property as a hidden property
func (p *schemaProperty) SetHidden() SubscriptionPropertyBuilder {
	p.hidden = true
	return p
}

// SetAPICRefField - set the apic reference field for this property
func (p *schemaProperty) SetAPICRefField(field string) SubscriptionPropertyBuilder {
	p.apicRefField = field
	return p
}

// IsString - mark the datatype of the property as a string
func (p *schemaProperty) IsString() SubscriptionPropertyBuilder {
	if p.dataType != "" {
		p.err = fmt.Errorf("The data type cannot be set to string, it is already set to %v", p.dataType)
	} else {
		p.dataType = "string"
	}
	return p
}

// Build - create the SubscriptionSchemaPropertyDefinition for use in the subscription schema builder
func (p *schemaProperty) Build() (*SubscriptionSchemaPropertyDefinition, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.name == "" {
		return nil, fmt.Errorf("Cannot add a subscription schema property without a name")
	}

	if p.dataType == "" {
		return nil, fmt.Errorf("Subscription schema property named %v must have a data type", p.name)
	}

	prop := &SubscriptionSchemaPropertyDefinition{
		Name:        p.name,
		Type:        p.dataType,
		Description: p.description,
		APICRef:     p.apicRefField,
		ReadOnly:    p.readOnly,
		Required:    p.required,
	}

	if p.hidden {
		prop.Format = "hidden"
	}
	// Convert map to string array
	if len(p.enums) > 0 {
		list := make([]string, 0)
		for key := range p.enums {
			list = append(list, key)
		}
		prop.Enum = list
	}

	return prop, nil
}
