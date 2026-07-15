package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type traceAccessRequestHandler struct {
	marketplaceHandler
	cache  agentcache.Manager
	client client
}

// NewTraceAccessRequestHandler creates a Handler for Access Requests for trace agent
func NewTraceAccessRequestHandler(cache agentcache.Manager, client client) Handler {
	return &traceAccessRequestHandler{
		cache:  cache,
		client: client,
	}
}

func (h *traceAccessRequestHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	action := GetActionFromContext(ctx)
	if action == proto.Event_SUBRESOURCEUPDATED && event.Metadata.Subresource == defs.XAgentDetails {
		return true
	}
	cachedAccessReq := h.cache.GetAccessRequest(event.Payload.Metadata.Id)
	if cachedAccessReq != nil && len(cachedAccessReq.Metadata.References) > 0 {
		return false
	}

	return true
}

// Handle processes grpc events triggered for AccessRequests for trace agent
func (h *traceAccessRequestHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)

	if action == proto.Event_DELETED {
		return h.cache.DeleteAccessRequest(resource.Metadata.ID)
	}

	if action == proto.Event_SUBRESOURCEUPDATED && meta.Subresource == defs.XAgentDetails {
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

	if h.shouldProcessForAgent(ar.Status, ar.Metadata.State) {
		cachedAccessReq := h.cache.GetAccessRequest(resource.Metadata.ID)
		if cachedAccessReq == nil || len(cachedAccessReq.Metadata.References) == 0 {
			enriched, err := h.client.GetResource(resource.GetSelfLink() + "?embed=metadata.references")
			if err != nil || enriched == nil {
				enriched = resource
			}
			h.cache.AddAccessRequest(enriched)
		}
	}
	return nil
}
