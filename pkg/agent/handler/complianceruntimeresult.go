package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type crrHandler struct {
	agentCacheManager agentcache.Manager
}

// NewCRRHandler creates a Handler for Compliance Runtime Results
func NewCRRHandler(agentCacheManager agentcache.Manager) Handler {
	return &crrHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (h *crrHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	return true
}

// HandleCache adds the ComplianceRuntimeResult to the cache during discoveryCache's bulk rebuild.
func (h *crrHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	h.agentCacheManager.AddComplianceRuntimeResult(resource)
	return nil
}

// GetAPIServerFields returns the fields needed to process the given event. A subresource update
// only needs a restricted fetch if the resource is already cached, so Handle can merge the
// updated subresource onto it; otherwise the full resource is needed to populate the cache from
// scratch, so no restriction is returned.
func (h *crrHandler) GetAPIServerFields(ctx context.Context, event *proto.Event) []string {
	if event.Metadata.GetSubresource() == "" {
		return nil
	}
	if existing, _ := h.agentCacheManager.GetComplianceRuntimeResultByID(event.Payload.Metadata.Id); existing == nil {
		return nil
	}
	return []string{"name", "metadata.id", event.Metadata.GetSubresource()}
}

// Handle processes grpc events triggered for Compliance Runtime Results
func (h *crrHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.ComplianceRuntimeResultGVK().Kind {
		return nil
	}

	if action == proto.Event_DELETED {
		return h.agentCacheManager.DeleteComplianceRuntimeResult(resource.Metadata.ID)
	}

	if meta != nil && meta.Subresource != "" {
		existing, _ := h.agentCacheManager.GetComplianceRuntimeResultByID(resource.Metadata.ID)
		if existing == nil {
			// GetAPIServerFields didn't restrict fields in this case, so resource is already the
			// full fetch - cache it directly.
			h.agentCacheManager.AddComplianceRuntimeResult(resource)
			return nil
		}
		if v := resource.GetSubResource(meta.Subresource); v != nil {
			existing.SetSubResource(meta.Subresource, v)
		}
		h.agentCacheManager.AddComplianceRuntimeResult(existing)
		return nil
	}

	h.agentCacheManager.AddComplianceRuntimeResult(resource)
	return nil
}
