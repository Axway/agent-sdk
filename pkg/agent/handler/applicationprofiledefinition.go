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

// HandleCache adds the ApplicationProfileDefinition to the cache during discoveryCache's bulk rebuild.
func (h *apdHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	h.agentCacheManager.AddApplicationProfileDefinition(resource)
	return nil
}

// GetAPIServerFields returns the fields needed to process the given event. A subresource update
// only needs a restricted fetch if the resource is already cached, so Handle can merge the
// updated subresource onto it; otherwise the full resource is needed to populate the cache from
// scratch, so no restriction is returned.
func (h *apdHandler) GetAPIServerFields(ctx context.Context, event *proto.Event) []string {
	if event.Metadata.GetSubresource() == "" {
		return nil
	}
	if existing, _ := h.agentCacheManager.GetApplicationProfileDefinitionByID(event.Payload.Metadata.Id); existing == nil {
		return nil
	}
	return []string{"name", "metadata.id", event.Metadata.GetSubresource()}
}

// Handle processes grpc events triggered for Application Profile Definitions
func (h *apdHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if action == proto.Event_DELETED {
		return h.agentCacheManager.DeleteApplicationProfileDefinition(resource.Metadata.ID)
	}

	if meta != nil && meta.Subresource != "" {
		existing, _ := h.agentCacheManager.GetApplicationProfileDefinitionByID(resource.Metadata.ID)
		if existing == nil {
			// GetAPIServerFields didn't restrict fields in this case, so resource is already the
			// full fetch - cache it directly.
			h.agentCacheManager.AddApplicationProfileDefinition(resource)
			return nil
		}
		if v := resource.GetSubResource(meta.Subresource); v != nil {
			existing.SetSubResource(meta.Subresource, v)
		}
		h.agentCacheManager.AddApplicationProfileDefinition(existing)
		return nil
	}

	h.agentCacheManager.AddApplicationProfileDefinition(resource)
	return nil
}
