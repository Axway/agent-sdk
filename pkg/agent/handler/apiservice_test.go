package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewAPISvcHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *apiv1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save a ResourceClient that has an externalAPIID attribute, and no externalAPIPrimaryKey attribute",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{
						definitions.XAgentDetails: map[string]interface{}{
							definitions.AttrExternalAPIID:   "123",
							definitions.AttrExternalAPIName: "name",
						},
					},
				},
			},
		},
		{
			name:     "should save a ResourceClient that has an externalAPIID attribute, and has the externalAPIPrimaryKey attribute",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{
						definitions.XAgentDetails: map[string]interface{}{
							definitions.AttrExternalAPIID:         "123",
							definitions.AttrExternalAPIPrimaryKey: "abc",
							definitions.AttrExternalAPIName:       "name",
						},
					},
				},
			},
		},
		{
			name:     "should fail to save the item to the cache when the externalAPIID attribute is not found",
			hasError: true,
			action:   proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{
						definitions.XAgentDetails: map[string]interface{}{},
					},
				},
			},
		},
		{
			name:     "should handle a delete action",
			hasError: false,
			action:   proto.Event_DELETED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.APIServiceGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{
						definitions.XAgentDetails: map[string]interface{}{
							definitions.AttrExternalAPIID:   "123",
							definitions.AttrExternalAPIName: "name",
						},
					},
				},
			},
		},
		{
			name:     "should return nil when the ResourceClient kind is not an APIService",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: catalog.CategoryGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{},
				},
			},
		},
	}
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewAPISvcHandler(cacheManager, "")

			err := handler.Handle(NewEventContext(tc.action, nil, tc.resource.Kind, tc.resource.Name), nil, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}

func TestAPISvcHandlerShouldHandle(t *testing.T) {
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
			handler := NewAPISvcHandler(cacheManager, "test-env")

			ctx := NewEventContext(proto.Event_CREATED, nil, management.APIServiceGVK().Kind, "svc")
			event := &proto.Event{
				Type: proto.Event_CREATED,
				Payload: &proto.ResourceInstance{
					Kind:     management.APIServiceGVK().Kind,
					Name:     "svc",
					Metadata: &proto.Metadata{Scope: tc.scope},
				},
			}

			assert.Equal(t, tc.expected, handler.ShouldHandle(ctx, event))
		})
	}
}
