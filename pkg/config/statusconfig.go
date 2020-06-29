package config

import (
	"time"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/cmd/properties"
)

const (
	statusPortDefault              = 8989
	statusHealthCheckPeriodDefault = 3 * time.Minute
)

// AddStatusConfigProperties -
func AddStatusConfigProperties(cmdProps properties.Properties) {
	// Status
	cmdProps.AddIntProperty("status.port", "statusPort", statusPortDefault, "The port that will serve the status endpoints")
	cmdProps.AddDurationProperty("status.healthCheckPeriod", "statusHealthCheckPeriod", statusHealthCheckPeriodDefault, "Time in minutes allotted for services to be ready before exiting discovery agent")
	cmdProps.AddBoolFlag("status", "Get the status of all the Health Checks")
}

// ParseStatusConfig -
func ParseStatusConfig(cmdProps properties.Properties) (StatusConfig, error) {
	cfg := &StatusConfiguration{
		Port:              cmdProps.IntPropertyValue("status.port"),
		HealthCheckPeriod: cmdProps.DurationPropertyValue("status.healthCheckPeriod"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// StatusConfig - Interface for status config
type StatusConfig interface {
	GetPort() int
	GetHealthCheckPeriod() time.Duration
}

// StatusConfiguration -
type StatusConfiguration struct {
	AuthConfig
	Port              int           `config:"port"`
	HealthCheckPeriod time.Duration `config:"healthCheckPeriod"`
}

// NewStatusConfig - create a new status config
func NewStatusConfig() StatusConfig {
	return &StatusConfiguration{
		Port:              statusPortDefault,
		HealthCheckPeriod: statusHealthCheckPeriodDefault,
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

const (
	pathPort              = "status.port"
	pathHealthcheckPeriod = "status.healthCheckPeriod"
)

// validate function
func (a *StatusConfiguration) validate() error {
	return nil
}
