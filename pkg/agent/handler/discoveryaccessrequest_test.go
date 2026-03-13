package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestDiscoveryAccessRequestHandler(t *testing.T) {
	type testCase struct {
		setup           func(agentcache.Manager, Handler)
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
				ar := &management.AccessRequest{
					ResourceMeta: apiv1.ResourceMeta{
						GroupVersionKind: management.AccessRequestGVK(),
						Metadata: apiv1.Metadata{
							ID: "arId",
							References: []apiv1.Reference{
								{
									ID:    "instanceId",
									Name:  "instance",
									Group: management.APIServiceInstanceGVK().Group,
									Kind:  management.APIServiceInstanceGVK().Kind,
								},
							},
						},
						Name: "arName",
					},
					Spec: management.AccessRequestSpec{
						ManagedApplication: "app",
						ApiServiceInstance: "instance",
					},
				}
				ri, _ := ar.AsInstance()
				return ri
			},
			event:           proto.Event_CREATED,
			expectCached:    false,
			expectCacheKeys: []string{},
		},
		"success status is cached": {
			setup: func(cm agentcache.Manager, _ Handler) {
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
				cm.AddAPIServiceInstance(inst)
				cm.AddManagedApplication(&apiv1.ResourceInstance{
					ResourceMeta: apiv1.ResourceMeta{
						Metadata: apiv1.Metadata{ID: "app"},
						Name:     "app",
					},
				})
			},
			resource: func() *apiv1.ResourceInstance {
				ar := &management.AccessRequest{
					ResourceMeta: apiv1.ResourceMeta{
						GroupVersionKind: management.AccessRequestGVK(),
						Metadata: apiv1.Metadata{
							ID: "arId",
							References: []apiv1.Reference{
								{
									ID:    "instanceId",
									Name:  "instance",
									Group: management.APIServiceInstanceGVK().Group,
									Kind:  management.APIServiceInstanceGVK().Kind,
								},
							},
						},
						Name: "arName",
					},
					Spec: management.AccessRequestSpec{
						ManagedApplication: "app",
						ApiServiceInstance: "instance",
					},
					Status: &apiv1.ResourceStatus{Level: "Success"},
				}
				ri, _ := ar.AsInstance()
				return ri
			},
			event:        proto.Event_CREATED,
			expectCached: true,
		},
		"delete event removes from cache": {
			setup: func(cm agentcache.Manager, h Handler) {
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
				cm.AddAPIServiceInstance(inst)
				cm.AddManagedApplication(&apiv1.ResourceInstance{
					ResourceMeta: apiv1.ResourceMeta{
						Metadata: apiv1.Metadata{ID: "app"},
						Name:     "app",
					},
				})
				ar := &management.AccessRequest{
					ResourceMeta: apiv1.ResourceMeta{
						GroupVersionKind: management.AccessRequestGVK(),
						Metadata: apiv1.Metadata{
							ID: "arId",
							References: []apiv1.Reference{
								{
									ID:    "instanceId",
									Name:  "instance",
									Group: management.APIServiceInstanceGVK().Group,
									Kind:  management.APIServiceInstanceGVK().Kind,
								},
							},
						},
						Name: "arName",
					},
					Spec: management.AccessRequestSpec{
						ManagedApplication: "app",
						ApiServiceInstance: "instance",
					},
					Status: &apiv1.ResourceStatus{Level: "Success"},
				}
				ri, _ := ar.AsInstance()
				h.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
			},
			resource: func() *apiv1.ResourceInstance {
				ar := &management.AccessRequest{
					ResourceMeta: apiv1.ResourceMeta{
						GroupVersionKind: management.AccessRequestGVK(),
						Metadata: apiv1.Metadata{
							ID: "arId",
							References: []apiv1.Reference{
								{
									ID:    "instanceId",
									Name:  "instance",
									Group: management.APIServiceInstanceGVK().Group,
									Kind:  management.APIServiceInstanceGVK().Kind,
								},
							},
						},
						Name: "arName",
					},
					Spec: management.AccessRequestSpec{
						ManagedApplication: "app",
						ApiServiceInstance: "instance",
					},
					Status: &apiv1.ResourceStatus{Level: "Success"},
				}
				ri, _ := ar.AsInstance()
				return ri
			},
			event:           proto.Event_DELETED,
			expectCached:    false,
			expectCacheKeys: []string{},
		},
		"deleting state with success status is not cached": {
			resource: func() *apiv1.ResourceInstance {
				ar := &management.AccessRequest{
					ResourceMeta: apiv1.ResourceMeta{
						GroupVersionKind: management.AccessRequestGVK(),
						Metadata: apiv1.Metadata{
							ID:    "arId",
							State: apiv1.ResourceDeleting,
						},
						Name: "arName",
					},
					Spec: management.AccessRequestSpec{
						ManagedApplication: "app",
						ApiServiceInstance: "instance",
					},
					Status: &apiv1.ResourceStatus{Level: "Success"},
				}
				ri, _ := ar.AsInstance()
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
			handler := NewDiscoveryAccessRequestHandler(cm)
			if tc.setup != nil {
				tc.setup(cm, handler)
			}
			ri := tc.resource()

			err := handler.Handle(NewEventContext(tc.event, nil, ri.Kind, ri.Name), nil, ri)
			assert.Nil(t, err)

			if tc.expectCached {
				assert.NotNil(t, cm.GetAccessRequest("arId"))
			} else if tc.expectCacheKeys != nil {
				assert.Equal(t, tc.expectCacheKeys, cm.GetAccessRequestCacheKeys())
			}
		})
	}
}
