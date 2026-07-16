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
	kinds             map[string]bool
}

type watchTopicFeatures interface {
	GetWatchResourceFilters() []config.ResourceFilter
}

func getWatchResourceKey(group, kind string) string {
	return fmt.Sprintf("%s:%s", group, kind)
}

type watchTopicOptions func(s *watchResourceHandler)

func WithWatchTopicFeatures(feature watchTopicFeatures) watchTopicOptions {
	return func(w *watchResourceHandler) {
		filters := feature.GetWatchResourceFilters()
		for _, filter := range filters {
			key := getWatchResourceKey(filter.Group, filter.Kind)
			w.watchGroupKindMap[key] = filter.IsCachedResource
			w.kinds[filter.Kind] = true
		}
	}
}

func WithWatchTopicGroupKind(groupKinds []v1.GroupKind) watchTopicOptions {
	return func(w *watchResourceHandler) {
		for _, gk := range groupKinds {
			key := getWatchResourceKey(gk.Group, gk.Kind)
			w.watchGroupKindMap[key] = true
			w.kinds[gk.Kind] = true
		}
	}
}

// NewWatchResourceHandler creates a Handler for custom watch resources to store resource in agent cache
func NewWatchResourceHandler(agentCacheManager agentcache.Manager, opts ...watchTopicOptions) Handler {
	w := &watchResourceHandler{
		agentCacheManager: agentCacheManager,
		watchGroupKindMap: map[string]bool{},
		kinds:             map[string]bool{},
	}

	for _, o := range opts {
		o(w)
	}

	return w
}

func (h *watchResourceHandler) Kinds() []string {
	kinds := make([]string, 0, len(h.kinds))
	for kind := range h.kinds {
		kinds = append(kinds, kind)
	}
	return kinds
}

func (h *watchResourceHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	key := getWatchResourceKey(event.Payload.Group, event.Payload.Kind)
	if ok := h.watchGroupKindMap[key]; !ok {
		return false
	}

	return true
}

// HandleCache adds the watch resource to the cache during discoveryCache's bulk rebuild.
func (h *watchResourceHandler) HandleCache(resource *v1.ResourceInstance) error {
	h.agentCacheManager.AddWatchResource(resource)
	return nil
}

func (h *watchResourceHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddWatchResource(resource)
		return nil
	}

	return h.agentCacheManager.DeleteWatchResource(resource.Group, resource.Kind, resource.Metadata.ID)
}
