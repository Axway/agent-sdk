package resource

import (
	"strings"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

func discoveryAgent(res *v1.ResourceInstance) *management.DiscoveryAgent {
	agentRes := &management.DiscoveryAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func mergeDiscoveryAgentWithConfig(agentRes *v1.ResourceInstance, cfg *config.CentralConfiguration) {
	da := discoveryAgent(agentRes)
	resCfgAdditionalTags := strings.Join(da.Spec.Config.AdditionalTags, ",")
	resCfgTeamName := da.Spec.Config.OwningTeam
	resCfgLogLevel := da.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, resCfgAdditionalTags, resCfgTeamName, resCfgLogLevel)
}
