package config

import "time"

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
