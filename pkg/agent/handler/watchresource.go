package handler

import (
	"context"
	"fmt"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type watchResourceHandler struct {
	agentCacheManager agentcache.Manager
	watchGroupKindMap map[string]bool
}

type watchTopicFeatures interface {
	GetWatchResourceFilters() []config.ResourceFilter
}

func getWatchResourceKey(group, kind string) string {
	return fmt.Sprintf("%s:%s", group, kind)
}

// NewWatchResourceHandler creates a Handler for custom watch resources to store resource in agent cache
func NewWatchResourceHandler(agentCacheManager agentcache.Manager, feature watchTopicFeatures) Handler {
	watchGroupKindMap := make(map[string]bool)

	filters := feature.GetWatchResourceFilters()
	for _, filter := range filters {
		key := getWatchResourceKey(filter.Group, filter.Kind)
		watchGroupKindMap[key] = filter.IsCachedResource
	}

	return &watchResourceHandler{
		agentCacheManager: agentCacheManager,
		watchGroupKindMap: watchGroupKindMap,
	}
}

func (h *watchResourceHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	key := getWatchResourceKey(resource.Group, resource.Kind)
	ok := h.watchGroupKindMap[key]
	if !ok {
		return nil
	}

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddWatchResource(resource)
		return nil
	}

	return h.agentCacheManager.DeleteWatchResource(resource.Group, resource.Kind, resource.Metadata.ID)
}
