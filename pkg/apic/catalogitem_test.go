package apic

import (
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
	itemID, err := client.getCatalogItemIDForConsumerInstance("00000")
	assert.Nil(t, err)
	assert.Equal(t, "", itemID)
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
