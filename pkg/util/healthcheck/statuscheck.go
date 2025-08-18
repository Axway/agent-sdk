package healthcheck

import (
	"sync"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

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
	logger      log.FieldLogger
	checker     CheckStatus
	statusMutex sync.Mutex
}

func (c *statusCheck) setStatus(s *Status) {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	c.Status = s
}

func (c *statusCheck) executeCheck() (StatusLevel, string) {
	// c.logger.Trace("executing health check")
	c.logger.Info("executing health check")
	s := c.checker(c.Name)
	c.setStatus(s)

	logger := c.logger.WithField("result", s.Result)
	if s.Result == OK {
		// logger.Trace("health check executed successfully")
		logger.Info("health check executed successfully")
		return OK, ""
	}
	logger.WithField("details", s.Details).Error("health check execution failed")
	return FAIL, s.Details
}
