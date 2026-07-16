package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type discoveryAccessRequest struct {
	marketplaceHandler
	cache agentcache.Manager
}

// NewDiscoveryAccessRequestHandler creates a Handler for AccessRequests for discovery agent cache building
func NewDiscoveryAccessRequestHandler(cache agentcache.Manager) Handler {
	return &discoveryAccessRequest{
		cache: cache,
	}
}

// Handle processes events triggered for AccessRequests during discovery cache building
func (h *discoveryAccessRequest) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.AccessRequestGVK().Kind {
		return nil
	}

	if action == proto.Event_DELETED {
		return h.cache.DeleteAccessRequest(resource.Metadata.ID)
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
		if cachedAccessReq == nil {
			h.cache.AddAccessRequest(resource)
		}
	}
	return nil
}
