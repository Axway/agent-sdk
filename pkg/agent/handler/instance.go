package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type instanceHandler struct {
	agentCacheManager agentcache.Manager
}

// NewInstanceHandler creates a Handler for API Service Instances.
func NewInstanceHandler(agentCacheManager agentcache.Manager) Handler {
	return &instanceHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (h *instanceHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.APIServiceInstanceGVK().Kind {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED || action == proto.Event_SUBRESOURCEUPDATED {
		h.agentCacheManager.AddAPIServiceInstance(resource)
	}

	if action == proto.Event_DELETED {
		key := resource.Metadata.ID
		return h.agentCacheManager.DeleteAPIServiceInstance(key)
	}

	return nil
}
