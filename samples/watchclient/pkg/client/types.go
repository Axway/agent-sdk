package client

import (
	"github.com/Axway/agent-sdk/pkg/apic/auth"
)

// Config the configuration for the Watch client
type Config struct {
	TenantID          string      `mapstructure:"tenant_id"`
	Host              string      `mapstructure:"host"`
	Port              uint32      `mapstructure:"port"`
	UseHarvester      bool        `mapstructure:"use_harvester"`
	HarvesterHost     string      `mapstructure:"harvester_host"`
	HarvesterPort     uint32      `mapstructure:"harvester_port"`
	HarvesterProtocol string      `mapstructure:"harvester_protocol"`
	Insecure          bool        `mapstructure:"insecure"`
	Auth              auth.Config `mapstructure:"auth"`
	TopicSelfLink     string      `mapstructure:"topic_self_link"`
	Level             string      `mapstructure:"log_level"`
	Format            string      `mapstructure:"log_format"`
}
