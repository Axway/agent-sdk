package apic

import (
	"net/http"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestNewSubscriptionSchemaBuilder(t *testing.T) {
	builder := NewSubscriptionSchemaBuilder(nil)
	assert.NotNil(t, builder)

	schemaBuilderProps := builder.(*schemaBuilder)

	// test all the default values
	assert.Nil(t, schemaBuilderProps.err)
	assert.Empty(t, schemaBuilderProps.name)
	assert.Empty(t, schemaBuilderProps.properties)
	assert.Len(t, schemaBuilderProps.uniqueKeys, 0)
	assert.Nil(t, schemaBuilderProps.apicClient)
}

func TestSubscriptionSchemaBuilderSetters(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
		{
			RespCode: http.StatusCreated,
		},
	})
	err := NewSubscriptionSchemaBuilder(svcClient).
		SetName("name").
		AddUniqueKey("key").
		AddProperty(NewSubscriptionSchemaPropertyBuilder().
			SetName("name").
			SetRequired().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		Register()

	assert.Nil(t, err)

	svcClient, mockHTTPClient = GetTestServiceClient()
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusCreated,
		},
	})
	err = NewSubscriptionSchemaBuilder(svcClient).
		SetName("name1").
		AddUniqueKey("key").
		AddProperty(NewSubscriptionSchemaPropertyBuilder().
			SetName("name").
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		AddProperty(NewSubscriptionSchemaPropertyBuilder().
			IsString().
			SetEnumValues([]string{"a", "b", "c"})).
		Register()

	assert.NotNil(t, err)
}
