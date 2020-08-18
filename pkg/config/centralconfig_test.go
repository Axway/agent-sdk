package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoveryAgentConfig(t *testing.T) {
	cfg := NewCentralConfig(DiscoveryAgent)
	centralConfig := cfg.(*CentralConfiguration)

	// Setup Auth config to ignore auth validation errors for this test
	authCfg := centralConfig.Auth.(*AuthConfiguration)
	authCfg.URL = "test"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "aaaa"
	authCfg.PrivateKey = "pppp"
	authCfg.PublicKey = "kkk"

	err := cfg.Validate()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.tenantID not set in config", err.Error())

	centralConfig.TenantID = "1111"
	err = cfg.Validate()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.url not set in config", err.Error())

	centralConfig.URL = "aaa"
	centralConfig.Mode = PublishToEnvironmentAndCatalog
	err = cfg.Validate()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.environment not set in config", err.Error())

	centralConfig.Environment = "eee"
	err = cfg.Validate()

	centralConfig.APIServerVersion = ""
	err = cfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, "Error central.apiServerVersion not set in config", err.Error())

	centralConfig.APIServerVersion = "v1alpha1"

	assert.Equal(t, "aaa/api/unifiedCatalog/v1/catalogItems", cfg.GetCatalogItemsURL())
	assert.Equal(t, "aaa/apis/management/v1alpha1/environments/eee/apiservices", cfg.GetServicesURL())
}

func TestTraceabilityAgentConfig(t *testing.T) {
	cfg := NewCentralConfig(TraceabilityAgent)
	centralConfig := cfg.(*CentralConfiguration)

	// Setup Auth config to ignore auth validation errors for this test
	authCfg := centralConfig.Auth.(*AuthConfiguration)
	authCfg.URL = "test"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "aaaa"
	authCfg.PrivateKey = "pppp"
	authCfg.PublicKey = "kkk"

	err := cfg.Validate()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.tenantID not set in config", err.Error())

	centralConfig.TenantID = "1111"
	err = cfg.Validate()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.url not set in config", err.Error())

	centralConfig.URL = "aaa"
	err = cfg.Validate()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.environment not set in config", err.Error())

	centralConfig.Environment = "111111"
	err = cfg.Validate()

	assert.Equal(t, "https://platform.axway.com", centralConfig.PlatformURL)

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.apicDeployment not set in config", err.Error())

	centralConfig.APICDeployment = "aaa"
	err = cfg.Validate()

	assert.Nil(t, err)

	centralConfig.ProxyURL = "https://foo.bar:1234"
	err = centralConfig.SetProxyEnvironmentVariable()
	assert.Nil(t, err)
	assert.Equal(t, centralConfig.ProxyURL, os.Getenv("HTTPS_PROXY"))

	centralConfig.ProxyURL = "http://foo1.bar:1234"
	err = centralConfig.SetProxyEnvironmentVariable()
	assert.Nil(t, err)
	assert.Equal(t, centralConfig.ProxyURL, os.Getenv("HTTP_PROXY"))
}
