package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type publishedProductHandler struct {
	agentCacheManager agentcache.Manager
}

// NewPublishedProductHandler creates a Handler for PublishedProduct resources
func NewPublishedProductHandler(agentCacheManager agentcache.Manager) Handler {
	return &publishedProductHandler{
		agentCacheManager: agentCacheManager,
	}
}

// Handle processes grpc events triggered for PublishedProduct resources
func (h *publishedProductHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != catalog.PublishedProductGVK().Kind {
		return nil
	}

	if action == proto.Event_DELETED {
		return h.agentCacheManager.DeletePublishedProduct(resource.Metadata.ID)
	}

	h.agentCacheManager.AddPublishedProduct(resource)
	return nil
}
