package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestTraceManagedApplicationHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewTraceManagedApplicationHandler(cm)
	ri := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1.EnvironmentGVK(),
		},
	}
	err := handler.Handle(proto.Event_CREATED, nil, ri)
	assert.Nil(t, err)
}

func TestTraceManagedApplicationHandler(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewTraceManagedApplicationHandler(cm)
	managedApp := &mv1.ManagedApplication{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: "appId",
			},
			Name: "appName",
		},
	}

	ri, _ := managedApp.AsInstance()
	// no status
	err := handler.Handle(proto.Event_CREATED, nil, ri)
	assert.Nil(t, err)
	assert.Equal(t, []string{}, cm.GetAccessRequestCacheKeys())

	managedApp.Status = &v1.ResourceStatus{
		Level: "Success",
	}

	ri, _ = managedApp.AsInstance()

	err = handler.Handle(proto.Event_CREATED, nil, ri)
	assert.Nil(t, err)
	cachedApp := cm.GetManagedApplication("appId")
	assert.NotNil(t, cachedApp)

	cachedApp = cm.GetManagedApplicationByName("appName")
	assert.NotNil(t, cachedApp)

	err = handler.Handle(proto.Event_DELETED, nil, ri)
	assert.Nil(t, err)

	cachedApp = cm.GetManagedApplication("appId")
	assert.Nil(t, cachedApp)
}
