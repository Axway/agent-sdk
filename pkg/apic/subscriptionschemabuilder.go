package apic

//SubscriptionSchemaBuilder -
type SubscriptionSchemaBuilder interface {
	SetName(name string) SubscriptionSchemaBuilder
	AddProperty(property SubscriptionPropertyBuilder) SubscriptionSchemaBuilder
	AddUniqueKey(keyName string) SubscriptionSchemaBuilder

	Register() error
}

type schemaBuilder struct {
	err        error
	name       string
	uniqueKeys []string
	properties map[string]SubscriptionSchemaPropertyDefinition
	apicClient Client
}

// NewSubscriptionSchemaBuilder - Creates a new subscription schema builder
func NewSubscriptionSchemaBuilder(apicClient Client) SubscriptionSchemaBuilder {
	return &schemaBuilder{
		properties: make(map[string]SubscriptionSchemaPropertyDefinition, 0),
		uniqueKeys: make([]string, 0),
		apicClient: apicClient,
	}
}

func (s *schemaBuilder) SetName(name string) SubscriptionSchemaBuilder {
	s.name = name
	return s
}

func (s *schemaBuilder) AddProperty(property SubscriptionPropertyBuilder) SubscriptionSchemaBuilder {
	prop, err := property.Build()
	if err == nil {
		s.properties[prop.Name] = *prop
	} else {
		s.err = err
	}
	return s
}

func (s *schemaBuilder) AddUniqueKey(keyName string) SubscriptionSchemaBuilder {
	s.uniqueKeys = append(s.uniqueKeys, keyName)
	return s
}

func (s *schemaBuilder) Register() error {
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
		SchemaVersion:     "http://json-schema.org/draft-04/schema#",
		SchemaDescription: "Subscription specification for authentication",
		Properties:        s.properties,
		UniqueKeys:        s.uniqueKeys,
		Required:          required,
	}

	return s.apicClient.RegisterSubscriptionSchema(schema)
}
