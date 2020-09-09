package apic

import (
	"net/http"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
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

	subscriptions := corecfg.SubscriptionConfiguration{
		Approval: &corecfg.ApprovalConfig{
			SubscriptionApprovalMode:    "webhook",
			SubscriptionApprovalWebhook: webhook,
		},
	}

	cfg := &corecfg.CentralConfiguration{
		Mode:        corecfg.PublishToEnvironmentAndCatalog,
		Environment: "testenvironment",
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost:8888",
			Realm:    "Broker",
			ClientID: "dummy",
		},
		SubscriptionConfiguration: &subscriptions,
	}

	apiClient := &api.MockHTTPClient{ResponseCode: http.StatusOK}
	svcClient := &ServiceClient{
		cfg:                                cfg,
		tokenRequester:                     MockTokenGetter,
		apiClient:                          apiClient,
		DefaultSubscriptionApprovalWebhook: webhook,
		DefaultSubscriptionSchema:          NewSubscriptionSchema(cfg.GetEnvironmentName() + SubscriptionSchemaNameSuffix),
	}
	svcClient.subscriptionMgr = newSubscriptionManager(svcClient)
	return svcClient, apiClient
}
