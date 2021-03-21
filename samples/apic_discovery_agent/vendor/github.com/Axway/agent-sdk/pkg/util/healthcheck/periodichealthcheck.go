package healthcheck

import (
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const maxConsecutiveErr = 3

type periodicHealthCheck struct {
	jobs.Job
	errCount int
}

func (sm *periodicHealthCheck) Ready() bool {
	// wait for the healthchecks to pass
	if RunChecks() != OK {
		return false
	}
	log.Debug("Periodic healthcheck is ready")
	return true
}

func (sm *periodicHealthCheck) Status() error {
	// Only error out once the healthcheck has failed 3 times
	if sm.errCount >= maxConsecutiveErr {
		return ErrMaxconsecutiveErrors.FormatError(maxConsecutiveErr)
	}
	return nil
}

func (sm *periodicHealthCheck) Execute() error {
	// Check that all healthchecks are OK
	if RunChecks() != OK {
		log.Error(errors.ErrHealthCheck)
		sm.errCount++
		log.Debugf("Healthcheck failed %v times", sm.errCount)
	} else {
		// All Healthchecks passed, reset to 0
		sm.errCount = 0
	}
	return nil
}
