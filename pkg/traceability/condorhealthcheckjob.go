package traceability

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

const healthcheckCondor = "Traceability connectivity"

// condorHealthCheckJob -
type condorHealthCheckJob struct {
	jobs.Job
	agentHealthChecker *traceabilityAgentHealthChecker
}

// Ready -
func (j *condorHealthCheckJob) Ready() bool {
	status := j.agentHealthChecker.healthcheck(healthcheckCondor)
	if status.Result == hc.OK {
		return true
	}
	return false
}

// Status -
func (j *condorHealthCheckJob) Status() error {
	status := j.agentHealthChecker.healthcheck(healthcheckCondor)
	if status.Result == hc.OK {
		return nil
	}
	return fmt.Errorf("error getting health check status for %s", healthcheckCondor)
}

// Execute -
func (j *condorHealthCheckJob) Execute() error {
	return nil
}
