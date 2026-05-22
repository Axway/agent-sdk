package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func idpMetadataResource(id, name, scopeName, tokenEndpoint string) *apiv1.ResourceInstance {
	return &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Name: name,
			Metadata: apiv1.Metadata{
				ID: id,
				Scope: apiv1.MetadataScope{
					Name: scopeName,
					Kind: management.IdentityProviderMetadataScopes[0],
				},
			},
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Kind: management.IdentityProviderMetadataGVK().Kind,
				},
			},
		},
		Spec: map[string]interface{}{
			"tokenEndpoint": tokenEndpoint,
		},
	}
}

func TestIDPHandler_Handle(t *testing.T) {
	tests := []struct {
		name           string
		action         proto.Event_Type
		resource       *apiv1.ResourceInstance
		expectInCache  bool
		checkCacheFunc func(t *testing.T, cm agentcache.Manager, ri *apiv1.ResourceInstance)
	}{
		{
			name:   "nil resource returns nil",
			action: proto.Event_CREATED,
		},
		{
			name:   "wrong kind returns nil",
			action: proto.Event_CREATED,
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{ID: "999"},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{Kind: catalog.CategoryGVK().Kind},
					},
				},
			},
		},
		{
			name:     "create IdentityProviderMetadata with valid spec adds to cache",
			action:   proto.Event_CREATED,
			resource: idpMetadataResource("meta-1", "my-meta", "my-idp", "https://token.endpoint/token"),
			checkCacheFunc: func(t *testing.T, cm agentcache.Manager, ri *apiv1.ResourceInstance) {
				assert.NotNil(t, cm.GetIdentityProviderMetadataByTokenUrl("https://token.endpoint/token"))
			},
		},
		{
			name:     "create IdentityProviderMetadata with empty tokenEndpoint skips cache",
			action:   proto.Event_CREATED,
			resource: idpMetadataResource("meta-2", "my-meta-2", "my-idp", ""),
			checkCacheFunc: func(t *testing.T, cm agentcache.Manager, ri *apiv1.ResourceInstance) {
				assert.Nil(t, cm.GetIdentityProviderMetadataByTokenUrl(""))
			},
		},
		{
			name:     "create IdentityProviderMetadata with empty scope name skips cache",
			action:   proto.Event_CREATED,
			resource: idpMetadataResource("meta-3", "my-meta-3", "", "https://token.endpoint/token"),
			checkCacheFunc: func(t *testing.T, cm agentcache.Manager, ri *apiv1.ResourceInstance) {
				assert.Nil(t, cm.GetIdentityProviderMetadataByTokenUrl("https://token.endpoint/token"))
			},
		},
		{
			name:     "update IdentityProviderMetadata updates cache",
			action:   proto.Event_UPDATED,
			resource: idpMetadataResource("meta-4", "my-meta-4", "my-idp", "https://token.endpoint/v2"),
			checkCacheFunc: func(t *testing.T, cm agentcache.Manager, ri *apiv1.ResourceInstance) {
				assert.NotNil(t, cm.GetIdentityProviderMetadataByTokenUrl("https://token.endpoint/v2"))
			},
		},
		{
			name:     "delete IdentityProviderMetadata removes from cache",
			action:   proto.Event_DELETED,
			resource: idpMetadataResource("meta-5", "my-meta-5", "my-idp", "https://token.endpoint/v3"),
			checkCacheFunc: func(t *testing.T, cm agentcache.Manager, ri *apiv1.ResourceInstance) {
				assert.Nil(t, cm.GetIdentityProviderMetadataByTokenUrl("https://token.endpoint/v3"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

			// pre-populate cache for delete tests so there is something to remove
			if tc.action == proto.Event_DELETED && tc.resource != nil {
				if tc.resource.Kind == management.IdentityProviderMetadataGVK().Kind {
					cm.AddIdentityProviderMetadata(tc.resource)
				}
			}

			handler := NewIDPHandler(cm, nil)
			err := handler.Handle(NewEventContext(tc.action, nil, "", ""), nil, tc.resource)
			assert.Nil(t, err)

			if tc.checkCacheFunc != nil {
				tc.checkCacheFunc(t, cm, tc.resource)
			}
		})
	}
}
