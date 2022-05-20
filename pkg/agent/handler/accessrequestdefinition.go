package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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

// Handle processes grpc events triggered for AccessRequests
func (h *ardHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := getActionFromContext(ctx)
	if resource.Kind != mv1.AccessRequestDefinitionGVK().Kind {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.agentCacheManager.AddAccessRequestDefinition(resource)
	}

	if action == proto.Event_DELETED {
		return h.agentCacheManager.DeleteAccessRequestDefinition(resource.Metadata.ID)
	}

	return nil
}
