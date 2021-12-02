package apic

import (
	"net/http"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/cache"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

// GetTestServiceClient - return a true ServiceClient, but with mocks for tokengetter and the HTTPClient and dummy values
// for various configurations. Has enough other configuration to make the client usable. This function also returns the
// MockHTTPClient so the caller can use it directly if needed, as it is not available directly from ServiceClient in other packages
func GetTestServiceClient() (*ServiceClient, *api.MockHTTPClient) {
	webhook := &corecfg.WebhookConfiguration{
		URL:     "http://foo.bar",
		Headers: "Header=contentType,Value=application/json",
		Secret:  "",
	}

	subscriptionCfg := corecfg.SubscriptionConfiguration{
		Approval: &corecfg.ApprovalConfig{
			SubscriptionApprovalMode:    "webhook",
			SubscriptionApprovalWebhook: webhook,
		},
		Notifications: &corecfg.NotificationConfig{
			Webhook: &corecfg.WebhookConfiguration{
				URL:     "http://bar.foo",
				Headers: "Header=contentType,Value=application/json",
			},
		},
	}

	subscriptionCfg.SetNotificationType(corecfg.NotifySMTP)

	cfg := &corecfg.CentralConfiguration{
		TeamName:     "testteam",
		TenantID:     "112456",
		Mode:         corecfg.PublishToEnvironmentAndCatalog,
		Environment:  "testenvironment",
		PollInterval: 1 * time.Second,
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost:8888",
			Realm:    "Broker",
			ClientID: "dummy",
		},
		SubscriptionConfiguration: &subscriptionCfg,
	}

	apiClient := &api.MockHTTPClient{ResponseCode: http.StatusOK}
	svcClient := &ServiceClient{
		cfg:                                cfg,
		tokenRequester:                     MockTokenGetter,
		subscriptionSchemaCache:            cache.New(),
		teamCache:                          cache.New(),
		apiClient:                          apiClient,
		DefaultSubscriptionApprovalWebhook: webhook,
		DefaultSubscriptionSchema:          NewSubscriptionSchema(cfg.GetEnvironmentName() + SubscriptionSchemaNameSuffix),
	}
	svcClient.subscriptionMgr = newSubscriptionManager(svcClient)
	return svcClient, apiClient
}

// GetTestServiceClientCentralConfiguration - cast and return the CentralConfiguration
func GetTestServiceClientCentralConfiguration(client *ServiceClient) *corecfg.CentralConfiguration {
	return client.cfg.(*corecfg.CentralConfiguration)
}
