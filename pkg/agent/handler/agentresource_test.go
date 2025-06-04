package handler

import (
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

type fakeSampler struct {
	enabled bool
}

func (f *fakeSampler) EnableSampling(samplingLimit int32, samplingEndTime time.Time, endpointsInfo map[string]management.TraceabilityAgentAgentstateSamplingEndpoints) {
	f.enabled = true
}

type fakeAgent struct {
	triggeredCompliance   bool
	triggeredTraceability bool
}

func (f *fakeAgent) TriggerProcessing() {
	f.triggeredCompliance = true
}

func (f *fakeAgent) TriggerTraceability() {
	f.triggeredTraceability = true
}

func TestAgentResourceHandler(t *testing.T) {
	tests := []struct {
		name                         string
		hasError                     bool
		resource                     *v1.ResourceInstance
		expectResourceUpdate         bool
		subresName                   string
		action                       proto.Event_Type
		fakeAgentHandler             *fakeAgent
		expectComplianceProcessing   bool
		expectTraceabilityProcessing bool
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
							Kind: management.DiscoveryAgentGVK().Kind,
						},
					},
				},
			},
			expectResourceUpdate: true,
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
							Kind: management.TraceabilityAgentGVK().Kind,
						},
					},
				},
			},
			expectResourceUpdate: true,
			fakeAgentHandler:     &fakeAgent{},
		},
		{
			name:       "should not trigger TraceabilityAgent processing on agent state subresource when sampling disabled",
			hasError:   false,
			action:     proto.Event_SUBRESOURCEUPDATED,
			subresName: management.TraceabilityAgentAgentstateSubResourceName,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: v1.Metadata{
						ID: "123",
					},
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: management.TraceabilityAgentGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{
						management.TraceabilityAgentAgentstateSubResourceName: map[string]interface{}{
							"sampling": map[string]interface{}{
								"enabled": false,
								"limit":   100,
								"endTime": v1.Time(time.Now()),
							},
						},
					},
				},
			},
			fakeAgentHandler: &fakeAgent{},
		},
		{
			name:       "should trigger TraceabilityAgent processing on agent state subresource when sampling enabled",
			hasError:   false,
			action:     proto.Event_SUBRESOURCEUPDATED,
			subresName: management.TraceabilityAgentAgentstateSubResourceName,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: v1.Metadata{
						ID: "123",
					},
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: management.TraceabilityAgentGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{
						management.TraceabilityAgentAgentstateSubResourceName: map[string]interface{}{
							"sampling": map[string]interface{}{
								"enabled": true,
								"limit":   100,
								"endTime": v1.Time(time.Now()),
							},
						},
					},
				},
			},
			fakeAgentHandler:             &fakeAgent{},
			expectTraceabilityProcessing: true,
		},
		{
			name:     "should save ComplianceAgent",
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
							Kind: management.ComplianceAgentGVK().Kind,
						},
					},
				},
			},
			expectResourceUpdate: true,
			fakeAgentHandler:     &fakeAgent{},
		},
		{
			name:       "should not trigger ComplianceAgent processing on x-agent-details subresource",
			hasError:   false,
			action:     proto.Event_SUBRESOURCEUPDATED,
			subresName: definitions.XAgentDetails,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: v1.Metadata{
						ID: "123",
					},
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: management.ComplianceAgentGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{
						definitions.XAgentDetails: map[string]interface{}{
							definitions.ComplianceAgentTrigger: "false",
						},
					},
				},
			},
			fakeAgentHandler: &fakeAgent{},
		},
		{
			name:       "should trigger ComplianceAgent processing",
			hasError:   false,
			action:     proto.Event_SUBRESOURCEUPDATED,
			subresName: definitions.XAgentDetails,
			resource: &v1.ResourceInstance{
				ResourceMeta: v1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: v1.Metadata{
						ID: "123",
					},
					GroupVersionKind: v1.GroupVersionKind{
						GroupKind: v1.GroupKind{
							Kind: management.ComplianceAgentGVK().Kind,
						},
					},
					SubResources: map[string]interface{}{
						definitions.XAgentDetails: map[string]interface{}{
							definitions.ComplianceAgentTrigger: "true",
						},
					},
				},
			},
			fakeAgentHandler:           &fakeAgent{},
			expectComplianceProcessing: true,
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
			if tc.fakeAgentHandler != nil {
				resourceManager.fakeHandler = tc.fakeAgentHandler
			}

			sampler := &fakeSampler{}
			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
			handler := NewAgentResourceHandler(resourceManager, sampler, cm)

			err := handler.Handle(NewEventContext(tc.action, nil, tc.resource.Kind, tc.resource.Name), &proto.EventMeta{Subresource: tc.subresName}, tc.resource)
			if tc.hasError {
				assert.Nil(t, err)
				assert.Nil(t, resourceManager.resource)
			}
			// resource update
			if tc.expectResourceUpdate {
				assert.Nil(t, err)
				assert.Equal(t, resourceManager.resource, tc.resource)
			}

			// agent processing
			switch {
			case tc.fakeAgentHandler != nil && tc.expectComplianceProcessing:
				assert.True(t, tc.fakeAgentHandler.triggeredCompliance)
				assert.False(t, tc.fakeAgentHandler.triggeredTraceability)
				assert.False(t, sampler.enabled)
			case tc.fakeAgentHandler != nil && tc.expectTraceabilityProcessing:
				assert.False(t, tc.fakeAgentHandler.triggeredCompliance)
				assert.True(t, tc.fakeAgentHandler.triggeredTraceability)
				assert.True(t, sampler.enabled)
			case tc.fakeAgentHandler != nil:
				assert.False(t, tc.fakeAgentHandler.triggeredCompliance)
				assert.False(t, tc.fakeAgentHandler.triggeredTraceability)
				assert.False(t, sampler.enabled)
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
	fakeHandler  interface{}
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

func (m *mockResourceManager) RegisterHandler(handler interface{}) {}

func (m *mockResourceManager) GetHandler() interface{} {
	return m.fakeHandler
}
