package apic

import (
	"encoding/json"
	"testing"

	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestSubscriptionSchemaRegistration(t *testing.T) {
	cfg := &corecfg.CentralConfiguration{
		TeamID: "test",
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost:8888",
			Realm:    "Broker",
			ClientID: "dummy",
		},
	}
	client := New(cfg)
	assert.NotNil(t, client)
	serviceClient := client.(*ServiceClient)
	assert.NotNil(t, serviceClient)
	assert.NotEqual(t, 0, len(serviceClient.SubscriptionSchemaMap))
	passthruSchema := serviceClient.SubscriptionSchemaMap[Passthrough]
	assert.NotNil(t, passthruSchema)

	apiKeySchema := NewSubscriptionSchema()
	apiKeySchema.AddProperty("prop1", "string", "someproperty", "", true)
	apiKeySchema.AddProperty("prop2", "int", "someproperty2", "", false)
	client.RegisterSubscriptionSchema(Apikey, apiKeySchema)

	registeredAPIKeySchema := serviceClient.SubscriptionSchemaMap[Apikey]
	assert.NotNil(t, registeredAPIKeySchema)
	rawAPIJson, _ := registeredAPIKeySchema.rawJSON()

	var registeredSchema subscriptionSchema
	json.Unmarshal([]byte(rawAPIJson), &registeredSchema)

	prop1 := registeredSchema.Properties["prop1"]
	assert.NotNil(t, prop1)
	assert.Equal(t, "string", prop1.Type)
	assert.Equal(t, "someproperty", prop1.Description)

	prop2 := registeredSchema.Properties["prop2"]
	assert.NotNil(t, prop2)
	assert.Equal(t, "string", prop1.Type)
	assert.Equal(t, "someproperty2", prop2.Description)

	assert.Contains(t, registeredSchema.Required, "prop1")
}
