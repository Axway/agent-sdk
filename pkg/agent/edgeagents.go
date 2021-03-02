package agent

import (
	"strings"

	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func edgeDiscoveryAgent(res *apiV1.ResourceInstance) *v1alpha1.EdgeDiscoveryAgent {
	agentRes := &v1alpha1.EdgeDiscoveryAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createEdgeDiscoveryAgentStatusResource(status, message string) *v1alpha1.EdgeDiscoveryAgent {
	agentRes := edgeDiscoveryAgent(GetAgentResource())
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message

	dataplaneType := edgeDataplaneType

	// Genenric implementation
	if agentRes.Spec.DiscoveryAgent == "" {
		// The discovery agent resource needs to be created
		createDiscoveryAgentResource(agentRes.Spec.Config, agentRes.Spec.Logging, dataplaneType)

		log.Debug("Update the discovery agent")
		agentRes.Spec.DiscoveryAgent = agent.cfg.GetAgentName()
		updateAgentResource(&agentRes)
	}

	// Update the discovery agent status resource
	updateAgentStatusAPI(createDiscoveryAgentStatusResource(status, message), v1alpha1.DiscoveryAgentResource)

	return agentRes
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

	dataplaneType := edgeDataplaneType

	// Genenric implementation
	if agentRes.Spec.TraceabilityAgent == "" {
		// The traceability agent resource needs to be created
		createTraceabilityAgentResource(agentRes.Spec.Config, agentRes.Spec.Logging, dataplaneType)

		log.Debug("Update the traceability agent")
		agentRes.Spec.TraceabilityAgent = agent.cfg.GetAgentName()
		updateAgentResource(&agentRes)
	}

	// Update the status sub-resource
	updateAgentStatusAPI(createDiscoveryAgentStatusResource(status, message), v1alpha1.TraceabilityAgentResource)

	return &agentRes
}

func mergeEdgeTraceabilityAgentWithConfig(cfg *config.CentralConfiguration) {
	// Nothing to merge
}
