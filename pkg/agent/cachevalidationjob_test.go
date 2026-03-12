package agent

import (
	"fmt"
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

// mockCacheGetter implements cacheGetter for tests
type mockCacheGetter struct {
	resources map[string]map[string]time.Time // keyed by "group/kind"
}

func newMockCacheGetter() *mockCacheGetter {
	return &mockCacheGetter{
		resources: make(map[string]map[string]time.Time),
	}
}

func (m *mockCacheGetter) GetCachedResourcesByKind(group, kind string) map[string]time.Time {
	key := group + "/" + kind
	if res, ok := m.resources[key]; ok {
		return res
	}
	return make(map[string]time.Time)
}

func (m *mockCacheGetter) setResources(group, kind string, resources map[string]time.Time) {
	m.resources[group+"/"+kind] = resources
}

// mockResourceClient implements resourceClient for tests
type mockResourceClient struct {
	resources map[string][]*apiv1.ResourceInstance // keyed by URL substring
	err       error
}

func newMockResourceClient() *mockResourceClient {
	return &mockResourceClient{
		resources: make(map[string][]*apiv1.ResourceInstance),
	}
}

func (m *mockResourceClient) GetAPIV1ResourceInstances(_ map[string]string, URL string) ([]*apiv1.ResourceInstance, error) {
	if m.err != nil {
		return nil, m.err
	}
	if res, ok := m.resources[URL]; ok {
		return res, nil
	}
	return []*apiv1.ResourceInstance{}, nil
}

func makeServerResource(kind, scopeKind, scopeName, name string, modTime time.Time) *apiv1.ResourceInstance {
	return &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Group: "management",
					Kind:  kind,
				},
				APIVersion: "v1alpha1",
			},
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{
					Kind: scopeKind,
					Name: scopeName,
				},
				Audit: apiv1.AuditMetadata{
					ModifyTimestamp: apiv1.Time(modTime),
				},
			},
			Name: name,
		},
	}
}

func TestValidatedKindsByAgentType(t *testing.T) {
	tests := []struct {
		name        string
		agentType   config.AgentType
		expectedIn  []string
		expectedOut []string
	}{
		{
			name:      "DiscoveryAgent validates discovery-relevant kinds",
			agentType: config.DiscoveryAgent,
			expectedIn: []string{
				management.APIServiceGVK().Kind,
				management.APIServiceInstanceGVK().Kind,
				management.ManagedApplicationGVK().Kind,
				management.AccessRequestGVK().Kind,
				management.AccessRequestDefinitionGVK().Kind,
				management.CredentialRequestDefinitionGVK().Kind,
				management.ApplicationProfileDefinitionGVK().Kind,
			},
			expectedOut: []string{
				management.ComplianceRuntimeResultGVK().Kind,
			},
		},
		{
			name:      "TraceabilityAgent validates traceability-relevant kinds",
			agentType: config.TraceabilityAgent,
			expectedIn: []string{
				management.APIServiceGVK().Kind,
				management.APIServiceInstanceGVK().Kind,
				management.ManagedApplicationGVK().Kind,
				management.AccessRequestGVK().Kind,
			},
			expectedOut: []string{
				management.AccessRequestDefinitionGVK().Kind,
				management.CredentialRequestDefinitionGVK().Kind,
				management.ApplicationProfileDefinitionGVK().Kind,
				management.ComplianceRuntimeResultGVK().Kind,
			},
		},
		{
			name:      "ComplianceAgent validates compliance-relevant kinds",
			agentType: config.ComplianceAgent,
			expectedIn: []string{
				management.APIServiceGVK().Kind,
				management.APIServiceInstanceGVK().Kind,
				management.ComplianceRuntimeResultGVK().Kind,
			},
			expectedOut: []string{
				management.ManagedApplicationGVK().Kind,
				management.AccessRequestGVK().Kind,
				management.AccessRequestDefinitionGVK().Kind,
				management.CredentialRequestDefinitionGVK().Kind,
				management.ApplicationProfileDefinitionGVK().Kind,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kinds := validatedKindsByAgentType(tc.agentType)
			for _, k := range tc.expectedIn {
				_, ok := kinds[k]
				assert.True(t, ok, "expected kind %s to be validated", k)
			}
			for _, k := range tc.expectedOut {
				_, ok := kinds[k]
				assert.False(t, ok, "expected kind %s to NOT be validated", k)
			}
		})
	}
}

