package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewAPISvcHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *v1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save a ResourceClient that has an externalAPIID attribute, and no externalAPIPrimaryKey attribute",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: mv1.APIServiceGVK().Kind,
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
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: mv1.APIServiceGVK().Kind,
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
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: mv1.APIServiceGVK().Kind,
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
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: mv1.APIServiceGVK().Kind,
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
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
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
			handler := NewAPISvcHandler(cacheManager)

			err := handler.Handle(tc.action, nil, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}
