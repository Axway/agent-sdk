package agent

import (
	"strings"

	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
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

	dataplaneType := "AWS"
	// Genenric implementation
	if agentRes.Spec.DiscoveryAgent == "" {
		// The generic type for this discovery agent needs to be created
		createDiscoveryAgentResource(agentRes.Spec.Config, agentRes.Spec.Logging, dataplaneType)

		log.Debug("Update the agent")
		agentRes.Spec.DiscoveryAgent = agent.cfg.GetAgentName()
		updateAgentResource(&agentRes)
	}

	// Update the generic resource status
	updateAgentStatusAPI(createDiscoveryAgentStatusResource(status, message), v1alpha1.DiscoveryAgentResource)

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

	dataplaneType := "AWS"
	// Genenric implementation
	if agentRes.Spec.TraceabilityAgent == "" {
		// The generic type for this discovery agent needs to be created
		createTraceabilityAgentResource(agentRes.Spec.Config, agentRes.Spec.Logging, dataplaneType)

		log.Debug("Update the agent")
		agentRes.Spec.TraceabilityAgent = agent.cfg.GetAgentName()
		updateAgentResource(&agentRes)
	}

	// Update the generic resource status
	updateAgentStatusAPI(createDiscoveryAgentStatusResource(status, message), v1alpha1.TraceabilityAgentResource)

	return &agentRes
}

func mergeAWSTraceabilityAgentWithConfig(cfg *config.CentralConfiguration) {
	// Nothing to merge
}
