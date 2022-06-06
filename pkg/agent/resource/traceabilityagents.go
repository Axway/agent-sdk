package resource

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

func traceabilityAgent(res *v1.ResourceInstance) *management.TraceabilityAgent {
	agentRes := &management.TraceabilityAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createTraceabilityAgentStatusResource(agentName, status, prevStatus, message string) *management.TraceabilityAgent {
	agentRes := management.TraceabilityAgent{}
	agentRes.Name = agentName
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.LatestAvailableVersion = config.AgentLatestVersion
	agentRes.Status.State = status
	agentRes.Status.PreviousState = prevStatus
	agentRes.Status.Message = message
	agentRes.Status.LastActivityTime = getTimestamp()
	agentRes.Status.SdkVersion = config.SDKVersion

	return &agentRes
}

func mergeTraceabilityAgentWithConfig(agentRes *v1.ResourceInstance, cfg *config.CentralConfiguration) {
	ta := traceabilityAgent(agentRes)
	resCfgTeamName := ta.Spec.Config.OwningTeam
	resCfgLogLevel := ta.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, "", resCfgTeamName, resCfgLogLevel)
}
