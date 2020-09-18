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

	cfgValidator, ok := cfg.(IConfigValidator)
	assert.True(t, ok)
	assert.NotNil(t, cfgValidator)

	err := cfgValidator.ValidateCfg()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.organizationID not set in config", err.Error())

	centralConfig.TenantID = "1111"
	err = cfgValidator.ValidateCfg()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.url not set in config", err.Error())

	centralConfig.URL = "aaa"
	centralConfig.Mode = PublishToEnvironmentAndCatalog
	err = cfgValidator.ValidateCfg()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.environment not set in config", err.Error())

	centralConfig.Environment = "eee"
	err = cfgValidator.ValidateCfg()

	centralConfig.APIServerVersion = ""
	err = cfgValidator.ValidateCfg()
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

	cfgValidator, ok := cfg.(IConfigValidator)
	assert.True(t, ok)
	assert.NotNil(t, cfgValidator)
	err := cfgValidator.ValidateCfg()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.organizationID not set in config", err.Error())

	centralConfig.TenantID = "1111"
	err = cfgValidator.ValidateCfg()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.url not set in config", err.Error())

	centralConfig.URL = "aaa"
	err = cfgValidator.ValidateCfg()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.environment not set in config", err.Error())

	centralConfig.Environment = "111111"
	err = cfgValidator.ValidateCfg()

	assert.Equal(t, "https://platform.axway.com", centralConfig.PlatformURL)

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.apicDeployment not set in config", err.Error())

	centralConfig.APICDeployment = "aaa"
	err = cfgValidator.ValidateCfg()

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

func TestTeamConfig(t *testing.T) {
	cfg := NewCentralConfig(TraceabilityAgent)
	centralConfig := cfg.(*CentralConfiguration)

	// Setup Auth config to ignore auth validation errors for this test
	authCfg := centralConfig.Auth.(*AuthConfiguration)
	authCfg.URL = "test"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "aaaa"
	authCfg.PrivateKey = "pppp"
	authCfg.PublicKey = "kkk"
	centralConfig.TenantID = "1111"
	centralConfig.URL = "aaa"
	centralConfig.Environment = "111111"
	centralConfig.APICDeployment = "aaa"

	// Should be nil, not yet set
	assert.Equal(t, "", centralConfig.GetTeamID(), "Team ID was expected to be blank as it has not yet been set")
	
	//Set it and validate
	teamID := "abc12:34567:def89:12345:67890"
	centralConfig.SetTeamID(teamID)
	assert.Equal(t, teamID, centralConfig.GetTeamID(), "The Team ID was not set appropriately")
}
