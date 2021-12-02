package agent

import (
	"sync"
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

var previousStatus string // The global previous status to be used by both update jobs
var updateStatusMutex *sync.Mutex

func init() {
	updateStatusMutex = &sync.Mutex{}
}

type agentStatusUpdate struct {
	jobs.Job
	previousActivityTime  time.Time
	currentActivityTime   time.Time
	immediateStatusChange bool
	typeOfStatusUpdate    string
}

var periodicStatusUpdate *agentStatusUpdate
var immediateStatusUpdate *agentStatusUpdate

func (su *agentStatusUpdate) Ready() bool {
	// Do not start until status will be running
	status := su.getCombinedStatus()
	if status != AgentRunning && su.immediateStatusChange {
		return false
	}

	log.Trace("Periodic status update is ready")
	su.currentActivityTime = time.Now()
	su.previousActivityTime = su.currentActivityTime
	return true
}

func (su *agentStatusUpdate) Status() error {
	return nil
}

func (su *agentStatusUpdate) Execute() error {
	// only one status update should execute at a time
	log.Tracef("get status update lock %s", su.typeOfStatusUpdate)
	updateStatusMutex.Lock()
	defer func() {
		log.Tracef("return status update lock %s", su.typeOfStatusUpdate)
		updateStatusMutex.Unlock()
	}()

	// get the status from the health check and jobs
	status := su.getCombinedStatus()
	log.Tracef("Type of agent status update being checked %s : ", su.typeOfStatusUpdate)

	// Check to see if this is the immediate status change
	// If change of status is coming FROM or TO 'unhealthy', then report this immediately
	if previousStatus != status && (su.immediateStatusChange && previousStatus == AgentRunning || status == AgentRunning) {
		log.Tracef("Status is changing from %s to %s. Report this change of status immediately.", previousStatus, status)
		UpdateStatusWithPrevious(status, previousStatus, "")
	} else if su.typeOfStatusUpdate == periodic {
		// If its a periodic check, tickle last activity so that UI shows agent is still alive.  Not needed for immediate check.
		log.Debugf("%s -- Last activity updated", su.typeOfStatusUpdate)
		UpdateStatusWithPrevious(status, previousStatus, "")
		su.previousActivityTime = su.currentActivityTime
	}

	previousStatus = status
	return nil
}

// StartAgentStatusUpdate - starts 2 separate jobs that runs the periodic status updates and immediate status updates
func StartAgentStatusUpdate() {
	if err := runStatusUpdateCheck(); err != nil {
		log.Errorf("not starting status update jobs: %s", err)
		return
	}
	startPeriodicStatusUpdate()
	startImmediateStatusUpdate()
}

// startPeriodicStatusUpdate - start periodic status updates based on report activity frequency config
func startPeriodicStatusUpdate() {
	interval := agent.cfg.GetReportActivityFrequency()
	periodicStatusUpdate = &agentStatusUpdate{
		typeOfStatusUpdate: periodic,
	}
	_, err := jobs.RegisterIntervalJobWithName(periodicStatusUpdate, interval, "Status Update")

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
	_, err := jobs.RegisterDetachedIntervalJobWithName(immediateStatusUpdate, interval, "Immediate Status Update")

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
