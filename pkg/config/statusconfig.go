package config

import (
	"time"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
)

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
		Port:              8989,
		HealthCheckPeriod: 3 * time.Minute,
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

// AddStatusConfigProperties - Adds the command properties needed for Status Config
func AddStatusConfigProperties(props properties.Properties) {
	props.AddIntProperty(pathPort, 8989, "The port that will serve the status endpoints")
	props.AddDurationProperty(pathHealthcheckPeriod, 3*time.Minute, "Time in minutes allotted for services to be ready before exiting discovery agent")
	props.AddBoolFlag("status", "Get the status of all the Health Checks")
}

// ParseStatusConfig - Parses the Status Config values form teh command line
func ParseStatusConfig(props properties.Properties) (StatusConfig, error) {
	cfg := &StatusConfiguration{
		Port:              props.IntPropertyValue(pathPort),
		HealthCheckPeriod: props.DurationPropertyValue(pathHealthcheckPeriod),
	}
	return cfg, nil
}
