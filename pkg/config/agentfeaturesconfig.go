package config

import (
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
)

// AgentFeaturesConfig - Interface to get agent features Config
type AgentFeaturesConfig interface {
	ConnectionToCentralEnabled() bool
	ProcessSystemSignalsEnabled() bool
	VersionCheckerEnabled() bool
}

// AgentFeaturesConfiguration - Structure to hold the agent features config
type AgentFeaturesConfiguration struct {
	AgentFeaturesConfig
	IConfigValidator
	ConnectToCentral     bool `config:"connectToCentral"`
	ProcessSystemSignals bool `config:"processSystemSignals"`
	VersionChecker       bool `config:"versionChecker"`
}

// NewAgentFeaturesConfiguration - Creates the default agent features config
func NewAgentFeaturesConfiguration() AgentFeaturesConfig {
	return &AgentFeaturesConfiguration{
		ConnectToCentral:     true,
		ProcessSystemSignals: true,
		VersionChecker:       true,
	}
}

// ConnectionToCentralEnabled - True if the agent is a standard agent that connects to Central
func (c *AgentFeaturesConfiguration) ConnectionToCentralEnabled() bool {
	return c.ConnectToCentral
}

// ProcessSystemSignalsEnabled - True if the agent SDK listens for system signals and manages shutdown
func (c *AgentFeaturesConfiguration) ProcessSystemSignalsEnabled() bool {
	return c.ProcessSystemSignals
}

// VersionCheckerEnabled - True if the agent SDK should check for newer versions of the agent.
func (c *AgentFeaturesConfiguration) VersionCheckerEnabled() bool {
	return c.VersionChecker
}

const (
	pathConnectToCentral     = "agentFeatures.connectToCentral"
	pathProcessSystemSignals = "agentFeatures.processSystemSignals"
	pathVersionChecker       = "agentFeatures.versionChecker"
)

// ValidateCfg - Validates the config, implementing IConfigInterface
func (c *AgentFeaturesConfiguration) ValidateCfg() (err error) {
	// No validation required
	return
}

// AddAgentFeaturesConfigProperties - Adds the command properties needed for Agent Features Config
func AddAgentFeaturesConfigProperties(props properties.Properties) {
	props.AddBoolProperty(pathConnectToCentral, true, "Controls whether the agent SDK connects to Central or not")
	props.AddBoolProperty(pathProcessSystemSignals, true, "Controls whether the agent SDK processes system signals or not")
	props.AddBoolProperty(pathVersionChecker, true, "Controls whether the agent SDK version checker will be enabled or not")
}

// ParseAgentFeaturesConfig - Parses the AgentFeatures Config values from the command line
func ParseAgentFeaturesConfig(props properties.Properties) (AgentFeaturesConfig, error) {
	cfg := &AgentFeaturesConfiguration{
		ConnectToCentral:     props.BoolPropertyValueOrTrue(pathConnectToCentral),
		ProcessSystemSignals: props.BoolPropertyValueOrTrue(pathProcessSystemSignals),
		VersionChecker:       props.BoolPropertyValueOrTrue(pathVersionChecker),
	}

	return cfg, nil
}
