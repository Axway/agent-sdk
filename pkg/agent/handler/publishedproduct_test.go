package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func publishedProductRI(name, id string) *apiv1.ResourceInstance {
	return &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name:     name,
			Metadata: apiv1.Metadata{ID: id},
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{Kind: catalog.PublishedProductGVK().Kind},
			},
		},
	}
}

func TestPublishedProductHandler(t *testing.T) {
	tests := map[string]struct {
		action   proto.Event_Type
		resource *apiv1.ResourceInstance
		setup    func(agentcache.Manager)
		wantErr  bool
	}{
		"create caches the resource": {
			action:   proto.Event_CREATED,
			resource: publishedProductRI("product-1", "id-1"),
		},
		"update caches the resource": {
			action:   proto.Event_UPDATED,
			resource: publishedProductRI("product-2", "id-2"),
		},
		"delete removes from cache": {
			action:   proto.Event_DELETED,
			resource: publishedProductRI("product-3", "id-3"),
			setup: func(m agentcache.Manager) {
				m.AddPublishedProduct(publishedProductRI("product-3", "id-3"))
			},
		},
		"wrong kind is ignored": {
			action: proto.Event_UPDATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:     "other",
					Metadata: apiv1.Metadata{ID: "id-other"},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{Kind: management.AccessRequestGVK().Kind},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
			if tc.setup != nil {
				tc.setup(cacheManager)
			}
			h := NewPublishedProductHandler(cacheManager)
			err := h.Handle(NewEventContext(tc.action, nil, tc.resource.Kind, tc.resource.Name), nil, tc.resource)
			if tc.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
