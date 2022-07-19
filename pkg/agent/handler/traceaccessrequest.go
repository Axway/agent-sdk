package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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

// Handle processes grpc events triggered for AccessRequests for trace agent
func (h *traceAccessRequestHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != mv1.AccessRequestGVK().Kind {
		return nil
	}

	if action == proto.Event_DELETED {
		return h.cache.DeleteAccessRequest(resource.Metadata.ID)
	}

	ar := &mv1.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		return err
	}

	ok := isStatusFound(ar.Status)
	if !ok {
		return nil
	}

	if h.shouldProcessForTrace(ar.Status, ar.Metadata.State) {
		cachedAccessReq := h.cache.GetAccessRequest(resource.Metadata.ID)
		if cachedAccessReq == nil {
			h.cache.AddAccessRequest(resource)
		}
	}
	return nil
}
