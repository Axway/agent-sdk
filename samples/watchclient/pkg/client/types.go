package client

import "time"

// AuthConfig the auth config
type AuthConfig struct {
	PrivateKey  string
	PublicKey   string
	KeyPassword string
	URL         string
	Audience    string
	ClientID    string
	Timeout     time.Duration
}

// Config the configuration for the Watch client
type Config struct {
	TenantID      string
	Host          string
	Port          uint32
	Insecure      bool
	Auth          AuthConfig
	TopicSelfLink string
}
