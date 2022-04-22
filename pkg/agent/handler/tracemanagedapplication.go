package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type traceManagedApplication struct {
	cache agentcache.Manager
}

// NewTraceManagedApplicationHandler creates a Handler for Access Requests for trace agent
func NewTraceManagedApplicationHandler(cache agentcache.Manager) Handler {
	return &traceManagedApplication{
		cache: cache,
	}
}

// Handle processes grpc events triggered for ManagedApplications for trace agent
func (h *traceManagedApplication) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := getActionFromContext(ctx)
	if resource.Kind != mv1.ManagedApplicationGVK().Kind {
		return nil
	}

	if action == proto.Event_DELETED {
		return h.cache.DeleteManagedApplication(resource.Metadata.ID)
	}

	app := &mv1.ManagedApplication{}
	err := app.FromInstance(resource)
	if err != nil {
		return err
	}

	ok := isStatusFound(app.Status)
	if !ok {
		return nil
	}

	if shouldProcessForTrace(app.Status.Level, app.Metadata.State) {
		cachedApp := h.cache.GetManagedApplication(resource.Metadata.ID)
		if cachedApp == nil {
			h.cache.AddManagedApplication(resource)
		}
	}
	return nil
}
