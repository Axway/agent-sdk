package handler

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)


const apiServiceInstance = "APIServiceInstance"

type instanceHandler struct {
	agentCacheManager agentcache.Manager
}

// NewInstanceHandler creates a Handler for API Service Instances.
func NewInstanceHandler(agentCacheManager agentcache.Manager) Handler {
	return &instanceHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (h *instanceHandler) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != apiServiceInstance {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.agentCacheManager.AddAPIServiceInstance(resource)
	}

	if action == proto.Event_DELETED {
		key := resource.Metadata.ID
		return h.agentCacheManager.DeleteAPIServiceInstance(key)
	}

	return nil
}
