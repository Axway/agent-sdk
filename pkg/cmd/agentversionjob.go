package cmd

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/agent/resource"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"

	"regexp"
	"strings"

	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	log "github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	avcCronSchedule = "@daily"
)

// AgentVersionCheckJob - polls for agent versions
type AgentVersionCheckJob struct {
	jobs.Job
	buildVersion string
	manager      resource.Manager
}

// NewAgentVersionCheckJob - creates a new agent version check job structure
func NewAgentVersionCheckJob(cfg config.CentralConfig) (*AgentVersionCheckJob, error) {
	// get current build version
	buildVersion, err := getBuildVersion()
	if err != nil {
		return nil, err
	}

	manager := agent.GetAgentResourceManager()
	if manager == nil {
		return nil, errors.ErrStartingVersionChecker.FormatError("could not get the agent resource manager")
	}

	return &AgentVersionCheckJob{
		manager:      manager,
		buildVersion: buildVersion,
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
		log.Trace(err)
		// Could not get update from agent state.  Warn that we could not determine version and continue processing
		log.Warn("Agent cannot determine the current available release. Be aware that your agent could be outdated.")
	}

	switch state {
	case "available":
		log.Warn("Please be aware that there is a newer agent version available")
	case "outdated":
		log.Error("current agent version is no longer supported. We strongly advise to update the agent as soon as possible.")
	case "retracted":
		log.Error("current agent version has a known issue, please update the agent immediately.")
	}
	return nil
}

func (avj *AgentVersionCheckJob) getAgentState() (string, error) {
	agentRes := avj.manager.GetAgentResource()
	if agentRes == nil {
		return "", fmt.Errorf("could not get the agent resource")
	}

	// The kind should be only DA or TA
	subResKey := management.DiscoveryAgentAgentstateSubResourceName
	if agentRes.GetGroupVersionKind().Kind == "TraceabilityAgent" {
		subResKey = management.TraceabilityAgentAgentstateSubResourceName
	}

	err := fmt.Errorf("could not find the agentstate from agent subresource")
	// This can happen at the first time the job executes, in which the resource has no agentState set beforehand
	agentStateIface := agentRes.GetSubResource(subResKey)
	if agentStateIface == nil {
		return "", err
	}

	if agentState, ok := agentStateIface.(map[string]interface{}); ok {
		if state, ok := agentState["update"]; ok {
			if update, ok := state.(string); ok {
				return update, nil
			}
		}
	}
	return "", err
}

func getBuildVersion() (string, error) {
	//remove -SHA from build version
	versionNoSHA := strings.Split(BuildVersion, "-")[0]

	//regex check for semantic versioning
	semVerRegexp := regexp.MustCompile(`\d.\d.\d`)
	if versionNoSHA == "" || !semVerRegexp.MatchString(versionNoSHA) {
		return "", errors.ErrStartingVersionChecker.FormatError("build version is missing or of noncompliant semantic versioning")
	}
	return versionNoSHA, nil
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
