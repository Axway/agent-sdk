package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
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

func (h *ardHandler) Kinds() []string {
	return []string{management.AccessRequestDefinitionGVK().Kind}
}

func (h *ardHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	if event.Payload.Kind != management.AccessRequestDefinitionGVK().Kind {
		return false
	}

	return true
}

// Handle processes grpc events triggered for AccessRequests
func (h *ardHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.AccessRequestDefinitionGVK().Kind {
		return nil
	}

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddAccessRequestDefinition(resource)
		return nil
	}

	return h.agentCacheManager.DeleteAccessRequestDefinition(resource.Metadata.ID)
}
