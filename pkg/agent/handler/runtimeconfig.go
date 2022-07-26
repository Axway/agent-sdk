package handler

import (
	"context"
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type runtimeconfig struct {
	cache agentcache.Manager
}

// NewRuntimeConfigHandler creates a Handler for RuntimeConfig
func NewRuntimeConfigHandler(cache agentcache.Manager) Handler {
	return &runtimeconfig{
		cache: cache,
	}
}

// Handle processes grpc events triggered for RuntimeConfig
func (h *runtimeconfig) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != mv1.AmplifyRuntimeConfigGVK().Kind {
		return nil
	}

	// add or update the cache with the access request
	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.cache.UpdateRuntimeconfigResource(resource)
	}

	return nil
}
