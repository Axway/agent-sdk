package stream

import (
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/sirupsen/logrus"
)

// Config the configuration for the Watch client
type Config struct {
	Auth     auth.TokenGetter `mapstructure:"auth"`
	Host     string           `mapstructure:"host"`
	Insecure bool             `mapstructure:"insecure"`
	Port     uint32           `mapstructure:"port"`
	TenantID string           `mapstructure:"tenant_id"`
}

// NewWatchManager creates a Manager for handling stream events.
func NewWatchManager(config *Config, logger logrus.FieldLogger) (wm.Manager, error) {
	entry := logger.WithField("package", "client")

	var watchOptions []wm.Option
	watchOptions = append(watchOptions, wm.WithLogger(entry))
	if config.Insecure {
		watchOptions = append(watchOptions, wm.WithTLSConfig(nil))
	}

	cfg := &wm.Config{
		Host:        config.Host,
		Port:        config.Port,
		TenantID:    config.TenantID,
		TokenGetter: config.Auth.GetToken,
	}

	return wm.New(cfg, logger, watchOptions...)
}
