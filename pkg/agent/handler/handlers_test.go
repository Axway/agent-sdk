package handler

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/stretchr/testify/assert"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
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
							Kind: apiService,
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
							Kind: apiService,
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
							Kind: apiService,
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
							Kind: apiService,
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
							Kind: category,
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

func TestNewCategoryHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *v1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save a category ResourceClient",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: category,
						},
					},
				},
			},
		},
		{
			name:     "should update a category ResourceClient",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: category,
						},
					},
				},
			},
		},
		{
			name:     "should delete a category ResourceClient",
			hasError: false,
			action:   proto.Event_DELETED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: category,
						},
					},
				},
			},
		},
		{
			name:     "should return nil when the kind is not a Category",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: apiService,
						},
					},
				},
			},
		},
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewCategoryHandler(cacheManager)

			err := handler.Handle(tc.action, nil, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}

func TestNewInstanceHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *v1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save an API Service Instance",
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
							Kind: apiServiceInstance,
						},
					},
				},
			},
		},
		{
			name:     "should update an API Service Instance",
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
							Kind: apiServiceInstance,
						},
					},
				},
			},
		},
		{
			name:     "should delete an API Service Instance",
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
							Kind: apiServiceInstance,
						},
					},
				},
			},
		},
		{
			name:     "should return nil when the kind is not an API Service Instance",
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
							Kind: category,
						},
					},
				},
			},
		},
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewInstanceHandler(cacheManager)

			err := handler.Handle(tc.action, nil, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}

func TestAgentResourceHandler(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		resource *v1.ResourceInstance
		action   proto.Event_Type
	}{
		{
			name:     "should save DiscoveryAgent",
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
							Kind: discoveryAgent,
						},
					},
				},
			},
		},
		{
			name:     "should save TraceabilityAgent",
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
							Kind: traceabilityAgent,
						},
					},
				},
			},
		},
		{
			name:     "should save GovernanceAgent",
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
							Kind: governanceAgent,
						},
					},
				},
			},
		},
		{
			name:     "should ignore processing agent resource",
			hasError: true,
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
							Kind: category,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resourceManager := &mockResourceManager{}

			handler := NewAgentResourceHandler(resourceManager)

			err := handler.Handle(tc.action, nil, tc.resource)
			if tc.hasError {
				assert.Nil(t, err)
				assert.Nil(t, resourceManager.resource)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, resourceManager.resource, tc.resource)
			}
		})
	}

}

type customHandler struct {
	err error
}

func (c *customHandler) Handle(_ proto.Event_Type, _ *proto.EventMeta, _ *v1.ResourceInstance) error {
	return c.err
}

func TestProxyHandler(t *testing.T) {
	tests := []struct {
		name     string
		handlers []Handler
		event    proto.Event_Type
		hasError bool
	}{
		{
			name:     "should not register any handlers, and return nil when Handle is called",
			event:    proto.Event_UPDATED,
			handlers: nil,
			hasError: false,
		},
		{
			name:  "should register a handler and return nil when Handle is called",
			event: proto.Event_CREATED,
			handlers: []Handler{
				&customHandler{},
			},
			hasError: false,
		},
		{
			name:  "should register two handlers and return nil when Handle is called",
			event: proto.Event_CREATED,
			handlers: []Handler{
				&customHandler{},
				&customHandler{},
			},
			hasError: false,
		},
		{
			name:  "should register a handler and return an error when Handle is called",
			event: proto.Event_CREATED,
			handlers: []Handler{
				&customHandler{err: fmt.Errorf("error")},
			},
			hasError: true,
		},
		{
			name:  "should register two handlers and return an error when Handle is called",
			event: proto.Event_CREATED,
			handlers: []Handler{
				&customHandler{},
				&customHandler{err: fmt.Errorf("error")},
			},
			hasError: true,
		},
		{
			name:  "should register two handlers and return an error when calling the first registered handler",
			event: proto.Event_CREATED,
			handlers: []Handler{
				&customHandler{err: fmt.Errorf("error")},
				&customHandler{},
			},
			hasError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ri := &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
				},
			}

			proxy := NewStreamWatchProxyHandler()

			for i, h := range tc.handlers {
				proxy.RegisterTargetHandler(fmt.Sprintf("%d", i), h)
			}

			err := proxy.Handle(tc.event, nil, ri)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}

			for i := range tc.handlers {
				proxy.UnregisterTargetHandler(fmt.Sprintf("%d", i))
			}
		})
	}
}

type mockResourceManager struct {
	resource *v1.ResourceInstance
}

func (m *mockResourceManager) SetAgentResource(agentResource *v1.ResourceInstance) {
	m.resource = agentResource
}

func (m *mockResourceManager) GetAgentResource() *v1.ResourceInstance {
	return m.resource
}

func (m *mockResourceManager) OnConfigChange(_ config.CentralConfig, _ apic.Client) {}

func (m *mockResourceManager) FetchAgentResource() error { return nil }

func (m *mockResourceManager) UpdateAgentStatus(_, _, _ string) error { return nil }