func TestCacheValidator_Execute_SkipsUnvalidatedKinds(t *testing.T) {
	client := newMockResourceClient()
	cacheMan := newMockCacheGetter()
	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: "management",
					Kind:  management.WatchTopicGVK().Kind, // not a validated kind
					Name:  "*",
				},
			},
		},
	}
	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Nil(t, err)
}

func TestCacheValidator_Execute_InSync(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	scopeName := "testEnv"
	svcGVK := management.APIServiceGVK()

	svc := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", modTime)
	svcURL := svc.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc}

	cacheKey := agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1")
	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		cacheKey: modTime,
	})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{
						Kind: "Environment",
						Name: scopeName,
					},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Nil(t, err)
}

func TestCacheValidator_Execute_CountMismatch(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	scopeName := "testEnv"
	svcGVK := management.APIServiceGVK()

	svc1 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", modTime)
	svc2 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc2", modTime)
	svcURL := svc1.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc1, svc2}

	// cache only has 1 resource
	cacheKey := agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1")
	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		cacheKey: modTime,
	})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{
						Kind: "Environment",
						Name: scopeName,
					},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Equal(t, errCacheOutOfSync, err)
}

func TestCacheValidator_Execute_ResourceMissingFromCache(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	scopeName := "testEnv"
	svcGVK := management.APIServiceGVK()

	svc1 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", modTime)
	svcURL := svc1.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc1}

	// cache has a different resource name (same count)
	cacheKey := agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc-other")
	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		cacheKey: modTime,
	})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{
						Kind: "Environment",
						Name: scopeName,
					},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Equal(t, errCacheOutOfSync, err)
}

func TestCacheValidator_Execute_ModifyTimeMismatch(t *testing.T) {
	serverTime := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	cacheTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	scopeName := "testEnv"
	svcGVK := management.APIServiceGVK()

	svc1 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", serverTime)
	svcURL := svc1.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc1}

	cacheKey := agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1")
	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		cacheKey: cacheTime,
	})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{
						Kind: "Environment",
						Name: scopeName,
					},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Equal(t, errCacheOutOfSync, err)
}

func TestCacheValidator_Execute_ZeroTimestampsAreIgnored(t *testing.T) {
	scopeName := "testEnv"
	svcGVK := management.APIServiceGVK()

	// server has zero timestamp
	svc1 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", time.Time{})
	svcURL := svc1.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc1}

	// cache has a non-zero timestamp — should still pass because server time is zero
	cacheKey := agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1")
	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		cacheKey: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{
						Kind: "Environment",
						Name: scopeName,
					},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Nil(t, err)
}

func TestCacheValidator_Execute_ExtraResourceInCache(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	scopeName := "testEnv"
	svcGVK := management.APIServiceGVK()

	// server has 2 resources
	svc1 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", modTime)
	svc2 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc2", modTime)
	svcURL := svc1.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc1, svc2}

	// cache has 2 resources but one is different
	key1 := agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1")
	key3 := agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc3")
	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		key1: modTime,
		key3: modTime, // not on server
	})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{
						Kind: "Environment",
						Name: scopeName,
					},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Equal(t, errCacheOutOfSync, err)
}

func TestCacheValidator_Execute_FetchError(t *testing.T) {
	scopeName := "testEnv"
	svcGVK := management.APIServiceGVK()

	client := newMockResourceClient()
	client.err = fmt.Errorf("network error")

	cacheMan := newMockCacheGetter()

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{
						Kind: "Environment",
						Name: scopeName,
					},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Equal(t, errCacheOutOfSync, err)
}

func TestCacheValidator_Execute_MultiScopeCompositeKey(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	svcGVK := management.APIServiceGVK()

	// two resources with same name but different scopes
	svc1 := makeServerResource(svcGVK.Kind, "Environment", "env1", "svc1", modTime)
	svc2 := makeServerResource(svcGVK.Kind, "Environment", "env2", "svc1", modTime)
	svcURL := svc1.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc1, svc2}

	key1 := agentcache.ResourceCacheKey(svcGVK.Kind, "env1", "svc1")
	key2 := agentcache.ResourceCacheKey(svcGVK.Kind, "env2", "svc1")
	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		key1: modTime,
		key2: modTime,
	})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{
						Kind: "Environment",
						Name: "env1",
					},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Nil(t, err)
}

