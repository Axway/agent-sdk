package healthcheck

import "sync"

// Status - the status of this healthcheck
type Status struct {
	Result  StatusLevel `json:"result"`
	Details string      `json:"details,omitempty"`
}

// CheckStatus - the format expected for the method to get the Healthcheck status
type CheckStatus func(name string) *Status

// statusCheck - the status check
type statusCheck struct {
	ID          string  `json:"-"`
	Name        string  `json:"name"`
	Endpoint    string  `json:"endpoint"`
	Status      *Status `json:"status"`
	checker     CheckStatus
	statusMutex *sync.Mutex
}

func (check *statusCheck) setStatus(s *Status) {
	check.Status = s
}

func (check *statusCheck) executeCheck() {
	s := check.checker(check.Name)
	check.setStatus(s)

	if check.Status.Result == OK {
		hcm.logger.
			WithField("check", check.Name).
			WithField("result", check.Status.Result).
			Trace("health check is OK")
	} else {
		hcm.logger.
			WithField("check", check.Name).
			WithField("result", check.Status.Result).
			WithField("details", check.Status.Details).
			Error("health check failed")
	}
}
