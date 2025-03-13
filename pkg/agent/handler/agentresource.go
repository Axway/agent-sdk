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
	discoveryAgent        = "DiscoveryAgent"
	traceabilityAgent     = "TraceabilityAgent"
	governanceAgent       = "GovernanceAgent"
	agentStateSubresource = "agentstate"
)

type sampling interface {
	EnableSampling(samplingLimit int32, samplingEndTime time.Time)
}

type TraceabilityTriggerHandler interface {
	TriggerTraceability()
}

type agentResourceHandler struct {
	agentResourceManager resource.Manager
	sampler              sampling
}

// NewAgentResourceHandler - creates a Handler for Agent resources
func NewAgentResourceHandler(agentResourceManager resource.Manager, sampler sampling) Handler {
	return &agentResourceHandler{
		agentResourceManager: agentResourceManager,
		sampler:              sampler,
	}
}

func (h *agentResourceHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)

	if h.agentResourceManager != nil && action == proto.Event_UPDATED {
		kind := resource.Kind
		switch kind {
		case discoveryAgent:
			fallthrough
		case traceabilityAgent:
			fallthrough
		case governanceAgent:
			h.agentResourceManager.SetAgentResource(resource)
		}
	}

	if !(action == proto.Event_SUBRESOURCEUPDATED && resource.Kind == traceabilityAgent && meta.Subresource == agentStateSubresource) {
		return nil
	}

	ta := &management.TraceabilityAgent{}
	err := ta.FromInstance(resource)
	if err != nil {
		return err
	}
	if ta.Agentstate.Sampling.Enabled {
		h.sampler.EnableSampling(ta.Agentstate.Sampling.Limit, time.Time(ta.Agentstate.Sampling.EndTime))

		if traceabilityTriggerHandler, ok := h.agentResourceManager.GetHandler().(TraceabilityTriggerHandler); ok {
			traceabilityTriggerHandler.TriggerTraceability()
		}
	}

	return nil
}
