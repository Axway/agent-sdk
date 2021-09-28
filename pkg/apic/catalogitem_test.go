package apic

import (
	"net/http"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestGetCatalogItemIDForConsumerInstance(t *testing.T) {
	client, httpClientMock := GetTestServiceClient()
	wd, _ := os.Getwd()
	httpClientMock.SetResponse(wd+"/testdata/catalogid-for-consumerinstance-good.json", http.StatusOK)
	testID := "e4f19a3173caf7290173e45f3a270f8b"
	itemID, err := client.GetCatalogItemIDForConsumerInstance(testID)
	assert.Nil(t, err)
	assert.Equal(t, testID, itemID)

	// no items found
	httpClientMock.SetResponse(wd+"/testdata/catalogid-for-consumerinstance-empty.json", http.StatusOK)
	itemID, err = client.GetCatalogItemIDForConsumerInstance("0000")
	assert.NotNil(t, err)

	// bad response
	httpClientMock.SetResponse("", http.StatusBadRequest)
	testID = "e4f19a3173caf7290173e45f3a270f8b"
	itemID, err = client.GetCatalogItemIDForConsumerInstance(testID)
	assert.NotNil(t, err)
	assert.Equal(t, "", itemID)
}

func TestGetCatalogItemName(t *testing.T) {
	client, httpClientMock := GetTestServiceClient()
	wd, _ := os.Getwd()
	httpClientMock.SetResponse(wd+"/testdata/catalogitem-good.json", http.StatusOK)
	testID := "e4f19a3173caf7290173e45f3a270f8b"
	name, err := client.GetCatalogItemName(testID)
	assert.Nil(t, err)
	assert.Equal(t, "DaleAPI (V7)", name)

	// no item found
	httpClientMock.SetResponse(wd+"/testdata/catalogitem-notexist.json", http.StatusNotFound)
	name, err = client.GetCatalogItemName("0000")
	assert.NotNil(t, err)
	assert.Equal(t, "", name)
}

func TestGetSubscriptionsForCatalogItem(t *testing.T) {
	client, httpClientMock := GetTestServiceClient()
	wd, _ := os.Getwd()
	httpClientMock.SetResponse(wd+"/testdata/subscriptions-for-catalog.json", http.StatusOK)
	testID := "e4f19a3173caf7290173e45f3a270f8b"
	subscriptions, err := client.getSubscriptionsForCatalogItem([]string{string(SubscriptionActive)}, testID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(subscriptions))

	// no item found
	httpClientMock.SetResponse(wd+"/testdata/catalogitem-notexist.json", http.StatusNotFound)
	subscriptions, err = client.GetSubscriptionsForCatalogItem(nil, "0000")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(subscriptions))

	// no item found
	httpClientMock.SetResponse("", http.StatusBadRequest)
	subscriptions, err = client.GetSubscriptionsForCatalogItem(nil, "0000")
	assert.NotNil(t, err)
	assert.Nil(t, subscriptions)
}

func TestGetSubscriptionDefinitionPropertiesForCatalogItem(t *testing.T) {
	client, httpClientMock := GetTestServiceClient()
	wd, _ := os.Getwd()

	// fail
	httpClientMock.SetResponses([]api.MockResponse{
		{
			ErrString: "Resource Not Found",
		},
	})
	schema, err := client.getSubscriptionDefinitionPropertiesForCatalogItem("id", "profile")
	assert.NotNil(t, err)
	assert.Nil(t, schema)

	// fail
	httpClientMock.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
	})
	schema, err = client.getSubscriptionDefinitionPropertiesForCatalogItem("id", "profile")
	assert.NotNil(t, err)
	assert.Nil(t, schema)

	// success
	httpClientMock.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/catalogitemsubscriptiondefprofile.json",
			RespCode: http.StatusOK,
		},
	})
	schema, err = client.getSubscriptionDefinitionPropertiesForCatalogItem("id", "profile")
	assert.Nil(t, err)
	assert.NotNil(t, schema)
	prop := schema.GetProperty("appName")
	assert.NotNil(t, prop)

	assert.True(t, arrContains(prop.Enum, "DaleApp"))
	assert.False(t, arrContains(prop.Enum, "FooApp"))
}

func TestUpdateSubscriptionDefinitionPropertiesForCatalogItem(t *testing.T) {
	client, httpClientMock := GetTestServiceClient()
	wd, _ := os.Getwd()

	// get a schema
	httpClientMock.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/catalogitemsubscriptiondefprofile.json",
			RespCode: http.StatusOK,
		},
	})
	schema, err := client.getSubscriptionDefinitionPropertiesForCatalogItem("id", "profile")
	assert.Nil(t, err)
	assert.NotNil(t, schema)

	// fail
	httpClientMock.SetResponses([]api.MockResponse{
		{
			ErrString: "Resource Not Found",
		},
	})
	err = client.updateSubscriptionDefinitionPropertiesForCatalogItem("id", "profile", schema)
	assert.NotNil(t, err)

	// fail
	httpClientMock.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
	})
	err = client.updateSubscriptionDefinitionPropertiesForCatalogItem("id", "profile", schema)
	assert.NotNil(t, err)

	// success
	httpClientMock.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusOK,
		},
	})
	err = client.updateSubscriptionDefinitionPropertiesForCatalogItem("id", "profile", schema)
	assert.Nil(t, err)
}

func TestCreateCategory(t *testing.T) {
	client, httpClientMock := GetTestServiceClient()
	wd, _ := os.Getwd()

	// success
	httpClientMock.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/category.json",
			RespCode: http.StatusCreated,
		},
	})
	categoryResource, err := client.CreateCategory("CategoryC")
	assert.Nil(t, err)
	assert.NotNil(t, categoryResource)

	// fail
	httpClientMock.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/empty-list.json",
			RespCode: http.StatusNotFound,
		},
	})
	categoryResource, err = client.CreateCategory("CategoryC")
	assert.NotNil(t, err)
	assert.Nil(t, categoryResource)
}
