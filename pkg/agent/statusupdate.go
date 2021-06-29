package agent

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	periodic  = "periodic status change"
	immediate = "immediate status change"
)

type agentStatusUpdate struct {
	jobs.Job
	previousActivityTime  time.Time
	currentActivityTime   time.Time
	prevStatus            string
	immediateStatusChange bool
	typeOfStatusUpdate    string
}

var periodicStatusUpdate *agentStatusUpdate
var immediateStatusUpdate *agentStatusUpdate

func (su *agentStatusUpdate) Ready() bool {
	if runStatusUpdateCheck() != nil {
		return false
	}
	// Do not start until status will be running
	status := su.getCombinedStatus()
	if status != AgentRunning {
		return false
	}

	log.Debug("Periodic status update is ready")
	su.currentActivityTime = time.Now()
	su.previousActivityTime = su.currentActivityTime
	return true
}

func (su *agentStatusUpdate) Status() error {
	// error out if the agent name does not exist
	err := runStatusUpdateCheck()
	if err != nil {
		return err
	}
	return nil
}

func (su *agentStatusUpdate) Execute() error {
	// error out if the agent name does not exist
	err := runStatusUpdateCheck()
	if err != nil {
		log.Error(errors.ErrPeriodicCheck.FormatError("periodic status updater"))
		return err
	}

	// get the status from the health check and jobs
	status := su.getCombinedStatus()
	log.Tracef("Type of agent status update being checked %s : ", su.typeOfStatusUpdate)

	if su.prevStatus != status {
		// Check to see if this is the immediate status change
		// If change of status is coming FROM or TO 'unhealthy', then report this immediately
		if su.immediateStatusChange && su.prevStatus == AgentRunning || status == AgentRunning {
			log.Tracef("Status is changing from %s to %s. Report this change of status immediately.", su.prevStatus, status)
			UpdateStatus(status, "")
			su.prevStatus = status
			return nil
		}

		UpdateLocalActivityTime()
	}

	// if the last timestamp for an event has changed, update the resource
	if time.Time(su.currentActivityTime).After(time.Time(su.previousActivityTime)) {
		log.Tracef("Activity change detected at %s, from previous activity at %s, updating status", su.currentActivityTime, su.previousActivityTime)
		UpdateStatus(status, "")
		su.prevStatus = status
		su.previousActivityTime = su.currentActivityTime
	}
	return nil
}

// StartAgentStatusUpdate - starts 2 separate jobs that runs the periodic status updates and immediate status updates
func StartAgentStatusUpdate() {
	startPeriodicStatusUpdate()
	startImmediateStatusUpdate()
}

// startPeriodicStatusUpdate - start periodic status updates based on report activity frequency config
func startPeriodicStatusUpdate() {
	interval := agent.cfg.GetReportActivityFrequency()
	periodicStatusUpdate = &agentStatusUpdate{
		typeOfStatusUpdate: periodic,
	}
	_, err := jobs.RegisterIntervalJob(periodicStatusUpdate, interval)

	if err != nil {
		log.Error(errors.ErrStartingAgentStatusUpdate.FormatError(periodic))
	}
}

// startImmediateStatusUpdate - start job that will 'immediately' update status.  NOTE : By 'immediately', this means currently 10 seconds.
// The time interval for this job is hard coded.
func startImmediateStatusUpdate() {
	interval := 10 * time.Second
	immediateStatusUpdate = &agentStatusUpdate{
		immediateStatusChange: true,
		typeOfStatusUpdate:    immediate,
	}
	_, err := jobs.RegisterDetachedIntervalJob(immediateStatusUpdate, interval)

	if err != nil {
		log.Error(errors.ErrStartingAgentStatusUpdate.FormatError(immediate))
	}
}

func (su *agentStatusUpdate) getCombinedStatus() string {
	status := su.getJobPoolStatus()
	hcStatus := su.getHealthcheckStatus()
	if hcStatus != AgentRunning {
		status = hcStatus
	}
	return status
}

// getJobPoolStatus
func (su *agentStatusUpdate) getJobPoolStatus() string {
	status := jobs.GetStatus()

	// update the status only if not running
	if status == jobs.PoolStatusStopped.String() {
		return AgentUnhealthy
	}
	return AgentRunning
}

// getHealthcheckStatus
func (su *agentStatusUpdate) getHealthcheckStatus() string {
	hcStatus := hc.GetGlobalStatus()

	// update the status only if not running
	if hcStatus == string(hc.FAIL) {
		return AgentUnhealthy
	}
	return AgentRunning
}

// runStatusUpdateCheck - returns an error if agent name is blank
func runStatusUpdateCheck() error {
	if agent.cfg.GetAgentName() == "" {
		return errors.ErrStartingAgentStatusUpdate.FormatError(periodic)
	}
	return nil
}

// UpdateLocalActivityTime - updates the local activity timestamp for the event to compare against
func UpdateLocalActivityTime() {
	periodicStatusUpdate.currentActivityTime = time.Now()
}

func getLocalActivityTime() time.Time {
	return periodicStatusUpdate.currentActivityTime
}
