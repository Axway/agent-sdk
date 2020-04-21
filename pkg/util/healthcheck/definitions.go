package healthcheck

// HealthChecker - info about the service
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

// StatusLevel - the level of the status of the healtheck
type StatusLevel string

const (
	// OK - healthcheck is running properly
	OK StatusLevel = "OK"
	// FAIL - healthcheck is failing
	FAIL StatusLevel = "FAIL"
)

// CheckStatus - the format expected for the method to get the Healthcheck status
type CheckStatus func(name string) *Status
