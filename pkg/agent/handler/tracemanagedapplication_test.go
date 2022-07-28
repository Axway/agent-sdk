package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestTraceManagedApplicationHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewTraceManagedApplicationHandler(cm)
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
		},
	}
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
}

func TestTraceManagedApplicationHandler(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewTraceManagedApplicationHandler(cm)
	managedApp := &management.ManagedApplication{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				ID: "appId",
			},
			Name: "appName",
		},
	}

	ri, _ := managedApp.AsInstance()
	// no status
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
	assert.Equal(t, []string{}, cm.GetAccessRequestCacheKeys())

	managedApp.Status = &apiv1.ResourceStatus{
		Level: "Success",
	}

	ri, _ = managedApp.AsInstance()

	err = handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
	cachedApp := cm.GetManagedApplication("appId")
	assert.NotNil(t, cachedApp)

	cachedApp = cm.GetManagedApplicationByName("appName")
	assert.NotNil(t, cachedApp)

	err = handler.Handle(NewEventContext(proto.Event_DELETED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)

	cachedApp = cm.GetManagedApplication("appId")
	assert.Nil(t, cachedApp)
}
