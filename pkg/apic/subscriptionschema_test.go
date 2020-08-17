package apic

import (
	"encoding/json"
	"net/http"
	"testing"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func commonSetup(t *testing.T) (Client, SubscriptionSchema) {
	cfg := &corecfg.CentralConfiguration{
		TeamID: "test",
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost:8888",
			Realm:    "Broker",
			ClientID: "dummy",
		},
		SubscriptionApprovalWebhook: corecfg.NewWebhookConfig(),
	}
	client := New(cfg)
	assert.NotNil(t, client)
	serviceClient := client.(*ServiceClient)
	assert.NotNil(t, serviceClient)

	serviceClient.tokenRequester = MockTokenGetter
	assert.NotNil(t, serviceClient.DefaultSubscriptionSchema)
	passthruSchema := serviceClient.DefaultSubscriptionSchema
	assert.NotNil(t, passthruSchema)

	apiKeySchema := NewSubscriptionSchema("testname")
	apiKeySchema.AddProperty("prop1", "string", "someproperty", "", true, []string{})
	apiKeySchema.AddProperty("prop2", "int", "someproperty2", "", false, []string{})

	schema := apiKeySchema.(*subscriptionSchema)
	assert.Equal(t, 0, len(schema.UniqueKeys))
	apiKeySchema.AddUniqueKey("abc")
	apiKeySchema.AddUniqueKey("def")
	assert.Equal(t, 2, len(schema.UniqueKeys))
	assert.Equal(t, "def", schema.UniqueKeys[1])

	return client, apiKeySchema
}

func TestRegisterSubscriptionSchema(t *testing.T) {
	client, apiKeySchema := commonSetup(t)
	serviceClient := client.(*ServiceClient)
	mock := api.MockClient{ResponseCode: http.StatusOK}
	serviceClient.apiClient = &mock
	err := client.RegisterSubscriptionSchema(apiKeySchema)
	assert.NotNil(t, err)

	// this return code should be good
	mock.ResponseCode = http.StatusCreated
	err = client.RegisterSubscriptionSchema(apiKeySchema)
	assert.Nil(t, err)

	registeredAPIKeySchema := serviceClient.RegisteredSubscriptionSchema
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

func TestUpdateSubscriptionSchema(t *testing.T) {
	client, apiKeySchema := commonSetup(t)
	serviceClient := client.(*ServiceClient)

	// this return code should fail
	mock := api.MockClient{ResponseCode: http.StatusNoContent}
	serviceClient.apiClient = &mock
	err := client.UpdateSubscriptionSchema(apiKeySchema)
	assert.NotNil(t, err)

	// this return code should be good
	mock.ResponseCode = http.StatusOK
	err = client.UpdateSubscriptionSchema(apiKeySchema)
	assert.Nil(t, err)
}
