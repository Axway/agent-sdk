package config

import (
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
	err = cfg.Validate()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.teamID not set in config", err.Error())

	centralConfig.TeamID = "aaa"
	err = cfg.Validate()
	assert.Nil(t, err)

	centralConfig.Mode = Connected
	err = cfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, "Error central.environmentName not set in config", err.Error())

	centralConfig.EnvironmentName = "eee"
	err = cfg.Validate()
	assert.Nil(t, err)

	centralConfig.APIServerVersion = ""
	err = cfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, "Error central.apiServerVersion not set in config", err.Error())

	centralConfig.APIServerVersion = "v1aplha1"

	assert.Equal(t, "aaa/api/unifiedCatalog/v1/catalogItems", cfg.GetCatalogItemsURL())
	assert.Equal(t, "aaa/apis/management/v1aplha1/environments", cfg.GetAPIServerEnvironmentsURL())
	assert.Equal(t, "aaa/apis/management/v1aplha1/environments/eee/apiservices", cfg.GetAPIServerServicesURL())
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
	assert.Equal(t, "Error central.apicDeployment not set in config", err.Error())

	centralConfig.APICDeployment = "aaa"
	err = cfg.Validate()

	assert.NotNil(t, err)
	assert.Equal(t, "Error central.environmentID not set in config", err.Error())

	centralConfig.EnvironmentID = "aaa"
	err = cfg.Validate()
	assert.Nil(t, err)
}
