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
	var resCfgTeamID string
	if da.Spec.Config.Owner != nil {
		resCfgTeamID = da.Spec.Config.Owner.ID
	}
	resCfgLogLevel := da.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, resCfgAdditionalTags, resCfgTeamID, resCfgLogLevel, nil)
}
