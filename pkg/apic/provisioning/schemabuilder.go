package provisioning

import (
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// propertyOrderSet - global flag to check if property order was set
var propertyOrderSet = false

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
	propertyOrder []string
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
	PropertyOrder     []string                      `json:"x-axway-order,omitempty"`
	Required          []string                      `json:"required,omitempty"`
}

// NewSchemaBuilder - Creates a new subscription schema builder
func NewSchemaBuilder() SchemaBuilder {
	return &schemaBuilder{
		properties:    make(map[string]propertyDefinition, 0),
		uniqueKeys:    make([]string, 0),
		propertyOrder: make([]string, 0),
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
	// If property names in the property order is bogus, it will be ignored when rendered
	s.propertyOrder = propertyOrder
	propertyOrderSet = true
	return s
}

// AddProperty - adds a new subscription schema property to the schema
func (s *schemaBuilder) AddProperty(property PropertyBuilder) SchemaBuilder {
	prop, err := property.Build()
	if err == nil {
		s.properties[prop.Name] = *prop
		// If property order wasn't set, add property as they come in
		if !propertyOrderSet {
			s.propertyOrder = append(s.propertyOrder, prop.Name)
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

	// validate property order
	for _, value := range s.properties {
		if len(s.propertyOrder) > 0 {
			// if property is not in the set property order, warn
			if !inPropertyOrder(value.Name, s.propertyOrder) {
				log.Warnf("property %s is not found in the property order", value.Name)
			}
		}

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
		PropertyOrder:     s.propertyOrder,
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

	// Set property set bool back to false for next schema builder
	propertyOrderSet = false

	return schemaMap, nil
}
