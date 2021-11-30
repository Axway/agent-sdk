package apic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"

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
	svcClient, _, apiKeySchema := commonSetup(t)
	serviceClient := svcClient.(*ServiceClient)
	schemaExists := false
	schemaCreated := false
	schemaUpdate := false
	existingWebhook := "webhook1"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b := []byte("")

		if strings.Contains(req.RequestURI, "/consumersubscriptiondefs/"+apiKeySchema.GetSubscriptionName()) {
			existingSchema := v1alpha1.ConsumerSubscriptionDefinition{
				Spec: v1alpha1.ConsumerSubscriptionDefinitionSpec{
					Webhooks: []string{existingWebhook},
				},
			}
			if req.Method == http.MethodGet {
				if schemaExists {
					b, _ = json.Marshal(existingSchema)
				} else {
					rw.WriteHeader(http.StatusNotFound)
				}
			} else {
				// PUT call
				schemaUpdate = true
				rw.WriteHeader(http.StatusOK)
				spec, _ := serviceClient.prepareSubscriptionDefinitionSpec(&existingSchema, apiKeySchema)
				b, _ = serviceClient.marshalSubscriptionDefinition(apiKeySchema.GetSubscriptionName(), spec)
			}
		}
		if req.Method == http.MethodPost && strings.Contains(req.RequestURI, "/consumersubscriptiondefs") {
			schemaCreated = true
			rw.WriteHeader(http.StatusCreated)
			spec, _ := serviceClient.prepareSubscriptionDefinitionSpec(nil, apiKeySchema)
			b, _ = serviceClient.marshalSubscriptionDefinition(apiKeySchema.GetSubscriptionName(), spec)
		}
		// Send response to be tested
		rw.Write(b)
	}))
	// Close the server when test finishes
	defer server.Close()
	cfg := serviceClient.cfg.(*config.CentralConfiguration)
	cfg.URL = server.URL
	serviceClient.apiClient = api.NewClient(nil, "")
	err := svcClient.RegisterSubscriptionSchema(apiKeySchema, false)
	assert.Nil(t, err)
	cachedSchema, err := serviceClient.subscriptionSchemaCache.Get(apiKeySchema.GetSubscriptionName())
	assert.NotNil(t, cachedSchema)
	assert.Contains(t, cachedSchema.(*v1alpha1.ConsumerSubscriptionDefinition).Spec.Webhooks, DefaultSubscriptionWebhookName)
	assert.True(t, schemaCreated)
	assert.False(t, schemaUpdate)

	schemaCreated = false
	serviceClient.subscriptionSchemaCache = cache.New()
	schemaExists = true
	err = svcClient.RegisterSubscriptionSchema(apiKeySchema, false)
	assert.Nil(t, err)
	cachedSchema, err = serviceClient.subscriptionSchemaCache.Get(apiKeySchema.GetSubscriptionName())
	assert.Contains(t, cachedSchema.(*v1alpha1.ConsumerSubscriptionDefinition).Spec.Webhooks, existingWebhook)
	assert.NotNil(t, cachedSchema)
	assert.False(t, schemaCreated)
	assert.False(t, schemaUpdate)

	err = svcClient.RegisterSubscriptionSchema(apiKeySchema, true)
	assert.Nil(t, err)
	cachedSchema, err = serviceClient.subscriptionSchemaCache.Get(apiKeySchema.GetSubscriptionName())
	assert.Contains(t, cachedSchema.(*v1alpha1.ConsumerSubscriptionDefinition).Spec.Webhooks, DefaultSubscriptionWebhookName)
	assert.Contains(t, cachedSchema.(*v1alpha1.ConsumerSubscriptionDefinition).Spec.Webhooks, existingWebhook)
	assert.NotNil(t, cachedSchema)
	assert.False(t, schemaCreated)
	assert.True(t, schemaUpdate)
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
	b := util.StringSliceContains(items, "b")
	assert.False(t, b)

	b = util.StringSliceContains(items, "c")
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
