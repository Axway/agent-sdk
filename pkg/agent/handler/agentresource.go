package handler

import (
	"context"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	discoveryAgent    = "DiscoveryAgent"
	traceabilityAgent = "TraceabilityAgent"
	governanceAgent   = "GovernanceAgent"
)

type samplingFeatures interface {
	SetIsSampling(bool)
	SetSamplingPerMinuteLimit(int)
}

type agentResourceHandler struct {
	agentResourceManager resource.Manager
	features             samplingFeatures
}

// NewAgentResourceHandler - creates a Handler for Agent resources
func NewAgentResourceHandler(agentResourceManager resource.Manager, features samplingFeatures) Handler {
	return &agentResourceHandler{
		agentResourceManager: agentResourceManager,
		features:             features,
	}
}

func (h *agentResourceHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if h.agentResourceManager != nil && action == proto.Event_UPDATED {
		kind := resource.Kind
		switch kind {
		case discoveryAgent:
			fallthrough
		case traceabilityAgent:
			h.checkToEnableSampling(resource)
			fallthrough
		case governanceAgent:
			h.agentResourceManager.SetAgentResource(resource)
		}
	}
	return nil
}

// EnableSampling -
func (h *agentResourceHandler) checkToEnableSampling(resource *v1.ResourceInstance) {
	ta := management.NewTraceabilityAgent("", "")
	err := ta.FromInstance(resource)
	if err != nil {
		return
	}

	if !ta.Agentstate.Sampling.Enabled {
		return
	}

	h.features.SetIsSampling(true)
	h.features.SetSamplingPerMinuteLimit(int(ta.Agentstate.Sampling.Limit))

	go func(ta *management.TraceabilityAgent) {
		tickerDuration := time.Until(time.Time(ta.Agentstate.Sampling.EndTime))
		if tickerDuration <= 0 {
			return
		}
		ticker := time.NewTicker(tickerDuration)
		<-ticker.C
		ticker.Stop()
		h.features.SetIsSampling(false)
	}(ta)
}
