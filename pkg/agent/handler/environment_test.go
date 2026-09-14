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

func TestEnvironmentHandler(t *testing.T) {
	tests := []struct {
		name            string
		hasError        bool
		credentialConfg config.CredentialConfig
		resource        *apiv1.ResourceInstance
		action          proto.Event_Type
		meta            *proto.EventMeta
	}{
		{
			name:     "should update an Environment subresource",
			hasError: false,
			action:   proto.Event_SUBRESOURCEUPDATED,
			credentialConfg: &config.CredentialConfiguration{
				ExpirationDays:      90,
				DeprovisionOnExpire: true,
			},
			meta: &proto.EventMeta{
				Subresource: management.EnvironmentPoliciesSubResourceName,
			},
			resource: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name:  "name",
					Title: "title",
					Metadata: apiv1.Metadata{
						ID: "12345",
					},
					GroupVersionKind: apiv1.GroupVersionKind{
						GroupKind: apiv1.GroupKind{
							Kind: management.EnvironmentGVK().Kind,
						},
					},
				},
			},
		},
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewEnvironmentHandler(cacheManager, tc.credentialConfg, tc.resource.Name)

			err := handler.Handle(NewEventContext(tc.action, nil, tc.resource.Kind, tc.resource.Name), tc.meta, tc.resource)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}

func TestEnvironmentHandlerShouldHandle(t *testing.T) {
	tests := []struct {
		name        string
		payloadName string
		scopeName   string
		action      proto.Event_Type
		subresource string
		expected    bool
	}{
		{
			name:        "should handle a policies subresource update on the agent's environment",
			payloadName: "test-env",
			scopeName:   "test-env",
			action:      proto.Event_SUBRESOURCEUPDATED,
			subresource: management.EnvironmentPoliciesSubResourceName,
			expected:    true,
		},
		{
			name:        "should not handle an event for another environment",
			payloadName: "other-env",
			scopeName:   "other-env",
			action:      proto.Event_SUBRESOURCEUPDATED,
			subresource: management.EnvironmentPoliciesSubResourceName,
			expected:    false,
		},
		{
			name:        "should not handle an action other than a subresource update",
			payloadName: "test-env",
			scopeName:   "test-env",
			action:      proto.Event_UPDATED,
			subresource: management.EnvironmentPoliciesSubResourceName,
			expected:    false,
		},
		{
			name:        "should not handle a subresource update for another subresource",
			payloadName: "test-env",
			scopeName:   "test-env",
			action:      proto.Event_SUBRESOURCEUPDATED,
			subresource: "status",
			expected:    false,
		},
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewEnvironmentHandler(cacheManager, &config.CredentialConfiguration{}, "test-env")

			meta := &proto.EventMeta{Subresource: tc.subresource}
			ctx := NewEventContext(tc.action, meta, management.EnvironmentGVK().Kind, tc.payloadName)
			event := &proto.Event{
				Type:     tc.action,
				Metadata: meta,
				Payload: &proto.ResourceInstance{
					Kind: management.EnvironmentGVK().Kind,
					Name: tc.payloadName,
					Metadata: &proto.Metadata{
						Scope: &proto.Metadata_ScopeKind{
							Kind: management.EnvironmentGVK().Kind,
							Name: tc.scopeName,
						},
					},
				},
			}

			assert.Equal(t, tc.expected, handler.ShouldHandle(ctx, event))
		})
	}
}
