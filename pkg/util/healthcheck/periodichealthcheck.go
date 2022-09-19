package healthcheck

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"
)

type periodicHealthCheck struct {
	jobs.Job
	interval time.Duration
}

func (sm *periodicHealthCheck) Ready() bool {
	return true
}

func (sm *periodicHealthCheck) Status() error {
	return nil
}

func (sm *periodicHealthCheck) Execute() error {
	status := RunChecks()
	if status != OK {
		logger.WithField("status", status).Warn("periodicHealthCheck status is not OK")
	}
	return nil
}
