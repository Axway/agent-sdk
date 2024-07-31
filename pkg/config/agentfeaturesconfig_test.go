package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAgentFeaturesConfig(t *testing.T) {
	cfg := NewAgentFeaturesConfiguration()
	agentFeaturesConfig := cfg.(*AgentFeaturesConfiguration)
	assert.True(t, agentFeaturesConfig.ConnectionToCentralEnabled())
	assert.True(t, agentFeaturesConfig.ProcessSystemSignalsEnabled())
	assert.True(t, agentFeaturesConfig.VersionCheckerEnabled())

	// TODO - UC deprecation, phase 2
	// Can be removed at the start of UC deprecation, phase 2
	assert.True(t, agentFeaturesConfig.PersistCacheEnabled())
	assert.True(t, agentFeaturesConfig.MarketplaceProvisioningEnabled())

	cfgValidator, ok := cfg.(IConfigValidator)
	assert.NotNil(t, cfgValidator)
	assert.True(t, ok)

	err := cfgValidator.ValidateCfg()
	assert.NoError(t, err)
}
