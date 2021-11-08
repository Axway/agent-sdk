package client

import (
	"github.com/Axway/agent-sdk/pkg/apic/auth"
)

// Config the configuration for the Watch client
type Config struct {
	TenantID      string          `mapstructure:"tenant_id"`
	Host          string          `mapstructure:"host"`
	Port          uint32          `mapstructure:"port"`
	Insecure      bool            `mapstructure:"insecure"`
	Auth          auth.AuthConfig `mapstructure:"auth"`
	TopicSelfLink string          `mapstructure:"topic_self_link"`
	Level         string          `mapstructure:"log_level"`
	Format        string          `mapstructure:"log_format"`
}
