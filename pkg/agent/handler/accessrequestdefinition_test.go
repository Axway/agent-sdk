package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestARDHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *v1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save an Access Request Definition",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: v1.Metadata{
						ID: "123",
					},
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: mv1.AccessRequestDefinitionGVK().Kind,
						},
					},
				},
			},
		},
		{
			name:     "should update an Access Request Definition",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: v1.Metadata{
						ID: "123",
					},
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: mv1.AccessRequestDefinitionGVK().Kind,
						},
					},
				},
			},
		},
		{
			name:     "should delete an Access Request Definition",
			hasError: false,
			action:   proto.Event_DELETED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: v1.Metadata{
						ID: "123",
					},
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: mv1.AccessRequestDefinitionGVK().Kind,
						},
					},
				},
			},
		},
		{
			name:     "should return nil when the kind is not an Access Request Definition",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: v1.Metadata{
						ID: "123",
					},
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: catalog.CategoryGVK().Kind,
						},
					},
				},
			},
		},
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewARDHandler(cacheManager)

			err := handler.Handle(tc.action, nil, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
