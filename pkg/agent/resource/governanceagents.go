package resource

import (
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

func governanceAgent(res *apiV1.ResourceInstance) *management.GovernanceAgent {
	agentRes := &management.GovernanceAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func createGovernanceAgentStatusResource(agentName, status, prevStatus, message string) *management.GovernanceAgent {
	agentRes := management.GovernanceAgent{}
	agentRes.Name = agentName
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.PreviousState = prevStatus
	agentRes.Status.Message = message
	agentRes.Status.SdkVersion = config.SDKVersion

	return &agentRes
}

func mergeGovernanceAgentWithConfig(agentRes *apiV1.ResourceInstance, cfg *config.CentralConfiguration) {
	governanceAgent(agentRes)
	applyResConfigToCentralConfig(cfg, "", "", "")
}
