package agent

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type periodicStatusUpdate struct {
	jobs.Job
	previousActivityTime time.Time
	currentActivityTime  time.Time
}

var statusUpdate *periodicStatusUpdate

func (su *periodicStatusUpdate) Ready() bool {
	if runStatusUpdateCheck() != nil {
		return false
	}
	log.Debug("Periodic status update is ready")
	su.currentActivityTime = time.Now()
	su.previousActivityTime = su.currentActivityTime
	return true
}

func (su *periodicStatusUpdate) Status() error {
	// error out if the agent name does not exist
	err := runStatusUpdateCheck()
	if err != nil {
		return err
	}
	return nil
}

func (su *periodicStatusUpdate) Execute() error {
	// error out if the agent name does not exist
	err := runStatusUpdateCheck()
	if err != nil {
		log.Error(errors.ErrPeriodicCheck.FormatError("periodic status updater"))
		return err
	}
	// if the last timestamp for an event has changed, update the status
	if time.Time(su.currentActivityTime).After(time.Time(su.previousActivityTime)) {
		log.Tracef("Activity change detected at %s, from previous activity at %s, updating status", su.currentActivityTime, su.previousActivityTime)
		UpdateStatus(AgentRunning, "")
		su.previousActivityTime = su.currentActivityTime
	}
	return nil
}

//StartPeriodicStatusUpdate - starts a job that runs the periodic status updates
func StartPeriodicStatusUpdate() {
	interval := agent.cfg.GetReportActivityFrequency()
	statusUpdate = &periodicStatusUpdate{}
	_, err := jobs.RegisterIntervalJob(statusUpdate, interval)

	if err != nil {
		log.Error(errors.Wrap(errors.ErrStartingPeriodicStatusUpdate, err.Error()))
	}
}

//runStatusUpdateCheck - returns an error if agent name is blank
func runStatusUpdateCheck() error {
	if agent.cfg.GetAgentName() == "" {
		return errors.ErrStartingPeriodicStatusUpdate
	}
	return nil
}

//UpdateLocalActivityTime - updates the local activity timestamp for the event to compare against
func UpdateLocalActivityTime() {
	statusUpdate.currentActivityTime = time.Now()
}
