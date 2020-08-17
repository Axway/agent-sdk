package apic

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
)

func newServiceClient() *ServiceClient {
	cfg := &corecfg.CentralConfiguration{
		// TeamID: "test",
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost:8888",
			Realm:    "Broker",
			ClientID: "dummy",
		},
	}
	return &ServiceClient{
		cfg:                       cfg,
		tokenRequester:            MockTokenGetter,
		apiClient:                 &api.MockClient{ResponseCode: http.StatusOK},
		DefaultSubscriptionSchema: NewSubscriptionSchema(cfg.GetEnvironmentName() + SubscriptionSchemaNameSuffix),
	}
}

func TestGetCatalogItemIDForConsumerInstance(t *testing.T) {
	client := newServiceClient()
	mock := client.apiClient.(*api.MockClient)
	wd, _ := os.Getwd()
	mock.SetResponse(wd+"/testdata/catalogid-for-consumerinstance-good.json", http.StatusOK)
	testID := "e4f19a3173caf7290173e45f3a270f8b"
	itemID, err := client.getCatalogItemIDForConsumerInstance(testID)
	assert.Nil(t, err)
	assert.Equal(t, testID, itemID)

	// no items found
	mock.SetResponse(wd+"/testdata/catalogid-for-consumerinstance-empty.json", http.StatusOK)
	itemID, err = client.getCatalogItemIDForConsumerInstance("0000")
	assert.NotNil(t, err)

	// bad response
	mock.SetResponse("", http.StatusBadRequest)
	testID = "e4f19a3173caf7290173e45f3a270f8b"
	itemID, err = client.getCatalogItemIDForConsumerInstance(testID)
	assert.NotNil(t, err)
	assert.Equal(t, "", itemID)
}

func TestGetCatalogItemName(t *testing.T) {
	client := newServiceClient()
	mock := client.apiClient.(*api.MockClient)
	wd, _ := os.Getwd()
	mock.SetResponse(wd+"/testdata/catalogitem-good.json", http.StatusOK)
	testID := "e4f19a3173caf7290173e45f3a270f8b"
	name, err := client.GetCatalogItemName(testID)
	assert.Nil(t, err)
	assert.Equal(t, "DaleAPI (V7)", name)

	// no item found
	mock.SetResponse(wd+"/testdata/catalogitem-notexist.json", http.StatusNotFound)
	name, err = client.GetCatalogItemName("0000")
	assert.NotNil(t, err)
	assert.Equal(t, "", name)
}

func TestGetSubscriptionsForCatalogItem(t *testing.T) {
	client := newServiceClient()
	mock := client.apiClient.(*api.MockClient)
	wd, _ := os.Getwd()
	mock.SetResponse(wd+"/testdata/subscriptions-for-catalog.json", http.StatusOK)
	testID := "e4f19a3173caf7290173e45f3a270f8b"
	subscriptions, err := client.getSubscriptionsForCatalogItem([]string{string(SubscriptionActive)}, testID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(subscriptions))

	// no item found
	mock.SetResponse(wd+"/testdata/catalogitem-notexist.json", http.StatusNotFound)
	subscriptions, err = client.GetSubscriptionsForCatalogItem(nil, "0000")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(subscriptions))

	// no item found
	mock.SetResponse("", http.StatusBadRequest)
	subscriptions, err = client.GetSubscriptionsForCatalogItem(nil, "0000")
	assert.NotNil(t, err)
	assert.Nil(t, subscriptions)
}
