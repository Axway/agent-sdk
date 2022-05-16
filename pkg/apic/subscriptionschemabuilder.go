package apic

import "github.com/Axway/agent-sdk/pkg/util/log"

// TODO - this file should be able to be removed once Unified Catalog support has been removed

// SubscriptionSchemaBuilder - used to build a subscription schema for API Central
type SubscriptionSchemaBuilder interface {
	Update(update bool) SubscriptionSchemaBuilder
	SetName(name string) SubscriptionSchemaBuilder
	AddProperty(property PropertyBuilder) SubscriptionSchemaBuilder
	AddUniqueKey(keyName string) SubscriptionSchemaBuilder
	SetJSONDraft07SchemaVersion() SubscriptionSchemaBuilder

	Register() error
}

// schemaBuilder - hold all of the details needs to create a subscription schema
type schemaBuilder struct {
	err           error
	name          string
	update        bool
	uniqueKeys    []string
	properties    map[string]SubscriptionSchemaPropertyDefinition
	schemaVersion string
	apicClient    Client
}

// NewSubscriptionSchemaBuilder - Creates a new subscription schema builder
func NewSubscriptionSchemaBuilder(apicClient Client) SubscriptionSchemaBuilder {
	return &schemaBuilder{
		properties:    make(map[string]SubscriptionSchemaPropertyDefinition, 0),
		uniqueKeys:    make([]string, 0),
		apicClient:    apicClient,
		schemaVersion: "http://json-schema.org/draft-04/schema#",
		update:        true,
	}
}

// Update - update the existing schmea (default) or not
func (s *schemaBuilder) Update(update bool) SubscriptionSchemaBuilder {
	s.update = update
	return s
}

// SetName - give the subscription schema a name
func (s *schemaBuilder) SetName(name string) SubscriptionSchemaBuilder {
	s.name = name
	return s
}

// AddProperty - adds a new subscription schema property to the schema
func (s *schemaBuilder) AddProperty(property PropertyBuilder) SubscriptionSchemaBuilder {
	prop, err := property.Build()
	if err == nil {
		s.properties[prop.Name] = *prop
	} else {
		s.err = err
	}
	return s
}

// AddUniqueKey - add a unique key to the schema
func (s *schemaBuilder) AddUniqueKey(keyName string) SubscriptionSchemaBuilder {
	s.uniqueKeys = append(s.uniqueKeys, keyName)
	return s
}

//SetJSONDraft07SchemaVersion - set the JSON schema for the subscription definition to Draft-07
func (s *schemaBuilder) SetJSONDraft07SchemaVersion() SubscriptionSchemaBuilder {
	s.schemaVersion = "http://json-schema.org/draft-07/schema#"
	return s
}

// Register - build and register the subscription schema
func (s *schemaBuilder) Register() error {
	if s.apicClient.(*ServiceClient).cfg.IsMarketplaceSubsEnabled() {
		log.Trace("Registering consumer subscrition definitions is disabled when using Marketplace Provisioning")
		return nil
	}
	if s.err != nil {
		return s.err
	}
	// Create the list of required properties
	required := make([]string, 0)
	for key, value := range s.properties {
		if value.Required {
			required = append(required, key)
		}
	}

	schema := &subscriptionSchema{
		SubscriptionName:  s.name,
		SchemaType:        "object",
		SchemaVersion:     s.schemaVersion,
		SchemaDescription: "Subscription specification for authentication",
		Properties:        s.properties,
		UniqueKeys:        s.uniqueKeys,
		Required:          required,
	}

	return s.apicClient.RegisterSubscriptionSchema(schema, s.update)
}
