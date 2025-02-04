package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type apdHandler struct {
	agentCacheManager agentcache.Manager
}

// NewAPDHandler creates a Handler for Application Profile Definitions
func NewAPDHandler(agentCacheManager agentcache.Manager) Handler {
	return &apdHandler{
		agentCacheManager: agentCacheManager,
	}
}

// Handle processes grpc events triggered for Application Profile Definitions
func (h *apdHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.ApplicationProfileDefinitionGVK().Kind {
		return nil
	}

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddApplicationProfileDefinition(resource)
		return nil
	}

	return h.agentCacheManager.DeleteApplicationProfileDefinition(resource.Metadata.ID)
}
