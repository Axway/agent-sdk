package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewInstanceHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *apiv1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save an API Service Instance",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: apiv1.Metadata{
						ID: "123",
					},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceInstanceGVK().Kind,
						},
					},
				},
			},
		},
		{
			name:     "should update an API Service Instance",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: apiv1.Metadata{
						ID: "123",
					},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceInstanceGVK().Kind,
						},
					},
				},
			},
		},
		{
			name:     "should add another API Service Instance",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: apiv1.Metadata{
						ID: "1234",
					},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceInstanceGVK().Kind,
						},
					},
				},
			},
		},
		{
			name:     "should update an API Service Instance subresource",
			hasError: false,
			action:   proto.Event_SUBRESOURCEUPDATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: apiv1.Metadata{
						ID: "12345",
					},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceInstanceGVK().Kind,
						},
					},
				},
			},
		},
		{
			name:     "should delete an API Service Instance",
			hasError: false,
			action:   proto.Event_DELETED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: apiv1.Metadata{
						ID: "123",
					},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceInstanceGVK().Kind,
						},
					},
				},
			},
		},
		{
			name:     "should return nil when the kind is not an API Service Instance",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: apiv1.Metadata{
						ID: "123",
					},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
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
			handler := NewInstanceHandler(cacheManager, "")
			err := handler.Handle(NewEventContext(tc.action, nil, tc.resource.Kind, tc.resource.Name), nil, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				if tc.resource.ResourceMeta.GroupVersionKind.GroupKind.Kind == management.APIServiceInstanceGVK().Kind &&
					(tc.action == proto.Event_CREATED ||
						tc.action == proto.Event_UPDATED ||
						tc.action == proto.Event_SUBRESOURCEUPDATED) {
					v, err := cacheManager.GetAPIServiceInstanceByID(tc.resource.Metadata.ID)
					assert.NoError(t, err)
					assert.NotNil(t, v)
				}
				assert.Nil(t, err)
			}
		})
	}

}

func TestInstanceHandlerShouldHandle(t *testing.T) {
	tests := []struct {
		name     string
		scope    *proto.Metadata_ScopeKind
		expected bool
	}{
		{
			name:     "should handle an event scoped to the agent's environment",
			scope:    &proto.Metadata_ScopeKind{Kind: management.EnvironmentGVK().Kind, Name: "test-env"},
			expected: true,
		},
		{
			name:     "should not handle an event scoped to another environment",
			scope:    &proto.Metadata_ScopeKind{Kind: management.EnvironmentGVK().Kind, Name: "other-env"},
			expected: false,
		},
		{
			name:     "should not handle an event with no scope",
			scope:    nil,
			expected: false,
		},
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewInstanceHandler(cacheManager, "test-env")

			ctx := NewEventContext(proto.Event_CREATED, nil, management.APIServiceInstanceGVK().Kind, "inst")
			event := &proto.Event{
				Type: proto.Event_CREATED,
				Payload: &proto.ResourceInstance{
					Kind:     management.APIServiceInstanceGVK().Kind,
					Name:     "inst",
					Metadata: &proto.Metadata{Scope: tc.scope},
				},
			}

			assert.Equal(t, tc.expected, handler.ShouldHandle(ctx, event))
		})
	}
}
