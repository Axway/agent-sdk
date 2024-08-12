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

	assert.True(t, agentFeaturesConfig.PersistCacheEnabled())

	cfgValidator, ok := cfg.(IConfigValidator)
	assert.NotNil(t, cfgValidator)
	assert.True(t, ok)

	err := cfgValidator.ValidateCfg()
	assert.NoError(t, err)
}
