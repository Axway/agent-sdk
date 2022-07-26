package handler

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_runtimeconfig_Handle(t *testing.T) {

	tests := []struct {
		action proto.Event_Type
		name   string
	}{
		{
			name:   "should handle a create event for a Runtimeconfig",
			action: proto.Event_CREATED,
		},
		{
			name:   "should handle an update event for a Runtimeconfig",
			action: proto.Event_UPDATED,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rtc := runtimeConfigForTest

			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

			handler := NewRuntimeConfigHandler(cm)

			cachedRTC := cm.GetRuntimeconfigResource()
			assert.Empty(t, cachedRTC)

			ri, _ := rtc.AsInstance()
			err := handler.Handle(NewEventContext(tc.action, nil, ri.Kind, ri.Name), nil, ri)

			cachedRTC = cm.GetRuntimeconfigResource()
			assert.Equal(t, ri, cachedRTC)
			assert.Nil(t, err)

		})
	}

}

var runtimeConfigForTest = mv1.AmplifyRuntimeConfig{
	ResourceMeta: v1.ResourceMeta{
		Metadata: v1.Metadata{
			ID: "123",
		},
	},
}
