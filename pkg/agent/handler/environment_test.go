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
