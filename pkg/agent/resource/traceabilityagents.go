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

func mergeTraceabilityAgentWithConfig(agentRes *v1.ResourceInstance, cfg *config.CentralConfiguration) {
	ta := traceabilityAgent(agentRes)
	resCfgTeamName := ta.Spec.Config.OwningTeam
	resCfgLogLevel := ta.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, "", resCfgTeamName, resCfgLogLevel)
}
