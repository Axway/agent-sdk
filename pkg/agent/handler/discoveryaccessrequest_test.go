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

func TestDiscoveryAccessRequestHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewDiscoveryAccessRequestHandler(cm)
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
		},
	}
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
}

func TestDiscoveryAccessRequestHandler(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewDiscoveryAccessRequestHandler(cm)

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
	// no status - should not be cached
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
	assert.Equal(t, []string{}, cm.GetAccessRequestCacheKeys())

	ar.Status = &apiv1.ResourceStatus{
		Level: "Success",
	}

	ri, _ = ar.AsInstance()

	// add instance and app to cache for access request dependencies
	inst := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				ID: "instanceId",
			},
			Name: "instance",
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID: "api",
				},
			},
		},
	}
	cm.AddAPIServiceInstance(inst)

	managedApp := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				ID: "app",
			},
			Name: "app",
		},
	}
	cm.AddManagedApplication(managedApp)

	// success status - should be cached
	err = handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
	cachedAR := cm.GetAccessRequest("arId")
	assert.NotNil(t, cachedAR)

	// delete event - should remove from cache
	err = handler.Handle(NewEventContext(proto.Event_DELETED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)

	cachedAR = cm.GetAccessRequest("arId")
	assert.Nil(t, cachedAR)
}

func TestDiscoveryAccessRequestHandler_deleting_state(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewDiscoveryAccessRequestHandler(cm)

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
		Status: &apiv1.ResourceStatus{
			Level: "Success",
		},
	}

	ri, _ := ar.AsInstance()
	// deleting state with success status - should NOT be cached
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
	assert.Equal(t, []string{}, cm.GetAccessRequestCacheKeys())
}