func TestCacheValidator_Execute_MultipleKinds(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	scopeName := "testEnv"

	svcGVK := management.APIServiceGVK()
	instGVK := management.APIServiceInstanceGVK()

	svc := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", modTime)
	svcURL := svc.GetKindLink()

	inst := makeServerResource(instGVK.Kind, "Environment", scopeName, "inst1", modTime)
	instURL := inst.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc}
	client.resources[instURL] = []*apiv1.ResourceInstance{inst}

	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
	})
	cacheMan.setResources(instGVK.Group, instGVK.Kind, map[string]time.Time{
		agentcache.ResourceCacheKey(instGVK.Kind, scopeName, "inst1"): modTime,
	})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
				},
				{
					Group: instGVK.Group,
					Kind:  instGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Nil(t, err)
}

func TestCacheValidator_Execute_SecondKindFails(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	scopeName := "testEnv"

	svcGVK := management.APIServiceGVK()
	instGVK := management.APIServiceInstanceGVK()

	svc := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", modTime)
	svcURL := svc.GetKindLink()

	inst := makeServerResource(instGVK.Kind, "Environment", scopeName, "inst1", modTime)
	instURL := inst.GetKindLink()

	client := newMockResourceClient()
	client.resources[svcURL] = []*apiv1.ResourceInstance{svc}
	client.resources[instURL] = []*apiv1.ResourceInstance{inst}

	cacheMan := newMockCacheGetter()
	cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
		agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
	})
	// instance cache is empty → count mismatch
	cacheMan.setResources(instGVK.Group, instGVK.Kind, map[string]time.Time{})

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
				},
				{
					Group: instGVK.Group,
					Kind:  instGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Equal(t, errCacheOutOfSync, err)
}

func TestCacheValidator_Execute_EmptyServerAndCache(t *testing.T) {
	scopeName := "testEnv"
	svcGVK := management.APIServiceGVK()

	client := newMockResourceClient()
	// Wire the URL so the client returns an empty slice (not nil)
	ri := apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind:  apiv1.GroupKind{Group: svcGVK.Group, Kind: svcGVK.Kind},
				APIVersion: "v1alpha1",
			},
			Metadata: apiv1.Metadata{
				Scope: apiv1.MetadataScope{Kind: "Environment", Name: scopeName},
			},
		},
	}
	client.resources[ri.GetKindLink()] = []*apiv1.ResourceInstance{}

	cacheMan := newMockCacheGetter()

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: svcGVK.Group,
					Kind:  svcGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
				},
			},
		},
	}

	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Nil(t, err)
}

func TestCacheValidator_Execute_AgentTypeFiltering(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	scopeName := "testEnv"

	crrGVK := management.ComplianceRuntimeResultGVK()
	crr := makeServerResource(crrGVK.Kind, "Environment", scopeName, "crr1", modTime)
	crrURL := crr.GetKindLink()

	client := newMockResourceClient()
	client.resources[crrURL] = []*apiv1.ResourceInstance{crr}

	// cache is empty for ComplianceRuntimeResult — would fail if the kind is validated
	cacheMan := newMockCacheGetter()

	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{
				{
					Group: crrGVK.Group,
					Kind:  crrGVK.Kind,
					Name:  "*",
					Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
				},
			},
		},
	}

	// DiscoveryAgent should skip ComplianceRuntimeResult
	cv := newCacheValidator(client, wt, cacheMan, config.DiscoveryAgent)
	err := cv.Execute()
	assert.Nil(t, err, "DiscoveryAgent should skip ComplianceRuntimeResult validation")

	// ComplianceAgent should validate ComplianceRuntimeResult and detect mismatch
	cv = newCacheValidator(client, wt, cacheMan, config.ComplianceAgent)
	err = cv.Execute()
	assert.Equal(t, errCacheOutOfSync, err, "ComplianceAgent should detect out-of-sync ComplianceRuntimeResult")
}
