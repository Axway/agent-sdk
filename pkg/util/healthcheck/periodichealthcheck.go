package healthcheck

import (
	"fmt"
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
	status := RunChecks()
	if status != OK {
		return fmt.Errorf("periodicHealthCheck status is not OK. Received status %s", status)
	}
	return nil
}
