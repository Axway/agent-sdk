package provisioning

import (
	"encoding/json"
	"fmt"
)

// SchemaBuilder - used to build a subscription schema for API Central
type SchemaBuilder interface {
	SetName(name string) SchemaBuilder
	SetDescription(description string) SchemaBuilder
	SetPropertyOrder(propertyOrder []string) SchemaBuilder
	AddProperty(property PropertyBuilder) SchemaBuilder
	AddUniqueKey(keyName string) SchemaBuilder
	// Build builds the json schema - this is called automatically by the resource builder
	Build() (map[string]interface{}, error)
}

// schemaBuilder - holds all the details needs to create a subscription schema
type schemaBuilder struct {
	err           error
	name          string
	description   string
	PropertyOrder []string `json:"x-axway-order,omitempty"`
	uniqueKeys    []string
	properties    map[string]propertyDefinition
	schemaVersion string
}

// jsonSchema - the schema generated from the builder
type jsonSchema struct {
	SubscriptionName  string                        `json:"-"`
	SchemaType        string                        `json:"type"`
	SchemaVersion     string                        `json:"$schema"`
	SchemaDescription string                        `json:"description"`
	Properties        map[string]propertyDefinition `json:"properties"`
	Required          []string                      `json:"required,omitempty"`
}

// NewSchemaBuilder - Creates a new subscription schema builder
func NewSchemaBuilder() SchemaBuilder {
	return &schemaBuilder{
		properties:    make(map[string]propertyDefinition, 0),
		uniqueKeys:    make([]string, 0),
		PropertyOrder: make([]string, 0),
		schemaVersion: "http://json-schema.org/draft-07/schema#",
	}
}

// SetName - give the subscription schema a name
func (s *schemaBuilder) SetName(name string) SchemaBuilder {
	s.name = name
	return s
}

// SetDescription - give the subscription schema a description
func (s *schemaBuilder) SetDescription(description string) SchemaBuilder {
	s.description = description
	return s
}

// SetPropertyOrder - Set a list of ordered fields to be rendered in the UI
func (s *schemaBuilder) SetPropertyOrder(propertyOrder []string) SchemaBuilder {
	s.PropertyOrder = propertyOrder
	return s
}

// AddProperty - adds a new subscription schema property to the schema
func (s *schemaBuilder) AddProperty(property PropertyBuilder) SchemaBuilder {
	prop, err := property.Build()
	if err == nil {
		s.properties[prop.Name] = *prop

		// validate property order
		if len(s.PropertyOrder) > 0 {
			if !inPropertyOrder(prop.Name, s.PropertyOrder) {
				s.err = fmt.Errorf("property %s is not found in the property order", prop.Name)
			}
		}
	} else {
		s.err = err
	}
	return s
}

// inPropertyOrder - check to see if the property is in the propertyOrder.
func inPropertyOrder(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

// AddUniqueKey - add a unique key to the schema
func (s *schemaBuilder) AddUniqueKey(keyName string) SchemaBuilder {
	s.uniqueKeys = append(s.uniqueKeys, keyName)
	return s
}

// Register - build and register the subscription schema
func (s *schemaBuilder) Build() (map[string]interface{}, error) {
	if s.err != nil {
		return nil, s.err
	}
	// Create the list of required properties
	required := make([]string, 0)
	for key, value := range s.properties {
		if value.Required {
			required = append(required, key)
		}
	}

	schema := &jsonSchema{
		SubscriptionName:  s.name,
		SchemaType:        "object",
		SchemaVersion:     s.schemaVersion,
		SchemaDescription: s.description,
		Properties:        s.properties,
		Required:          required,
	}

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	schemaMap := map[string]interface{}{}
	err = json.Unmarshal(schemaBytes, &schemaMap)
	if err != nil {
		return nil, err
	}
	return schemaMap, nil
}
