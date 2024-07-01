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
	envName           string
}

// NewInstanceHandler creates a Handler for API Service Instances.
func NewInstanceHandler(agentCacheManager agentcache.Manager, envName string) Handler {
	return &instanceHandler{
		agentCacheManager: agentCacheManager,
		envName:           envName,
	}
}

func (h *instanceHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.APIServiceInstanceGVK().Kind {
		return nil
	}

	if resource.Metadata.Scope.Name != h.envName {
		return nil
	}

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddAPIServiceInstance(resource)
		return nil
	}

	return h.agentCacheManager.DeleteAPIServiceInstance(resource.Metadata.ID)
}
