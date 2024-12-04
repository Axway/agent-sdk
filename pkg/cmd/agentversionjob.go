package cmd

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/agent/resource"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"

	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	log "github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	avcCronSchedule = "@daily"

	agentStateCurrent   = "current"
	agentStateAvailable = "available"
	agentStateOutdated  = "outdated"
	agentStateRetracted = "retracted"
)

// AgentVersionCheckJob - polls for agent versions
type AgentVersionCheckJob struct {
	jobs.Job
	logger  log.FieldLogger
	manager resource.Manager
}

// NewAgentVersionCheckJob - creates a new agent version check job structure
func NewAgentVersionCheckJob(cfg config.CentralConfig) (*AgentVersionCheckJob, error) {
	manager := agent.GetAgentResourceManager()
	if manager == nil {
		return nil, errors.ErrStartingVersionChecker.FormatError("could not get the agent resource manager")
	}

	return &AgentVersionCheckJob{
		manager: manager,
		logger: log.NewFieldLogger().
			WithPackage("sdk.cmd").
			WithComponent("agentVersionJob"),
	}, nil
}

// Ready -
func (avj *AgentVersionCheckJob) Ready() bool {
	return true
}

// Status -
func (avj *AgentVersionCheckJob) Status() error {
	return nil
}

// Execute - run agent version check job one time
func (avj *AgentVersionCheckJob) Execute() error {
	state, err := avj.getAgentState()
	if err != nil {
		avj.logger.WithError(err).Warn("agent cannot determine the current available release. Be aware that your agent could be outdated.")
		return nil
	}

	switch state {
	case agentStateCurrent:
		avj.logger.Trace("agent is up to date.")
	case agentStateAvailable:
		avj.logger.Warn("please be aware that there is a newer agent version available.")
	case agentStateOutdated:
		avj.logger.Error("current agent version is no longer supported. We strongly advise to update the agent as soon as possible.")
	case agentStateRetracted:
		avj.logger.Error("current agent version has a known issue, please update the agent immediately.")
	}
	return nil
}

func (avj *AgentVersionCheckJob) getAgentState() (string, error) {
	agentRes := avj.manager.GetAgentResource()
	if agentRes == nil {
		return "", fmt.Errorf("could not get the agent resource")
	}

	switch agentRes.GetGroupVersionKind().Kind {
	case "TraceabilityAgent":
		ta := management.NewTraceabilityAgent("", "")
		if err := ta.FromInstance(agentRes); err != nil {
			return "", fmt.Errorf("could not convert resource instance to TraceabilityAgent resource")
		}
		return ta.Agentstate.Update, nil
	case "DiscoveryAgent":
		da := management.NewDiscoveryAgent("", "")
		if err := da.FromInstance(agentRes); err != nil {
			return "", fmt.Errorf("could not convert resource instance to DiscoveryAgent resource")
		}
		return da.Agentstate.Update, nil
	}

	return "", fmt.Errorf("agent resource is neither Discovery nor Traceability")
}

// startVersionCheckJobs - starts both a single run and continuous checks
func startVersionCheckJobs(cfg config.CentralConfig, agentFeaturesCfg config.AgentFeaturesConfig) {
	if !util.IsNotTest() || !agentFeaturesCfg.VersionCheckerEnabled() {
		return
	}
	// register the agent version checker single run job
	checkJob, err := NewAgentVersionCheckJob(cfg)
	if err != nil {
		log.Errorf("could not create the agent version checker: %v", err.Error())
		return
	}
	if id, err := jobs.RegisterSingleRunJobWithName(checkJob, "Version Check"); err == nil {
		log.Tracef("registered agent version checker job: %s", id)
	}
	if id, err := jobs.RegisterScheduledJobWithName(checkJob, avcCronSchedule, "Version Check Schedule"); err == nil {
		log.Tracef("registered agent version checker cronjob: %s", id)
	}
}
