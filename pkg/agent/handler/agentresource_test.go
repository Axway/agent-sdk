package handler

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

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
							Kind: catalog.CategoryGVK().Kind,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resourceManager := &mockResourceManager{}

			handler := NewAgentResourceHandler(resourceManager, nil)

			err := handler.Handle(NewEventContext(tc.action, nil, tc.resource.Kind, tc.resource.Name), nil, tc.resource)
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

type EventSyncCache interface {
	RebuildCache()
}

type mockResourceManager struct {
	resource     *v1.ResourceInstance
	rebuildCache resource.EventSyncCache
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

func (m *mockResourceManager) GetAgentResourceVersion() (string, error) {
	return "", nil
}

func (m *mockResourceManager) AddUpdateAgentDetails(key, value string) {}

func (m *mockResourceManager) SetRebuildCacheFunc(rebuildCache resource.EventSyncCache) {
	m.rebuildCache = rebuildCache
}
