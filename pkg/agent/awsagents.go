package agent

import (
	"strings"

	apiV1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
)

func awsDiscoveryAgent(res *apiV1.ResourceInstance) *v1alpha1.AWSDiscoveryAgent {
	agentRes := &v1alpha1.AWSDiscoveryAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createAWSDiscoveryAgentStatusResource(status, message string) *v1alpha1.AWSDiscoveryAgent {
	agentRes := v1alpha1.AWSDiscoveryAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message

	return &agentRes
}

func mergeAWSDiscoveryAgentWithConfig(cfg *config.CentralConfiguration) {
	da := awsDiscoveryAgent(GetAgentResource())
	resCfgAdditionalTags := strings.Join(da.Spec.Config.AdditionalTags, ",")
	resCfgTeamName := da.Spec.Config.OwningTeam
	resCfgLogLevel := da.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, resCfgAdditionalTags, resCfgTeamName, resCfgLogLevel)
}

func awsTraceabilityAgent(res *apiV1.ResourceInstance) *v1alpha1.AWSTraceabilityAgent {
	agentRes := &v1alpha1.AWSTraceabilityAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createAWSTraceabilityAgentStatusResource(status, message string) *v1alpha1.AWSTraceabilityAgent {
	agentRes := v1alpha1.AWSTraceabilityAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message

	return &agentRes
}

func mergeAWSTraceabilityAgentWithConfig(cfg *config.CentralConfiguration) {
	// Nothing to merge
}
