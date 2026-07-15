package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
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

func (h *apdHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	return true
}

// Handle processes grpc events triggered for Application Profile Definitions
func (h *apdHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if action != proto.Event_DELETED {
		h.agentCacheManager.AddApplicationProfileDefinition(resource)
		return nil
	}

	return h.agentCacheManager.DeleteApplicationProfileDefinition(resource.Metadata.ID)
}
