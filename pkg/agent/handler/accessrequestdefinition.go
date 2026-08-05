package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type ardHandler struct {
	agentCacheManager agentcache.Manager
}

// NewARDHandler creates a Handler for Access Requests
func NewARDHandler(agentCacheManager agentcache.Manager) Handler {
	return &ardHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (h *ardHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	return true
}

// HandleCache adds the AccessRequestDefinition to the cache during discoveryCache's bulk rebuild.
func (h *ardHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	h.agentCacheManager.AddAccessRequestDefinition(resource)
	return nil
}

// Handle processes grpc events triggered for AccessRequests
func (h *ardHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddAccessRequestDefinition(resource)
		return nil
	}

	return h.agentCacheManager.DeleteAccessRequestDefinition(resource.Metadata.ID)
}
