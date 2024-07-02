package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type categoryHandler struct {
	agentCacheManager agentcache.Manager
}

// NewCategoryHandler creates a Handler for Categories.
func NewCategoryHandler(agentCacheManager agentcache.Manager) Handler {
	return &categoryHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (c *categoryHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != catalog.CategoryGVK().Kind {
		return nil
	}

	if action != proto.Event_DELETED {
		c.agentCacheManager.AddCategory(resource)
		return nil
	}

	return c.agentCacheManager.DeleteCategory(resource.Name)
}
