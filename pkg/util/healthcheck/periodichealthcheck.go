package healthcheck

import (
	"fmt"

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
	log.Debug("Checking if periodic healthcheck is ready")
	if RunChecks() != OK {
		return false
	}
	log.Debug("Periodic healthcheck is ready")
	return true
}

func (sm *periodicHealthCheck) Status() error {
	// Only error out once the healthcheck has failed 3 times
	if sm.errCount >= maxConsecutiveErr {
		log.Debugf("Healthchecks failed %v consecutive times, stopping jobs", sm.errCount)
		return fmt.Errorf("Healthchecks failed 3 consecutive times, pausing execution")
	}
	return nil
}

func (sm *periodicHealthCheck) Execute() error {
	// Check that all healthchecks are OK
	log.Debug("Periodic healthcheck executing")
	if RunChecks() != OK {
		log.Error(errors.ErrHealthCheck)
		sm.errCount++
		log.Debugf("Healthcheck failed %v times", sm.errCount)
	} else {
		log.Debug("Healthcheck passed")
		// All Healthchecks passed, reset to 0
		sm.errCount = 0
	}
	return nil
}
