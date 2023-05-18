package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	environment "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type environmentHandler struct {
	agentCacheManager agentcache.Manager
}

// NewEnvironmentHandler creates a Handler for Environments.
func NewEnvironmentHandler(agentCacheManager agentcache.Manager) Handler {
	return &environmentHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (c *environmentHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != environment.EnvironmentGVK().Kind {
		return nil
	}

	if action == proto.Event_UPDATED {
		c.agentCacheManager.AddEnvironment(resource)
	}

	return nil
}
