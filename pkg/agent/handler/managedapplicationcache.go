package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type managedApplicationCacheHandler struct {
	marketplaceHandler
	cache agentcache.Manager
}

// NewManagedApplicationCacheHandler creates a Handler that keeps the ManagedApplication cache up
// to date from watch events. It is used by both the discovery agent and the traceability agent,
// which only need to keep their local ManagedApplication cache current rather than provision anything.
func NewManagedApplicationCacheHandler(cache agentcache.Manager) Handler {
	return &managedApplicationCacheHandler{cache: cache}
}

func (h *managedApplicationCacheHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	action := GetActionFromContext(ctx)
	if action == proto.Event_DELETED {
		h.cache.DeleteManagedApplication(event.Payload.Metadata.Id)
		return false
	}
	cachedApp := h.cache.GetManagedApplication(event.Payload.Metadata.Id)
	if cachedApp != nil {
		return false
	}

	return true
}

// HandleCache builds the ManagedApplication cache during discoveryCache's bulk rebuild.
func (h *managedApplicationCacheHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	h.cache.AddManagedApplication(resource)
	return nil
}

// Handle processes events triggered for ManagedApplications during discovery/trace agent cache building
func (h *managedApplicationCacheHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
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
