package apic

import (
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
		// SubscriptionApprovalWebhook: corecfg.NewWebhookConfig(),
	}
	return &ServiceClient{
		cfg:                       cfg,
		tokenRequester:            MockTokenGetter,
		apiClient:                 &api.MockClient{ResponseCode: 200},
		DefaultSubscriptionSchema: NewSubscriptionSchema(cfg.GetEnvironmentName() + SubscriptionSchemaNameSuffix),
	}

}
func TestGetCatalogItemIDForConsumerInstance(t *testing.T) {
	client := newServiceClient()
	mock := client.apiClient.(*api.MockClient)
	wd, _ := os.Getwd()
	mock.SetResponse(wd+"/testdata/catalogid-for-consumerinstance-good.json", 200)
	testID := "e4f19a3173caf7290173e45f3a270f8b"
	itemID, err := client.getCatalogItemIDForConsumerInstance(testID)
	assert.Nil(t, err)
	assert.Equal(t, testID, itemID)

	itemID, err = client.getCatalogItemIDForConsumerInstance("0000")
	assert.Nil(t, err)
	assert.Equal(t, testID, itemID)
}

func TestGetCatalogItemName(t *testing.T) {
	client := newServiceClient()
	name, err := client.GetCatalogItemName("12345")
	assert.Nil(t, err)
	assert.Equal(t, "", name)
}
func TestGetSubscriptionsForCatalogItem(t *testing.T) {
	client := newServiceClient()
	subscriptions, err := client.getSubscriptionsForCatalogItem(nil, "12345")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(subscriptions))
}
