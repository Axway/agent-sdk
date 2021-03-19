package healthcheck

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"
)

const maxConsecutiveErr = 3

type periodicHealthCheck struct {
	jobs.Job
	errCount int
	interval time.Duration
}

func (sm *periodicHealthCheck) Ready() bool {
	return true
}

func (sm *periodicHealthCheck) Status() error {
	return nil
}

func (sm *periodicHealthCheck) Execute() error {
	for {
		// Execute all healthchecks
		RunChecks()
		time.Sleep(sm.interval)
	}
}
