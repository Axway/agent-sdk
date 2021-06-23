package agent

import (
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

func governanceAgent(res *apiV1.ResourceInstance) *v1alpha1.GovernanceAgent {
	agentRes := &v1alpha1.GovernanceAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createGovernanceAgentStatusResource(status, message string) *v1alpha1.GovernanceAgent {
	agentRes := v1alpha1.GovernanceAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message

	return &agentRes
}

func mergeGovernanceAgentWithConfig(cfg *config.CentralConfiguration) {
	governanceAgent(GetAgentResource())
	//resCfgAdditionalTags := strings.Join(ga.Spec.Config.AdditionalTags, ",")
	//resCfgTeamName := ga.Spec.Config.OwningTeam
	resCfgLogLevel := "info" //ga.Spec.Logging.Level
	applyResConfigToCentralConfig(cfg, "", "", resCfgLogLevel)
}
