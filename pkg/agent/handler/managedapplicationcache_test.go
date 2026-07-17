package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestManagedApplicationCacheHandler(t *testing.T) {
	type testCase struct {
		setup           func(agentcache.Manager, Handler) // optional: pre-populate cache
		resource        func() *apiv1.ResourceInstance
		event           proto.Event_Type
		expectCached    bool
		expectCacheKeys []string
	}

	tests := map[string]testCase{
		"wrong kind is ignored": {
			resource: func() *apiv1.ResourceInstance {
				return &apiv1.ResourceInstance{
					ResourceMeta: apiv1.ResourceMeta{
						GroupVersionKind: management.EnvironmentGVK(),
					},
				}
			},
			event:           proto.Event_CREATED,
			expectCached:    false,
			expectCacheKeys: []string{},
		},
		"no status is not cached": {
			resource: func() *apiv1.ResourceInstance {
				app := &management.ManagedApplication{
					ResourceMeta: apiv1.ResourceMeta{
						Metadata: apiv1.Metadata{ID: "appId"},
						Name:     "appName",
					},
				}
				ri, _ := app.AsInstance()
				return ri
			},
			event:           proto.Event_CREATED,
			expectCached:    false,
			expectCacheKeys: []string{},
		},
		"success status is cached": {
			resource: func() *apiv1.ResourceInstance {
				app := &management.ManagedApplication{
					ResourceMeta: apiv1.ResourceMeta{
						Metadata: apiv1.Metadata{ID: "appId"},
						Name:     "appName",
					},
					Status: &apiv1.ResourceStatus{Level: "Success"},
				}
				ri, _ := app.AsInstance()
				return ri
			},
			event:        proto.Event_CREATED,
			expectCached: true,
		},
		"delete event removes from cache": {
			setup: func(cm agentcache.Manager, h Handler) {
				app := &management.ManagedApplication{
					ResourceMeta: apiv1.ResourceMeta{
						Metadata: apiv1.Metadata{ID: "appId"},
						Name:     "appName",
					},
					Status: &apiv1.ResourceStatus{Level: "Success"},
				}
				ri, _ := app.AsInstance()
				h.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
			},
			resource: func() *apiv1.ResourceInstance {
				app := &management.ManagedApplication{
					ResourceMeta: apiv1.ResourceMeta{
						Metadata: apiv1.Metadata{ID: "appId"},
						Name:     "appName",
					},
					Status: &apiv1.ResourceStatus{Level: "Success"},
				}
				ri, _ := app.AsInstance()
				return ri
			},
			event:           proto.Event_DELETED,
			expectCached:    false,
			expectCacheKeys: []string{},
		},
		"deleting state with success status is not cached": {
			resource: func() *apiv1.ResourceInstance {
				app := &management.ManagedApplication{
					ResourceMeta: apiv1.ResourceMeta{
						Metadata: apiv1.Metadata{
							ID:    "appId",
							State: apiv1.ResourceDeleting,
						},
						Name: "appName",
					},
					Status: &apiv1.ResourceStatus{Level: "Success"},
				}
				ri, _ := app.AsInstance()
				return ri
			},
			event:           proto.Event_CREATED,
			expectCached:    false,
			expectCacheKeys: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
			handler := NewManagedApplicationCacheHandler(cm)
			if tc.setup != nil {
				tc.setup(cm, handler)
			}
			ri := tc.resource()

			ctx := NewEventContext(tc.event, nil, ri.Kind, ri.Name)
			event := NewEventFromResource(tc.event, nil, ri)

			var err error
			if handler.ShouldHandle(ctx, event) {
				err = handler.Handle(ctx, nil, ri)
			}
			assert.Nil(t, err)

			if tc.expectCached {
				assert.NotNil(t, cm.GetManagedApplication("appId"))
				assert.NotNil(t, cm.GetManagedApplicationByName("appName"))
			} else if tc.expectCacheKeys != nil {
				assert.Equal(t, tc.expectCacheKeys, cm.GetManagedApplicationCacheKeys())
			}
		})
	}
}
