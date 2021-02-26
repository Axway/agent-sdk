package agent

import (
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func createTraceabilityAgentResource(config v1alpha1.TraceabilityAgentSpecConfig, logging v1alpha1.DiscoveryAgentSpecLogging, gatewayType string) {
	// The generic type for this traceability agent needs to be created
	genericAgentRes := v1alpha1.TraceabilityAgent{}

	genericAgentRes.Spec.Config = config
	genericAgentRes.Spec.Logging = logging
	genericAgentRes.Spec.DataplaneType = gatewayType
	genericAgentRes.Name = agent.cfg.GetAgentName()

	log.Debug("Creating the generic resource")
	createAgentResource(&genericAgentRes)

	log.Debug("Updating the generic resource status")
	updateAgentStatusAPI(&genericAgentRes, v1alpha1.TraceabilityAgentResource)
}

func createTraceabilityAgentStatusResource(status, message string) *v1alpha1.TraceabilityAgent {
	agentRes := v1alpha1.TraceabilityAgent{}
	agentRes.Name = agent.cfg.GetAgentName()
	agentRes.Status.Version = config.AgentVersion
	agentRes.Status.State = status
	agentRes.Status.Message = message

	return &agentRes
}

func mergeTraceabilityAgentWithConfig(cfg *config.CentralConfiguration) {
	// Nothing to merge
}
