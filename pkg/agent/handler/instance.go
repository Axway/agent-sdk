package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
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

func (h *instanceHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	if event.Payload.Metadata.Scope.Name != h.envName {
		return false
	}

	return true
}

// HandleCache adds the API Service Instance to the cache during discoveryCache's bulk rebuild.
func (h *instanceHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	h.agentCacheManager.AddAPIServiceInstance(resource)
	return nil
}

// GetAPIServerFields returns the fields needed to process the given event. A subresource update
// only needs a restricted fetch if the resource is already cached, so Handle can merge the
// updated subresource onto it; otherwise the full resource is needed to populate the cache from
// scratch, so no restriction is returned.
func (h *instanceHandler) GetAPIServerFields(ctx context.Context, event *proto.Event) []string {
	if event.Metadata.Subresource == "" {
		return nil
	}
	if existing, _ := h.agentCacheManager.GetAPIServiceInstanceByID(event.Payload.Metadata.Id); existing == nil {
		return nil
	}
	return []string{"name", "metadata.id", event.Metadata.Subresource}
}

func (h *instanceHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)

	if action == proto.Event_DELETED {
		return h.agentCacheManager.DeleteAPIServiceInstance(resource.Metadata.ID)
	}

	if meta != nil && meta.Subresource != "" {
		existing, _ := h.agentCacheManager.GetAPIServiceInstanceByID(resource.Metadata.ID)
		if existing == nil {
			// GetAPIServerFields didn't restrict fields in this case, so resource is already the
			// full fetch - cache it directly.
			h.agentCacheManager.AddAPIServiceInstance(resource)
			return nil
		}
		if v := resource.GetSubResource(meta.Subresource); v != nil {
			existing.SetSubResource(meta.Subresource, v)
		}
		h.agentCacheManager.AddAPIServiceInstance(existing)
		return nil
	}

	h.agentCacheManager.AddAPIServiceInstance(resource)
	return nil
}
