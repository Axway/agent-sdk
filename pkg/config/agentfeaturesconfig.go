package config

import (
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
)

// AgentFeaturesConfig - Interface to get agent features Config
type AgentFeaturesConfig interface {
	ConnectionToCentralEnabled() bool
	ProcessSystemSignalsEnabled() bool
	VersionCheckerEnabled() bool
	PersistCacheEnabled() bool
	MarketplaceProvisioningEnabled() bool
	GetExternalIDPConfig() ExternalIDPConfig
}

// AgentFeaturesConfiguration - Structure to hold the agent features config
type AgentFeaturesConfiguration struct {
	AgentFeaturesConfig
	IConfigValidator
	ConnectToCentral        bool              `config:"connectToCentral"`
	ProcessSystemSignals    bool              `config:"processSystemSignals"`
	VersionChecker          bool              `config:"versionChecker"`
	PersistCache            bool              `config:"persistCache"`
	MarketplaceProvisioning bool              `config:"marketplaceProvisioning"`
	ExternalIDPConfig       ExternalIDPConfig `config:"idp"`
}

// NewAgentFeaturesConfiguration - Creates the default agent features config
func NewAgentFeaturesConfiguration() AgentFeaturesConfig {
	return &AgentFeaturesConfiguration{
		ConnectToCentral:        true,
		ProcessSystemSignals:    true,
		VersionChecker:          true,
		PersistCache:            false,
		MarketplaceProvisioning: false,
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

// PersistCacheEnabled - True if the agent SDK should use persistence for agent cache.
func (c *AgentFeaturesConfiguration) PersistCacheEnabled() bool {
	return c.PersistCache
}

// MarketplaceProvisioningEnabled - True if the agent SDK should handle marketplace subscriptions.
func (c *AgentFeaturesConfiguration) MarketplaceProvisioningEnabled() bool {
	return c.MarketplaceProvisioning
}

// GetExternalIDPConfig - returns the config for external IdP providers
func (c *AgentFeaturesConfiguration) GetExternalIDPConfig() ExternalIDPConfig {
	return c.ExternalIDPConfig
}

const (
	pathConnectToCentral        = "agentFeatures.connectToCentral"
	pathProcessSystemSignals    = "agentFeatures.processSystemSignals"
	pathVersionChecker          = "agentFeatures.versionChecker"
	pathPersistCache            = "agentFeatures.persistCache"
	pathMarketplaceProvisioning = "agentFeatures.marketplaceProvisioning"
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
	props.AddBoolProperty(pathPersistCache, false, "Controls whether the agent SDK will persist agent cache or not")
	props.AddBoolProperty(pathMarketplaceProvisioning, false, "Controls whether the agent should handle Marketplace Subscriptions or not")
	addExternalIDPProperties(props)
}

// ParseAgentFeaturesConfig - Parses the AgentFeatures Config values from the command line
func ParseAgentFeaturesConfig(props properties.Properties) (AgentFeaturesConfig, error) {
	cfg := &AgentFeaturesConfiguration{
		ConnectToCentral:        props.BoolPropertyValueOrTrue(pathConnectToCentral),
		ProcessSystemSignals:    props.BoolPropertyValueOrTrue(pathProcessSystemSignals),
		VersionChecker:          props.BoolPropertyValueOrTrue(pathVersionChecker),
		PersistCache:            props.BoolPropertyValueOrTrue(pathPersistCache),
		MarketplaceProvisioning: props.BoolPropertyValueOrTrue(pathMarketplaceProvisioning),
	}
	externalIDPCfg, err := parseExternalIDPConfig(props)
	if err != nil {
		return nil, err
	}
	cfg.ExternalIDPConfig = externalIDPCfg
	return cfg, nil
}
