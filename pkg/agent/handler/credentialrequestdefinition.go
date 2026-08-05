package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type crdHandler struct {
	agentCacheManager agentcache.Manager
}

// NewCRDHandler creates a Handler for Credential Request Definitions
func NewCRDHandler(agentCacheManager agentcache.Manager) Handler {
	return &crdHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (h *crdHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	return true
}

// HandleCache adds the CredentialRequestDefinition to the cache during discoveryCache's bulk rebuild.
func (h *crdHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	h.agentCacheManager.AddCredentialRequestDefinition(resource)
	return nil
}

// GetAPIServerFields returns the fields needed to process the given event. A subresource update
// only needs a restricted fetch if the resource is already cached, so Handle can merge the
// updated subresource onto it; otherwise the full resource is needed to populate the cache from
// scratch, so no restriction is returned.
func (h *crdHandler) GetAPIServerFields(ctx context.Context, event *proto.Event) []string {
	if event.Metadata.GetSubresource() == "" {
		return nil
	}
	if existing, _ := h.agentCacheManager.GetCredentialRequestDefinitionByID(event.Payload.Metadata.Id); existing == nil {
		return nil
	}
	return []string{"name", "metadata.id", event.Metadata.GetSubresource()}
}

// Handle processes grpc events triggered for Credentials
func (h *crdHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if action == proto.Event_DELETED {
		return h.agentCacheManager.DeleteCredentialRequestDefinition(resource.Metadata.ID)
	}

	if meta != nil && meta.Subresource != "" {
		existing, _ := h.agentCacheManager.GetCredentialRequestDefinitionByID(resource.Metadata.ID)
		if existing == nil {
			// GetAPIServerFields didn't restrict fields in this case, so resource is already the
			// full fetch - cache it directly.
			h.agentCacheManager.AddCredentialRequestDefinition(resource)
			return nil
		}
		if v := resource.GetSubResource(meta.Subresource); v != nil {
			existing.SetSubResource(meta.Subresource, v)
		}
		h.agentCacheManager.AddCredentialRequestDefinition(existing)
		return nil
	}

	h.agentCacheManager.AddCredentialRequestDefinition(resource)
	return nil
}
