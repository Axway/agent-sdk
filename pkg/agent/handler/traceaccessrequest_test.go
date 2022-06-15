package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestTraceAccessRequestHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{}
	handler := NewTraceAccessRequestHandler(cm, c)
	ri := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1.EnvironmentGVK(),
		},
	}
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
}

func TestTraceAccessRequestTraceHandler(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{}
	handler := NewTraceAccessRequestHandler(cm, c)
	ar := &mv1.AccessRequest{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1.AccessRequestGVK(),
			Metadata: v1.Metadata{
				ID: "ar",
				References: []v1.Reference{
					{
						ID:   "instanceId",
						Name: "instance",
						Kind: "APIServiceInstance",
					},
				},
			},
			Name: "ar",
		},
		Spec: mv1.AccessRequestSpec{
			ManagedApplication: "app",
			ApiServiceInstance: "instance",
		},
		References: []interface{}{
			mv1.AccessRequestReferencesSubscription{
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

	ar.Status = &v1.ResourceStatus{
		Level: "Success",
	}
	ri, _ = ar.AsInstance()

	inst := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
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

	managedApp := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: "app",
			},
			Name: "app",
		},
	}
	cm.AddManagedApplication(managedApp)

	c.getRI = &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
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
