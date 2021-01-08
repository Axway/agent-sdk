package config

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
)

// StatusConfig - Interface for status config
type StatusConfig interface {
	GetPort() int
	GetHealthCheckPeriod() time.Duration
	GetHealthCheckInterval() time.Duration
	ValidateCfg() error
}

// StatusConfiguration -
type StatusConfiguration struct {
	StatusConfig
	Port                int           `config:"port"`
	HealthCheckPeriod   time.Duration `config:"healthCheckPeriod"`
	HealthCheckInterval time.Duration `config:"healthCheckInterval"` // this for binary agents only
}

// NewStatusConfig - create a new status config
func NewStatusConfig() StatusConfig {
	return &StatusConfiguration{
		Port:                8989,
		HealthCheckPeriod:   3 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

// GetPort - Returns the status port
func (a *StatusConfiguration) GetPort() int {
	return a.Port
}

// GetHealthCheckPeriod - Returns the timeout before exiting discovery agent
func (a *StatusConfiguration) GetHealthCheckPeriod() time.Duration {
	return a.HealthCheckPeriod
}

// GetHealthCheckInterval - Returns the interval between running periodic health checks (binary agents only)
func (a *StatusConfiguration) GetHealthCheckInterval() time.Duration {
	return a.HealthCheckInterval
}

const (
	pathPort                = "status.port"
	pathHealthcheckPeriod   = "status.healthCheckPeriod"
	pathHealthcheckInterval = "status.healthCheckInterval"
)

// AddStatusConfigProperties - Adds the command properties needed for Status Config
func AddStatusConfigProperties(props properties.Properties) {
	props.AddIntProperty(pathPort, 8989, "The port that will serve the status endpoints")
	props.AddDurationProperty(pathHealthcheckPeriod, 3*time.Minute, "Time in minutes allotted for services to be ready before exiting discovery agent")
	props.AddDurationProperty(pathHealthcheckInterval, 30*time.Second, "Time between running periodic health checker. Can be between 30 seconds and 5 minutes (binary agents only)")
	props.AddBoolFlag("status", "Get the status of all the Health Checks")
}

// ParseStatusConfig - Parses the Status Config values form teh command line
func ParseStatusConfig(props properties.Properties) (StatusConfig, error) {
	cfg := &StatusConfiguration{
		Port:                props.IntPropertyValue(pathPort),
		HealthCheckPeriod:   props.DurationPropertyValue(pathHealthcheckPeriod),
		HealthCheckInterval: props.DurationPropertyValue(pathHealthcheckInterval),
	}
	return cfg, nil
}

// ValidateCfg - Validates the config, implementing IConfigInterface
func (a *StatusConfiguration) ValidateCfg() error {
	mins := a.GetHealthCheckPeriod().Minutes()
	if mins < 1 || mins > 5 {
		return ErrStatusHealthCheckPeriod
	}

	secs := a.GetHealthCheckInterval().Seconds()
	if secs < 30 || secs > 300 {
		return ErrStatusHealthCheckInterval
	}
	return nil
}
