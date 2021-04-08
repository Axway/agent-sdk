package agent

import (
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

func traceabilityAgent(res *apiV1.ResourceInstance) *v1alpha1.TraceabilityAgent {
	agentRes := &v1alpha1.TraceabilityAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createTraceabilityAgentStatusResource(status, message string) *v1alpha1.TraceabilityAgent {
	agentRes := v1alpha1.TraceabilityAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message
	agentRes.Status.LastActivityTime = getTimestamp()

	return &agentRes
}

func mergeTraceabilityAgentWithConfig(cfg *config.CentralConfiguration) {
	ta := traceabilityAgent(GetAgentResource())
	resCfgTeamName := ta.Spec.Config.OwningTeam
	resCfgLogLevel := ta.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, "", resCfgTeamName, resCfgLogLevel)
}
