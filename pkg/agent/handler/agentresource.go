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
	agentStateSubresource = "agentstate"
)

type sampling interface {
	EnableSampling(samplingLimit int32, samplingEndTime time.Time)
}

type TraceabilityTriggerHandler interface {
	TriggerTraceability()
}

// Register an AgentResourceUpdateHandler in an agent to trigger events when changes to the resource is made
type AgentResourceUpdateHandler interface {
	AgentResourceUpdate(ctx context.Context, resource *v1.ResourceInstance)
}

type agentTypeHandler func(action proto.Event_Type, subres string, resource *v1.ResourceInstance) error

type agentResourceHandler struct {
	agentResourceManager resource.Manager
	sampler              sampling
	agentTypeHandler     map[string]agentTypeHandler
}

// NewAgentResourceHandler - creates a Handler for Agent resources
func NewAgentResourceHandler(agentResourceManager resource.Manager, sampler sampling) Handler {
	h := &agentResourceHandler{
		agentResourceManager: agentResourceManager,
		sampler:              sampler,
	}
	h.agentTypeHandler = map[string]agentTypeHandler{
		management.DiscoveryAgentGVK().Kind:    h.handleDiscovery,
		management.TraceabilityAgentGVK().Kind: h.handleTraceability,
		management.ComplianceAgentGVK().Kind:   h.handleCompliance,
	}
	return h
}

func (h *agentResourceHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	// skip any processing if the agent resource manager is not set
	if h.agentResourceManager == nil {
		return nil
	}

	action := GetActionFromContext(ctx)
	if f, ok := h.agentTypeHandler[resource.Kind]; ok {
		return f(action, meta.Subresource, resource)
	}

	return nil
}

func (h *agentResourceHandler) handleDiscovery(action proto.Event_Type, subres string, resource *v1.ResourceInstance) error {
	if action == proto.Event_UPDATED {
		h.agentResourceManager.SetAgentResource(resource)
	}
	return nil
}

func (h *agentResourceHandler) handleTraceability(action proto.Event_Type, subres string, resource *v1.ResourceInstance) error {

	switch {
	case action == proto.Event_UPDATED:
		h.agentResourceManager.SetAgentResource(resource)
	case action == proto.Event_SUBRESOURCEUPDATED && subres == management.TraceabilityAgentAgentstateSubResourceName:
		return h.handleTraceabilitySampling(resource)
	}

	return nil
}

func (h *agentResourceHandler) handleTraceabilitySampling(resource *v1.ResourceInstance) error {
	ta := &management.TraceabilityAgent{}
	err := ta.FromInstance(resource)
	if err != nil {
		return err
	}

	if !ta.Agentstate.Sampling.Enabled {
		return nil
	}

	h.sampler.EnableSampling(ta.Agentstate.Sampling.Limit, time.Time(ta.Agentstate.Sampling.EndTime))
	if traceabilityTriggerHandler, ok := h.agentResourceManager.GetHandler().(TraceabilityTriggerHandler); ok {
		traceabilityTriggerHandler.TriggerTraceability()
	}
	return nil
}

func (h *agentResourceHandler) handleCompliance(action proto.Event_Type, subres string, resource *v1.ResourceInstance) error {
	if action == proto.Event_UPDATED {
		h.agentResourceManager.SetAgentResource(resource)
	}
	return nil
}
