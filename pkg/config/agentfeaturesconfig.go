package config

import (
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
)

type AgentFeaturesConfig interface {
	ConnectionToCentralEnabled() bool
	ProcessSystemSignalsEnabled() bool
	AgentVersionCheckerEnabled() bool
}

type AgentFeaturesConfiguration struct {
	AgentFeaturesConfig
	IConfigValidator
	ConnectToCentral     bool `config:"connectToCentral"`
	ProcessSystemSignals bool `config:"processSystemSignals"`
	AgentVersionChecker  bool `config:"agentVersionChecker"`
}

// NewCentralConfig - Creates the default central config
func NewAgentFeaturesConfiguration() AgentFeaturesConfig {
	return &AgentFeaturesConfiguration{
		ConnectToCentral:     true,
		ProcessSystemSignals: true,
		AgentVersionChecker:  true,
	}
}

// ConnectionToCentralEnabled -
func (c *AgentFeaturesConfiguration) ConnectionToCentralEnabled() bool {
	return c.ConnectToCentral
}

// ProcessSystemSignalsEnabled -
func (c *AgentFeaturesConfiguration) ProcessSystemSignalsEnabled() bool {
	return c.ProcessSystemSignals
}

// VersionCheckerEnabled -
func (c *AgentFeaturesConfiguration) AgentVersionCheckerEnabled() bool {
	return c.AgentVersionChecker
}

const (
	pathConnectToCentral     = "agentFeatures.connectToCentral"
	pathProcessSystemSignals = "agentFeatures.processSystemSignals"
	pathAgentVersionChecker  = "agentFeatures.agentVersionChecker"
)

// ValidateCfg - Validates the config, implementing IConfigInterface
func (c *AgentFeaturesConfiguration) ValidateCfg() (err error) {
	exception.Block{
		Try: func() {
			c.validateConfig()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()

	return
}

func (c *AgentFeaturesConfiguration) validateConfig() {
}

// AddAgentFeaturesConfigProperties - Adds the command properties needed for Agent Features Config
func AddAgentFeaturesConfigProperties(props properties.Properties) {
	props.AddBoolProperty(pathConnectToCentral, true, "Controls whether the agent SDK connects to Central or not")
	props.AddBoolProperty(pathProcessSystemSignals, true, "Controls whether the agent SDK processes system signals or not")
	props.AddBoolProperty(pathAgentVersionChecker, true, "Controls whether the agent SDK version checker will be enabled or not")
}

// ParseAgentFeaturesConfig - Parses the AgentFeatures Config values from the command line
func ParseAgentFeaturesConfig(props properties.Properties) (AgentFeaturesConfig, error) {
	cfg := &AgentFeaturesConfiguration{
		ConnectToCentral:     props.BoolPropertyValue(pathConnectToCentral),
		ProcessSystemSignals: props.BoolPropertyValue(pathProcessSystemSignals),
		AgentVersionChecker:  props.BoolPropertyValue(pathAgentVersionChecker),
	}

	return cfg, nil
}
