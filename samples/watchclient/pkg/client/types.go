package client

import "time"

// AuthConfig the auth config
type AuthConfig struct {
	PrivateKey  string        `mapstructure:"private_key"`
	PublicKey   string        `mapstructure:"public_key"`
	KeyPassword string        `mapstructure:"key_password"`
	URL         string        `mapstructure:"url"`
	Audience    string        `mapstructure:"audience"`
	ClientID    string        `mapstructure:"client_id"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// Config the configuration for the Watch client
type Config struct {
	TenantID      string     `mapstructure:"tenant_id"`
	Host          string     `mapstructure:"host"`
	Port          uint32     `mapstructure:"port"`
	Insecure      bool       `mapstructure:"insecure"`
	Auth          AuthConfig `mapstructure:"auth"`
	TopicSelfLink string     `mapstructure:"topic_self_link"`
	Level         string     `mapstructure:"log_level"`
	Format        string     `mapstructure:"log_format"`
}
