package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
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

func (h *instanceHandler) Kinds() []string {
	return []string{management.APIServiceInstanceGVK().Kind}
}

func (h *instanceHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	if event.Payload.Kind != management.APIServiceInstanceGVK().Kind || event.Payload.Metadata.Scope.Name != h.envName {
		return false
	}

	return true
}

func (h *instanceHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddAPIServiceInstance(resource)
		return nil
	}

	return h.agentCacheManager.DeleteAPIServiceInstance(resource.Metadata.ID)
}
