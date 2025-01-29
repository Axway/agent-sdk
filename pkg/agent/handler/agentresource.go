package handler

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	discoveryAgent    = "DiscoveryAgent"
	traceabilityAgent = "TraceabilityAgent"
)

type agentResourceHandler struct {
	agentResourceManager resource.Manager
}

// NewAgentResourceHandler - creates a Handler for Agent resources
func NewAgentResourceHandler(agentResourceManager resource.Manager) Handler {
	return &agentResourceHandler{
		agentResourceManager: agentResourceManager,
	}
}

func (h *agentResourceHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	return nil
}
