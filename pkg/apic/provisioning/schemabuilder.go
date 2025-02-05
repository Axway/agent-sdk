package provisioning

import (
	"encoding/json"
	"sort"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// SchemaBuilder - used to build a subscription schema for API Central
type SchemaParser interface {
	Parse(schemaBytes []byte) (map[string]PropertyDefinition, error)
}

type schemaParser struct {
}

// NewSchemaBuilder - Creates a new subscription schema builder
func NewSchemaParser() SchemaParser {
	return &schemaParser{}
}

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
	err              error
	name             string
	description      string
	propertyOrder    []string
	uniqueKeys       []string
	properties       map[string]propertyDefinition
	dependencies     map[string]*oneOfPropertyDefinitions
	schemaVersion    string
	propertyOrderSet bool
}

// jsonSchema - the schema generated from the builder
type jsonSchema struct {
	SubscriptionName  string                               `json:"-"`
	SchemaType        string                               `json:"type"`
	SchemaVersion     string                               `json:"$schema"`
	SchemaDescription string                               `json:"description"`
	Properties        map[string]propertyDefinition        `json:"properties"`
	Dependencies      map[string]*oneOfPropertyDefinitions `json:"dependencies,omitempty"`
	PropertyOrder     []string                             `json:"x-axway-order,omitempty"`
	Required          []string                             `json:"required,omitempty"`
}

// NewSchemaBuilder - Creates a new subscription schema builder
func NewSchemaBuilder() SchemaBuilder {
	return &schemaBuilder{
		properties:       make(map[string]propertyDefinition, 0),
		dependencies:     make(map[string]*oneOfPropertyDefinitions),
		uniqueKeys:       make([]string, 0),
		propertyOrder:    make([]string, 0),
		propertyOrderSet: false,
		schemaVersion:    "http://json-schema.org/draft-07/schema#",
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
	s.propertyOrderSet = true
	return s
}

// AddProperty - adds a new subscription schema property to the schema
func (s *schemaBuilder) AddProperty(property PropertyBuilder) SchemaBuilder {
	prop, err := property.Build()
	if err != nil {
		s.err = err
		return s
	}

	s.properties[prop.Name] = *prop

	// If property order wasn't set, add property as they come in
	if !s.propertyOrderSet {
		s.propertyOrder = append(s.propertyOrder, prop.Name)
	}

	dep, err := property.BuildDependencies()
	if err != nil {
		s.err = err
	}
	if dep != nil {
		s.dependencies[prop.Name] = dep
	}

	return s
}

// inList - check to see if the string is in the list.
func inList(value string, list []string) bool {
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

	// validate that the property added is in the property order set by the implementation
	for _, value := range s.properties {
		if len(s.propertyOrder) > 0 {
			// if property is not in the set property order, warn
			if !inList(value.Name, s.propertyOrder) {
				log.Warnf("property %s is not found in the property order", value.Name)
			}
		}
	}

	// validate that the properties in the property order were added
	// and that all props in property order are only in once
	if len(s.propertyOrder) > 0 {
		newOrder := []string{}
		props := map[string]struct{}{}
		for _, orderedProperty := range s.propertyOrder {
			if _, ok := s.properties[orderedProperty]; !ok {
				log.Warnf("ordered property %s, was not added as a property", orderedProperty)
			}

			if _, ok := props[orderedProperty]; !ok {
				newOrder = append(newOrder, orderedProperty)
				props[orderedProperty] = struct{}{}
			}
		}
		s.propertyOrder = newOrder
	}

	// Create the list of required properties
	required := make([]string, 0)
	for key, value := range s.properties {
		if value.Required {
			required = append(required, key)
		}
	}
	sort.Strings(required)
	schema := &jsonSchema{
		SubscriptionName:  s.name,
		SchemaType:        "object",
		SchemaVersion:     s.schemaVersion,
		SchemaDescription: s.description,
		Properties:        s.properties,
		Dependencies:      s.dependencies,
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

	return schemaMap, nil
}

func (s *schemaParser) Parse(schemaBytes []byte) (map[string]PropertyDefinition, error) {
	schema := &jsonSchema{}
	err := json.Unmarshal(schemaBytes, schema)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]PropertyDefinition)
	for s, p := range schema.Properties {
		buf, _ := json.Marshal(p)
		newprop := &propertyDefinition{}
		json.Unmarshal(buf, newprop)
		ret[s] = newprop
	}
	return ret, nil
}
