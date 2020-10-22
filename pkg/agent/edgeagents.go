package agent

import (
	"strings"

	apiV1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
)

func edgeDiscoveryAgent(res *apiV1.ResourceInstance) *v1alpha1.EdgeDiscoveryAgent {
	agentRes := &v1alpha1.EdgeDiscoveryAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createEdgeDiscoveryAgentStatusResource(status, message string) *v1alpha1.EdgeDiscoveryAgent {
	agentRes := v1alpha1.EdgeDiscoveryAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message

	return &agentRes
}

func mergeEdgeDiscoveryAgentWithConfig(cfg *config.CentralConfiguration) {
	da := edgeDiscoveryAgent(GetAgentResource())
	resCfgAdditionalTags := strings.Join(da.Spec.Config.AdditionalTags, ",")
	resCfgTeamName := da.Spec.Config.OwningTeam
	resCfgLogLevel := da.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, resCfgAdditionalTags, resCfgTeamName, resCfgLogLevel)
}

func edgeTraceabilityAgent(res *apiV1.ResourceInstance) *v1alpha1.EdgeTraceabilityAgent {
	agentRes := &v1alpha1.EdgeTraceabilityAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createEdgeTraceabilityAgentStatusResource(status, message string) *v1alpha1.EdgeTraceabilityAgent {
	agentRes := v1alpha1.EdgeTraceabilityAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message

	return &agentRes
}

func mergeEdgeTraceabilityAgentWithConfig(cfg *config.CentralConfiguration) {
	// Nothing to merge
}
