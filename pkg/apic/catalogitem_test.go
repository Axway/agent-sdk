package apic

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCatalogItemIDForConsumerInstance(t *testing.T) {
	client, httpClientMock := GetTestServiceClient()
	wd, _ := os.Getwd()
	httpClientMock.SetResponse(wd+"/testdata/catalogid-for-consumerinstance-good.json", http.StatusOK)
	testID := "e4f19a3173caf7290173e45f3a270f8b"
	itemID, err := client.getCatalogItemIDForConsumerInstance(testID)
	assert.Nil(t, err)
	assert.Equal(t, testID, itemID)

	// no items found
	httpClientMock.SetResponse(wd+"/testdata/catalogid-for-consumerinstance-empty.json", http.StatusOK)
	itemID, err = client.getCatalogItemIDForConsumerInstance("0000")
	assert.NotNil(t, err)

	// bad response
	httpClientMock.SetResponse("", http.StatusBadRequest)
	testID = "e4f19a3173caf7290173e45f3a270f8b"
	itemID, err = client.getCatalogItemIDForConsumerInstance(testID)
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
