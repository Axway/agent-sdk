package apic

import (
	"encoding/json"
	"net/http"
	"testing"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func commonSetup(t *testing.T) (Client, *api.MockHTTPClient, SubscriptionSchema) {
	svcClient, mockHTTPClient := GetTestServiceClient()
	assert.NotNil(t, svcClient)
	assert.NotNil(t, mockHTTPClient)

	assert.NotNil(t, svcClient.DefaultSubscriptionSchema)

	apiKeySchema := NewSubscriptionSchema("testname")
	apiKeySchema.AddProperty("prop1", "string", "someproperty", "", true, []string{})
	apiKeySchema.AddProperty("prop2", "int", "someproperty2", "", false, []string{})

	schema := apiKeySchema.(*subscriptionSchema)
	assert.Equal(t, 0, len(schema.UniqueKeys))
	apiKeySchema.AddUniqueKey("abc")
	apiKeySchema.AddUniqueKey("def")
	assert.Equal(t, 2, len(schema.UniqueKeys))
	assert.Equal(t, "def", schema.UniqueKeys[1])

	return svcClient, mockHTTPClient, apiKeySchema
}

func TestRegisterSubscriptionSchema(t *testing.T) {
	svcClient, mockHTTPClient, apiKeySchema := commonSetup(t)
	mockHTTPClient.ResponseCode = http.StatusOK
	err := svcClient.RegisterSubscriptionSchema(apiKeySchema)
	assert.NotNil(t, err)

	// this return code should be good
	mockHTTPClient.ResponseCode = http.StatusCreated
	err = svcClient.RegisterSubscriptionSchema(apiKeySchema)
	assert.Nil(t, err)

	serviceClient := svcClient.(*ServiceClient)
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
	svcClient, mockHTTPClient, apiKeySchema := commonSetup(t)

	// this return code should fail
	mockHTTPClient.ResponseCode = http.StatusNoContent
	err := svcClient.UpdateSubscriptionSchema(apiKeySchema)
	assert.NotNil(t, err)

	// this return code should be good
	mockHTTPClient.ResponseCode = http.StatusOK
	err = svcClient.UpdateSubscriptionSchema(apiKeySchema)
	assert.Nil(t, err)
}

func TestContains(t *testing.T) {
	items := []string{"c", "d", "e"}
	b := util.StringArrayContains(items, "b")
	assert.False(t, b)

	b = util.StringArrayContains(items, "c")
	assert.True(t, b)
}

func TestGetProperty(t *testing.T) {
	_, _, schema := commonSetup(t)
	p := schema.GetProperty("prop3")
	assert.Nil(t, p)

	p = schema.GetProperty("prop1")
	assert.NotNil(t, p)
	assert.Equal(t, "someproperty", p.Description)
}

func TestGetProfilePropValue(t *testing.T) {
	svcClient, _, _ := commonSetup(t)
	sc := svcClient.(*ServiceClient)
	def := &v1alpha1.ConsumerSubscriptionDefinition{}
	p := sc.getProfilePropValue(def)
	assert.Nil(t, p)

	props := v1alpha1.ConsumerSubscriptionDefinitionSpecSchemaProperties{
		Key:   profileKey,
		Value: map[string]interface{}{"key1": "value1"},
	}

	def.Spec.Schema.Properties = []v1alpha1.ConsumerSubscriptionDefinitionSpecSchemaProperties{props}
	p = sc.getProfilePropValue(def)
	assert.NotNil(t, p)
	assert.Equal(t, "value1", p["key1"])
}
