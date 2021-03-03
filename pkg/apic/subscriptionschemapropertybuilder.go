package apic

import "fmt"

//SubscriptionPropertyBuilder -
type SubscriptionPropertyBuilder interface {
	SetName(name string) SubscriptionPropertyBuilder
	SetDescription(description string) SubscriptionPropertyBuilder
	SetEnumValues(values []string) SubscriptionPropertyBuilder
	AddEnumValue(value string) SubscriptionPropertyBuilder
	SetRequired() SubscriptionPropertyBuilder
	SetReadOnly() SubscriptionPropertyBuilder
	SetAPICRefField(field string) SubscriptionPropertyBuilder
	IsString() SubscriptionPropertyBuilder
	Build() (*SubscriptionSchemaPropertyDefinition, error)
}

type schemaProperty struct {
	SubscriptionPropertyBuilder
	err          error
	name         string
	description  string
	apicRefField string
	enums        []string
	required     bool
	readOnly     bool
	dataType     string
}

// NewSubscriptionSchemaPropertyBuilder - Creates a new subscription schema builder
func NewSubscriptionSchemaPropertyBuilder() SubscriptionPropertyBuilder {
	return &schemaProperty{
		enums: make([]string, 0),
	}
}

func (p *schemaProperty) SetName(name string) SubscriptionPropertyBuilder {
	p.name = name
	return p
}

func (p *schemaProperty) SetDescription(description string) SubscriptionPropertyBuilder {
	p.description = description
	return p
}

func (p *schemaProperty) SetEnumValues(values []string) SubscriptionPropertyBuilder {
	p.enums = values
	return p
}

func (p *schemaProperty) AddEnumValue(value string) SubscriptionPropertyBuilder {
	p.enums = append(p.enums, value)
	return p
}

func (p *schemaProperty) SetRequired() SubscriptionPropertyBuilder {
	p.required = true
	return p
}

func (p *schemaProperty) SetReadOnly() SubscriptionPropertyBuilder {
	p.readOnly = true
	return p
}

func (p *schemaProperty) SetAPICRefField(field string) SubscriptionPropertyBuilder {
	p.apicRefField = field
	return p
}

func (p *schemaProperty) IsString() SubscriptionPropertyBuilder {
	if p.dataType != "" {
		p.err = fmt.Errorf("The data type cannot be set to string, it is already set to %v", p.dataType)
	} else {
		p.dataType = "string"
	}
	return p
}

//SubscriptionSchemaPropertyDefinition
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

	if len(p.enums) > 0 {
		prop.Enum = p.enums
	}

	return prop, nil
}
