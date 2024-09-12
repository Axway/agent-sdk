package watchmanager

import (
	"errors"
)

// Config - represents the config for watch service connection
type Config struct {
	Host        string
	Port        uint32
	TenantID    string
	TokenGetter TokenGetter
	UserAgent   string
}

func (c *Config) validateCfg() error {
	if c == nil {
		return errors.New("watch config not initialized")
	}

	if c.Host == "" {
		return errors.New("invalid watch config: watch service host not specified")
	}

	if c.TenantID == "" {
		return errors.New("invalid watch config: organization ID is not specified")
	}

	if c.TokenGetter == nil {
		return errors.New("invalid watch config: token getter not configured")
	}
	return nil
}
