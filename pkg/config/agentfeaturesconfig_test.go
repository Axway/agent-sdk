package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultAgentFeaturesConfig(t *testing.T) {
	cfg := NewAgentFeaturesConfiguration()
	agentFeaturesConfig := cfg.(*AgentFeaturesConfiguration)
	assert.True(t, agentFeaturesConfig.ConnectionToCentralEnabled())
	assert.True(t, agentFeaturesConfig.ProcessSystemSignalsEnabled())
	assert.True(t, agentFeaturesConfig.VersionCheckerEnabled())

	cfgValidator, ok := cfg.(IConfigValidator)
	assert.NotNil(t, cfgValidator)
	assert.True(t, ok)

	err := cfgValidator.ValidateCfg()
	assert.NoError(t, err)
}
