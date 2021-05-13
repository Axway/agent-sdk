package healthcheck

import "time"

const defaultCheckInterval = 30 * time.Second

// healthChecker - info about the service
type healthChecker struct {
	Name       string                  `json:"name"`
	Version    string                  `json:"version,omitempty"`
	Status     StatusLevel             `json:"status"`
	Checks     map[string]*statusCheck `json:"statusChecks"`
	registered bool
}

// Status - the status of this healthcheck
type Status struct {
	Result  StatusLevel `json:"result"`
	Details string      `json:"details,omitempty"`
}

// statusCheck - the status check
type statusCheck struct {
	ID       string  `json:"-"`
	Name     string  `json:"name"`
	Endpoint string  `json:"endpoint"`
	Status   *Status `json:"status"`
	checker  CheckStatus
}

// StatusLevel - the level of the status of the healthcheck
type StatusLevel string

const (
	// OK - healthcheck is running properly
	OK StatusLevel = "OK"
	// FAIL - healthcheck is failing
	FAIL StatusLevel = "FAIL"
)

// CheckStatus - the format expected for the method to get the Healthcheck status
type CheckStatus func(name string) *Status

// GetStatusLevel format for the function to get the StatusLevel of an endpoint
type GetStatusLevel func(endpoint string) StatusLevel

// RegisterHealth is for registering a healthcheck function
type RegisterHealth func(name, endpoint string, check CheckStatus) (string, error)
