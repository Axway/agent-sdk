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

func TestManageIDPResources(t *testing.T) {
	tests := map[string]struct {
		enabled  bool
		expected bool
	}{
		"disabled returns false": {enabled: false, expected: false},
		"enabled returns true":   {enabled: true, expected: true},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			cfg := &AgentFeaturesConfiguration{IDPResourceMgmt: tc.enabled}
			assert.Equal(t, tc.expected, cfg.ManageIDPResources())
		})
	}
}
