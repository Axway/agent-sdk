package apic

import (
	"net/http"
	"sync"
	"time"

	cache2 "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/util/log"

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

	cfg := &corecfg.CentralConfiguration{
		TeamName:     "testteam",
		TenantID:     "112456",
		Environment:  "testenvironment",
		PollInterval: 1 * time.Second,
		PageSize:     100,
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost:8888",
			Realm:    "Broker",
			ClientID: "dummy",
		},
		CredentialConfig: &corecfg.CredentialConfiguration{},
	}

	apiClient := &api.MockHTTPClient{ResponseCode: http.StatusOK}
	svcClient := &ServiceClient{
		cfg:                                cfg,
		tokenRequester:                     MockTokenGetter,
		subscriptionSchemaCache:            cache.New(),
		caches:                             cache2.NewAgentCacheManager(cfg, false),
		apiClient:                          apiClient,
		DefaultSubscriptionApprovalWebhook: webhook,
		logger:                             log.NewFieldLogger(),
		pageSizes:                          map[string]int{},
		pageSizeMutex:                      &sync.Mutex{},
	}

	return svcClient, apiClient
}

// GetTestServiceClientCentralConfiguration - cast and return the CentralConfiguration
func GetTestServiceClientCentralConfiguration(client *ServiceClient) *corecfg.CentralConfiguration {
	return client.cfg.(*corecfg.CentralConfiguration)
}
