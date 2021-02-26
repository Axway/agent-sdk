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
	// The generic type for this discovery agent needs to be created
	genericAgentRes := v1alpha1.DiscoveryAgent{}

	genericAgentRes.Spec.Config = config
	genericAgentRes.Spec.Logging = logging
	genericAgentRes.Spec.DataplaneType = gatewayType
	genericAgentRes.Name = agent.cfg.GetAgentName()

	log.Debug("Creating the generic resource")
	createAgentResource(&genericAgentRes)

	log.Debug("Updating the generic resource status")
	updateAgentStatusAPI(&genericAgentRes, v1alpha1.DiscoveryAgentResource)
}

func createDiscoveryAgentStatusResource(status, message string) *v1alpha1.DiscoveryAgent {
	agentRes := v1alpha1.DiscoveryAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message

	return &agentRes
}

func mergeDiscoveryAgentWithConfig(cfg *config.CentralConfiguration) {
	da := discoveryAgent(GetAgentResource())
	resCfgAdditionalTags := strings.Join(da.Spec.Config.AdditionalTags, ",")
	resCfgTeamName := da.Spec.Config.OwningTeam
	resCfgLogLevel := da.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, resCfgAdditionalTags, resCfgTeamName, resCfgLogLevel)
}
