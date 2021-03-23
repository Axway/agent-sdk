package agent

import (
	"strings"

	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func discoveryAgent(res *apiV1.ResourceInstance) *v1alpha1.DiscoveryAgent {
	agentRes := &v1alpha1.DiscoveryAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createDiscoveryAgentResource(config v1alpha1.DiscoveryAgentSpecConfig, logging v1alpha1.DiscoveryAgentSpecLogging, gatewayType string) {
	// The discovery agent resource needs to be created
	agentResource := v1alpha1.DiscoveryAgent{}

	agentResource.Spec.Config = config
	agentResource.Spec.Logging = logging
	agentResource.Spec.DataplaneType = gatewayType
	agentResource.Name = agent.cfg.GetAgentName()

	log.Debug("Creating the discovery agent resource")
	createAgentResource(&agentResource)

	log.Debug("Updating the discovery agent status sub-resource")
	updateAgentStatusAPI(&agentResource, v1alpha1.DiscoveryAgentResource)
}

func createDiscoveryAgentStatusResource(status, message string) *v1alpha1.DiscoveryAgent {
	agentRes := v1alpha1.DiscoveryAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message
	agentRes.Status.LastActivityTime = getTimestamp()

	return &agentRes
}

func mergeDiscoveryAgentWithConfig(cfg *config.CentralConfiguration) {
	da := discoveryAgent(GetAgentResource())
	resCfgAdditionalTags := strings.Join(da.Spec.Config.AdditionalTags, ",")
	resCfgTeamName := da.Spec.Config.OwningTeam
	resCfgLogLevel := da.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, resCfgAdditionalTags, resCfgTeamName, resCfgLogLevel)
}
