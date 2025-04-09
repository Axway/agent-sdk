package resource

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

func complianceAgent(res *v1.ResourceInstance) *management.ComplianceAgent {
	agentRes := &management.ComplianceAgent{}
	agentRes.FromInstance(res)

	return agentRes
}

func mergeComplianceAgentWithConfig(agentRes *v1.ResourceInstance, cfg *config.CentralConfiguration) {
	ca := complianceAgent(agentRes)

	applyResConfigToCentralConfig(cfg, "", "", "", ca.Spec.Config.ManagedEnvironments)
}
