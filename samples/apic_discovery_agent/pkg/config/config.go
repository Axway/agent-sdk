package config

import (
	"errors"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

// AgentConfig - represents the config for agent
type AgentConfig struct {
	CentralCfg corecfg.CentralConfig `config:"central"`
	GatewayCfg *GatewayConfig        `config:"gateway-section"`
}

// GatewayConfig - represents the config for gateway
type GatewayConfig struct {
	corecfg.IConfigValidator
	SpecPath   string `config:"specPath"`
	ConfigKey1 string `config:"config_key_1"`
	ConfigKey2 string `config:"config_key_2"`
	ConfigKey3 string `config:"config_key_3"`
}

// ValidateCfg - Validates the gateway config
func (c *GatewayConfig) ValidateCfg() (err error) {
	if c.SpecPath == "" {
		return errors.New("Invalid gateway configuration: specPath is not configured")
	}

	return
}
