package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
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
	if h.agentKind == config.TraceabilityAgent && action == proto.Event_SUBRESOURCEUPDATED && event.Metadata.Subresource == defs.XAgentDetails {
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

// HandleCache builds the AccessRequest cache during discoveryCache's bulk rebuild. For the
// traceability agent, the resource is already fetched with embed=metadata.references, so no extra
// enrichment call is needed here.
func (h *accessRequestCacheHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	h.cache.AddAccessRequest(resource)
	return nil
}

// Handle processes events triggered for AccessRequests during discovery/trace agent cache building
func (h *accessRequestCacheHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if h.agentKind == config.TraceabilityAgent && action == proto.Event_SUBRESOURCEUPDATED && meta.Subresource == defs.XAgentDetails {
		// update the cache with the new x-agent-details subresource
		h.cache.AddAccessRequest(resource)
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
