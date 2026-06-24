package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type discoveryManagedApplication struct {
	marketplaceHandler
	cache agentcache.Manager
}

// NewDiscoveryManagedApplicationHandler creates a Handler for ManagedApplications for discovery agent cache building
func NewDiscoveryManagedApplicationHandler(cache agentcache.Manager) Handler {
	return &discoveryManagedApplication{
		cache: cache,
	}
}

// Handle processes events triggered for ManagedApplications during discovery cache building
func (h *discoveryManagedApplication) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.ManagedApplicationGVK().Kind {
		return nil
	}

	if action == proto.Event_DELETED {
		return h.cache.DeleteManagedApplication(resource.Metadata.ID)
	}

	app := &management.ManagedApplication{}
	err := app.FromInstance(resource)
	if err != nil {
		return err
	}

	ok := isStatusFound(app.Status)
	if !ok {
		return nil
	}

	if h.shouldProcessForAgent(app.Status, app.Metadata.State) {
		cachedApp := h.cache.GetManagedApplication(resource.Metadata.ID)
		if cachedApp == nil {
			h.cache.AddManagedApplication(resource)
		}
	}
	return nil
}
