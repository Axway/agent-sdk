package traceability

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// traceabilityHealthCheck -
type traceabilityHealthCheck struct {
	jobs.Job
	logger  log.FieldLogger
	ready   bool
	prevErr error
}

func newTraceabilityHealthCheckJob() *traceabilityHealthCheck {
	return &traceabilityHealthCheck{
		logger: log.NewFieldLogger().WithComponent("traceabilityHealthCheck").WithPackage(traceabilityStr),
	}
}

// Ready -
func (j *traceabilityHealthCheck) Ready() bool {
	j.ready = j.checkConnections() == nil
	return j.ready
}

// Status -
func (j *traceabilityHealthCheck) Status() error {
	return j.prevErr
}

// Execute -
func (j *traceabilityHealthCheck) Execute() error {
	return j.checkConnections()
}

func (j *traceabilityHealthCheck) healthcheck(name string) *hc.Status {
	// Create the default status
	status := &hc.Status{
		Result: hc.OK,
	}

	if !j.ready || j.prevErr != nil {
		status.Result = hc.FAIL
		status.Details = "agent not connected to traceability yet"
	}

	if j.prevErr != nil {
		status.Details = fmt.Sprintf("connection error: %s Failed. %s", name, j.prevErr.Error())
	}
	return status
}

func (j *traceabilityHealthCheck) checkConnections() error {
	client, err := getClient()
	if err != nil {
		j.logger.WithError(err).Error("could not get traceability client")
		return err
	}
	j.prevErr = client.Connect()
	if j.prevErr != nil {
		j.logger.WithError(j.prevErr).Error("connection failed")
	} else {
		j.logger.Trace("connection to traceability succeeded")
	}
	return j.prevErr
}
