package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

func makeTraceAR(id, name, appName, instanceID string) *management.AccessRequest {
	return &management.AccessRequest{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.AccessRequestGVK(),
			Metadata: apiv1.Metadata{
				ID: id,
				References: []apiv1.Reference{
					{
						ID:    instanceID,
						Name:  "instance",
						Group: management.APIServiceInstanceGVK().Group,
						Kind:  management.APIServiceInstanceGVK().Kind,
					},
				},
			},
			Name: name,
		},
		Spec: management.AccessRequestSpec{
			ManagedApplication: appName,
			ApiServiceInstance: "instance",
		},
		Status: &apiv1.ResourceStatus{Level: "Success"},
	}
}

func TestTraceAccessRequestTraceHandler(t *testing.T) {
	inst := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{ID: "instanceId"},
			Name:     "instance",
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID: "api",
				},
			},
		},
	}

	ar := makeTraceAR("ar", "ar", "app", "instanceId")
	ri, _ := ar.AsInstance()

	// enriched version of the same AR (simulates ?embed=metadata.references response)
	arEnriched := makeTraceAR("ar", "ar", "app", "instanceId")
	arEnriched.ResourceMeta.Embedded = map[string]apiv1.EmbeddedReferences{
		"publishedproducts": {
			References: []apiv1.EmbeddedReference{
				{Kind: "PublishedProduct", Name: "pp1"},
			},
		},
	}
	enrichedRI, _ := arEnriched.AsInstance()

	// simulates a resource that was cached via the error-fallback path (no embedded references)
	arNoRefs := &management.AccessRequest{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.AccessRequestGVK(),
			Metadata:         apiv1.Metadata{ID: "ar"},
			Name:             "ar",
		},
		Spec: management.AccessRequestSpec{
			ManagedApplication: "app",
			ApiServiceInstance: "instance",
		},
		Status: &apiv1.ResourceStatus{Level: "Success"},
	}
	riNoRefs, _ := arNoRefs.AsInstance()

	noStatusAR := &management.AccessRequest{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.AccessRequestGVK(),
			Metadata:         apiv1.Metadata{ID: "ar2"},
			Name:             "ar2",
		},
	}
	noStatusRI, _ := noStatusAR.AsInstance()

	wrongKindRI := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
		},
	}

	tests := map[string]struct {
		action         proto.Event_Type
		ri             *apiv1.ResourceInstance
		getRI          *apiv1.ResourceInstance
		getErr         error
		cachedRI       *apiv1.ResourceInstance
		expectCached   bool
		expectEnriched bool
	}{
		"wrong kind - no-op": {
			action:       proto.Event_CREATED,
			ri:           wrongKindRI,
			expectCached: false,
		},
		"no status - not cached": {
			action:       proto.Event_CREATED,
			ri:           noStatusRI,
			expectCached: false,
		},
		"created, GET succeeds - enriched RI cached": {
			action:         proto.Event_CREATED,
			ri:             ri,
			getRI:          enrichedRI,
			expectCached:   true,
			expectEnriched: true,
		},
		"created, GET fails - watch RI cached as fallback": {
			action:       proto.Event_CREATED,
			ri:           ri,
			getErr:       fmt.Errorf("network error"),
			expectCached: true,
		},
		"created, GET returns nil - watch RI cached as fallback": {
			action:       proto.Event_CREATED,
			ri:           ri,
			getRI:        nil,
			expectCached: true,
		},
		"created, already cached with references - no GET call": {
			action:       proto.Event_CREATED,
			ri:           ri,
			cachedRI:     ri,
			expectCached: true,
		},
		"updated, cached without references - re-fetches enriched RI": {
			action:         proto.Event_UPDATED,
			ri:             ri,
			cachedRI:       riNoRefs,
			getRI:          enrichedRI,
			expectCached:   true,
			expectEnriched: true,
		},
		"deleted - removed from cache": {
			action:       proto.Event_DELETED,
			ri:           ri,
			cachedRI:     ri,
			expectCached: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
			cm.AddAPIServiceInstance(inst)
			cm.AddManagedApplication(&apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{Metadata: apiv1.Metadata{ID: "app"}, Name: "app"},
			})

			if tc.cachedRI != nil {
				cm.AddAccessRequest(tc.cachedRI)
			}

			c := &mockClient{getRI: tc.getRI, getErr: tc.getErr}
			handler := NewTraceAccessRequestHandler(cm, c)

			err := handler.Handle(NewEventContext(tc.action, nil, tc.ri.Kind, tc.ri.Name), nil, tc.ri)
			assert.Nil(t, err)

			cached := cm.GetAccessRequest("ar")
			if tc.expectCached {
				assert.NotNil(t, cached)
				if tc.expectEnriched && tc.getRI != nil {
					assert.Equal(t, enrichedRI.Embedded, cached.Embedded)
				}
			} else {
				assert.Nil(t, cached)
			}
		})
	}
}
