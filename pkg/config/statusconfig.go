package config

// StatusConfig - Interface for status config
type StatusConfig interface {
	GetPort() int
}

// StatusConfiguration -
type StatusConfiguration struct {
	AuthConfig
	Port int `config:"port"`
}

// NewStatusConfig - create a new status config
func NewStatusConfig() StatusConfig {
	return &StatusConfiguration{
		Port: 8989,
	}
}

// GetPort - Returns the status port
func (a *StatusConfiguration) GetPort() int {
	return a.Port
}
