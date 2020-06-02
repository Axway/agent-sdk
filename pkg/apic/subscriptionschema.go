package apic

import (
	"encoding/json"
)

// APIServerSubscriptionSchema -
type APIServerSubscriptionSchema struct {
	Properties []CatalogRevisionProperty `json:"properties,omitempty"`
}

// APIServerSubscriptionDefinitionSpec -
type APIServerSubscriptionDefinitionSpec struct {
	Webhooks []string                    `json:"webhooks,omitempty"`
	Schema   APIServerSubscriptionSchema `json:"schema,omitempty"`
}

// SubscriptionSchema -
type SubscriptionSchema interface {
	AddProperty(name, dataType, description, apicRefField string, isRequired bool)
	AddUniqueKey(keyName string)
	rawJSON() (json.RawMessage, error)
}

type subscriptionSchemaPropertyDefinition struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	APICRef     string `json:"x-axway-ref-apic,omitempty"`
}

type subscriptionSchema struct {
	SchemaType        string                                          `json:"type"`
	SchemaVersion     string                                          `json:"$schema"`
	SchemaDescription string                                          `json:"description"`
	Properties        map[string]subscriptionSchemaPropertyDefinition `json:"properties"`
	Required          []string                                        `json:"required,omitempty"`
	UniqueKeys        []string                                        `json:"x-axway-unique-keys,omitempty"`
}

// NewSubscriptionSchema -
func NewSubscriptionSchema() SubscriptionSchema {
	return &subscriptionSchema{
		SchemaType:        "object",
		SchemaVersion:     "http://json-schema.org/draft-04/schema#",
		SchemaDescription: "Subscription specification for authentication",
		Properties:        make(map[string]subscriptionSchemaPropertyDefinition),
		Required:          make([]string, 0),
	}
}

// AddProperty -
func (ss *subscriptionSchema) AddProperty(name, dataType, description, apicRefField string, isRequired bool) {
	ss.Properties[name] = subscriptionSchemaPropertyDefinition{
		Type:        dataType,
		Description: description,
		APICRef:     apicRefField,
	}
	if isRequired {
		ss.Required = append(ss.Required, name)
	}
}

// AddUniqueKey -
func (ss *subscriptionSchema) AddUniqueKey(keyName string) {
	ss.UniqueKeys = append(ss.UniqueKeys, keyName)
}

// rawJSON -
func (ss *subscriptionSchema) rawJSON() (json.RawMessage, error) {
	schemaBuffer, err := json.Marshal(ss)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(schemaBuffer), nil
}
