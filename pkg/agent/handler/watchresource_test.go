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

type mockWatchTopicFeatures struct {
	filterList []config.ResourceFilter
}

func (m *mockWatchTopicFeatures) GetWatchResourceFilters() []config.ResourceFilter {
	return m.filterList
}

func createWatchResource(group, kind, id, name string) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1.GroupVersionKind{
				GroupKind: v1.GroupKind{
					Group: group,
					Kind:  kind,
				},
			},
			Metadata: v1.Metadata{
				ID: id,
			},
			Name: name,
		},
	}
}

func TestWatchResourceHandler(t *testing.T) {
	features := &mockWatchTopicFeatures{filterList: []config.ResourceFilter{
		{
			Group:            mv1.CredentialGVK().Group,
			Kind:             mv1.CredentialGVK().Kind,
			Name:             "*",
			IsCachedResource: true,
			EventTypes:       []config.ResourceEventType{"created"},
			Scope: &config.ResourceScope{
				Kind: mv1.EnvironmentGVK().Kind,
				Name: "test-env",
			},
		},
	}}

	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewWatchResourceHandler(cm, WithWatchTopicFeatures(features))

	res := createWatchResource(mv1.SecretGVK().Group, mv1.SecretGVK().Kind, "secret-id-1", "secret-name-1")
	// not cached resource
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, res.Kind, res.Name), nil, res)
	assert.Nil(t, err)
	assert.Equal(t, []string{}, cm.GetWatchResourceCacheKeys(mv1.SecretGVK().Group, mv1.SecretGVK().Kind))
	cachedRes := cm.GetWatchResourceByID(mv1.SecretGVK().Group, mv1.SecretGVK().Kind, "credential-id-1")
	assert.Empty(t, cachedRes)

	res = createWatchResource(mv1.CredentialGVK().Group, mv1.CredentialGVK().Kind, "credential-id-1", "credential-name-1")
	err = handler.Handle(NewEventContext(proto.Event_CREATED, nil, res.Kind, res.Name), nil, res)
	assert.Nil(t, err)
	cachedGroupKindKeys := cm.GetWatchResourceCacheKeys(mv1.CredentialGVK().Group, mv1.CredentialGVK().Kind)
	assert.NotEqual(t, []string{}, cachedGroupKindKeys)
	cachedRes = cm.GetWatchResourceByID(mv1.CredentialGVK().Group, mv1.CredentialGVK().Kind, "credential-id-1")
	assert.NotEmpty(t, cachedRes)

	cachedRes = cm.GetWatchResourceByName(mv1.CredentialGVK().Group, mv1.CredentialGVK().Kind, "credential-name-1")
	assert.NotNil(t, cachedRes)

	err = handler.Handle(NewEventContext(proto.Event_DELETED, nil, res.Kind, res.Name), nil, res)
	assert.Nil(t, err)

	cachedRes = cm.GetWatchResourceByKey(cachedGroupKindKeys[0])
	assert.Nil(t, cachedRes)
}
