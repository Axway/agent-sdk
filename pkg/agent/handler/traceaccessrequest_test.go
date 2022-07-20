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

func TestTraceAccessRequestHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{}
	handler := NewTraceAccessRequestHandler(cm, c)
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
		},
	}
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
}

func TestTraceAccessRequestTraceHandler(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{}
	handler := NewTraceAccessRequestHandler(cm, c)
	ar := &management.AccessRequest{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.AccessRequestGVK(),
			Metadata: apiv1.Metadata{
				ID: "ar",
				References: []apiv1.Reference{
					{
						ID:   "instanceId",
						Name: "instance",
						Kind: "APIServiceInstance",
					},
				},
			},
			Name: "ar",
		},
		Spec: management.AccessRequestSpec{
			ManagedApplication: "app",
			ApiServiceInstance: "instance",
		},
		References: []interface{}{
			management.AccessRequestReferencesSubscription{
				Kind: defs.Subscription,
				Name: "catalog/subscription-name",
			},
		},
	}
	ri, _ := ar.AsInstance()

	// no status
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
	assert.Equal(t, []string{}, cm.GetAccessRequestCacheKeys())

	ar.Status = &apiv1.ResourceStatus{
		Level: "Success",
	}
	ri, _ = ar.AsInstance()

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

	c.getRI = &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				ID: "subscription-id",
			},
			Name: "subscription-name",
		},
	}

	err = handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
	cachedAR := cm.GetAccessRequest("ar")
	assert.NotNil(t, cachedAR)

	cachedAR = cm.GetAccessRequestByAppAndAPI("app", "api", "")
	assert.NotNil(t, cachedAR)

	err = handler.Handle(NewEventContext(proto.Event_DELETED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)

	cachedAR = cm.GetAccessRequest("ar")
	assert.Nil(t, cachedAR)

}
