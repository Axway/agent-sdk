package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// accessRequestCacheHandler builds and maintains the AccessRequest cache from watch events. It is
// shared by the discovery agent and the traceability agent, which only need to keep their local
// AccessRequest cache up to date rather than provision anything. The traceability agent additionally
// enriches cached resources with metadata.references, which the discovery agent doesn't need.
type accessRequestCacheHandler struct {
	marketplaceHandler
	agentKind config.AgentType
	cache     agentcache.Manager
	client    client // only used by the traceability agent, to enrich cached resources
}

// NewAccessRequestCacheHandler creates a Handler for AccessRequests for discovery/trace agent cache
// building. client is only required when agentKind is config.TraceabilityAgent.
func NewAccessRequestCacheHandler(agentKind config.AgentType, cache agentcache.Manager, client client) Handler {
	return &accessRequestCacheHandler{
		agentKind: agentKind,
		cache:     cache,
		client:    client,
	}
}

func (h *accessRequestCacheHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	action := GetActionFromContext(ctx)
	if action == proto.Event_SUBRESOURCEUPDATED && event.Metadata.GetSubresource() == defs.XAgentDetails {
		return true
	}
	if action == proto.Event_DELETED {
		h.cache.DeleteAccessRequest(event.Payload.Metadata.Id)
		return false
	}

	cachedAccessReq := h.cache.GetAccessRequest(event.Payload.Metadata.Id)
	if h.agentKind == config.TraceabilityAgent {
		if cachedAccessReq != nil && len(cachedAccessReq.Metadata.References) > 0 {
			return false
		}
	} else if cachedAccessReq != nil {
		return false
	}

	return true
}

// GetAPIServerFields returns the fields needed to process the given event. A subresource update
// only needs a restricted fetch if the resource is already cached, so Handle can merge the
// updated subresource onto it; otherwise the full resource is needed to populate the cache from
// scratch, so no restriction is returned.
func (h *accessRequestCacheHandler) GetAPIServerFields(ctx context.Context, event *proto.Event) []string {
	action := GetActionFromContext(ctx)
	if action == proto.Event_SUBRESOURCEUPDATED && event.Metadata.GetSubresource() == defs.XAgentDetails {
		if existing := h.cache.GetAccessRequest(event.Payload.Metadata.Id); existing == nil {
			return nil
		}
		return []string{"name", "metadata.id", event.Metadata.GetSubresource()}
	}
	return nil
}

func (h *accessRequestCacheHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	h.cache.AddAccessRequest(resource)
	return nil
}

// Handle processes events triggered for AccessRequests during discovery/trace agent cache building
func (h *accessRequestCacheHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if action == proto.Event_SUBRESOURCEUPDATED && meta.GetSubresource() == defs.XAgentDetails {
		existing := h.cache.GetAccessRequest(resource.Metadata.ID)
		if existing == nil {
			// GetAPIServerFields didn't restrict fields in this case, so resource is already the
			// full fetch - cache it directly.
			h.cache.AddAccessRequest(resource)
			return nil
		}
		if newDetails := util.GetAgentDetails(resource); newDetails != nil {
			util.SetAgentDetails(existing, newDetails)
		}
		// update the cache with the new x-agent-details subresource
		h.cache.AddAccessRequest(existing)
		return nil
	}

	ar := &management.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		return err
	}

	ok := isStatusFound(ar.Status)
	if !ok {
		return nil
	}

	if !h.shouldProcessForAgent(ar.Status, ar.Metadata.State) {
		return nil
	}

	cachedAccessReq := h.cache.GetAccessRequest(resource.Metadata.ID)
	if h.agentKind == config.TraceabilityAgent {
		if cachedAccessReq == nil || len(cachedAccessReq.Metadata.References) == 0 {
			enriched, err := h.client.GetResource(resource.GetSelfLink() + "?embed=metadata.references")
			if err != nil || enriched == nil {
				enriched = resource
			}
			h.cache.AddAccessRequest(enriched)
		}
		return nil
	}

	if cachedAccessReq == nil {
		h.cache.AddAccessRequest(resource)
	}
	return nil
}
