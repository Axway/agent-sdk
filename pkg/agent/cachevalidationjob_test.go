package agent

import (
	"fmt"
	"strings"
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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

func (m *mockCacheGetter) GetCachedResourcesByKind(group, kind, scopeName string) map[string]time.Time {
	key := group + "/" + kind
	res, ok := m.resources[key]
	if !ok {
		return make(map[string]time.Time)
	}
	if scopeName == "" {
		return res
	}
	filtered := make(map[string]time.Time)
	for k, v := range res {
		// key format: kind/scopeName/name
		parts := strings.SplitN(k, "/", 3)
		if len(parts) == 3 && parts[1] == scopeName {
			filtered[k] = v
		}
	}
	return filtered
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

func TestCacheValidator_Execute(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	serverTime := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)

	svcGVK := management.APIServiceGVK()
	instGVK := management.APIServiceInstanceGVK()
	crrGVK := management.ComplianceRuntimeResultGVK()
	scopeName := "testEnv"

	svc1 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", modTime)
	svc2 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc2", modTime)
	inst1 := makeServerResource(instGVK.Kind, "Environment", scopeName, "inst1", modTime)
	crr1 := makeServerResource(crrGVK.Kind, "Environment", scopeName, "crr1", modTime)

	singleScopeWatchTopic := func(gvk apiv1.GroupVersionKind, scopeName string) *management.WatchTopic {
		return &management.WatchTopic{
			Spec: management.WatchTopicSpec{
				Filters: []management.WatchTopicSpecFilters{
					{
						Group: gvk.Group,
						Kind:  gvk.Kind,
						Name:  "*",
						Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
					},
				},
			},
		}
	}

	type testCase struct {
		setup      func(*mockResourceClient, *mockCacheGetter)
		watchTopic *management.WatchTopic
		expectErr  error
	}

	tests := map[string]testCase{
		"skips non-cached kinds": {
			watchTopic: &management.WatchTopic{
				Spec: management.WatchTopicSpec{
					Filters: []management.WatchTopicSpecFilters{
						{Group: "management", Kind: management.WatchTopicGVK().Kind, Name: "*"},
					},
				},
			},
			expectErr: nil,
		},
		"in sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
			},
			watchTopic: singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:  nil,
		},
		"count mismatch": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1, svc2}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
			},
			watchTopic: singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:  errCacheOutOfSync,
		},
		"resource missing from cache": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc-other"): modTime,
				})
			},
			watchTopic: singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:  errCacheOutOfSync,
		},
		"modify time mismatch": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				svcNewer := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", serverTime)
				client.resources[svcNewer.GetKindLink()] = []*apiv1.ResourceInstance{svcNewer}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
			},
			watchTopic: singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:  errCacheOutOfSync,
		},
		"zero timestamps are ignored": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				svcZero := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", time.Time{})
				client.resources[svcZero.GetKindLink()] = []*apiv1.ResourceInstance{svcZero}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				})
			},
			watchTopic: singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:  nil,
		},
		"extra resource in cache": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1, svc2}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc3"): modTime,
				})
			},
			watchTopic: singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:  errCacheOutOfSync,
		},
		"fetch error": {
			setup: func(client *mockResourceClient, _ *mockCacheGetter) {
				client.err = fmt.Errorf("network error")
			},
			watchTopic: singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:  errCacheOutOfSync,
		},
		"APIServiceInstances from multiple scopes are each validated independently": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				instEnv1 := makeServerResource(instGVK.Kind, "Environment", "env1", "inst1", modTime)
				instEnv2 := makeServerResource(instGVK.Kind, "Environment", "env2", "inst2", modTime)
				client.resources[instEnv1.GetKindLink()] = []*apiv1.ResourceInstance{instEnv1}
				client.resources[instEnv2.GetKindLink()] = []*apiv1.ResourceInstance{instEnv2}
				cacheMan.setResources(instGVK.Group, instGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(instGVK.Kind, "env1", "inst1"): modTime,
					agentcache.ResourceCacheKey(instGVK.Kind, "env2", "inst2"): modTime,
				})
			},
			watchTopic: &management.WatchTopic{
				Spec: management.WatchTopicSpec{
					Filters: []management.WatchTopicSpecFilters{
						{Group: instGVK.Group, Kind: instGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: "env1"}},
						{Group: instGVK.Group, Kind: instGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: "env2"}},
					},
				},
			},
			expectErr: nil,
		},
		"multiple kinds in sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				client.resources[inst1.GetKindLink()] = []*apiv1.ResourceInstance{inst1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
				cacheMan.setResources(instGVK.Group, instGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(instGVK.Kind, scopeName, "inst1"): modTime,
				})
			},
			watchTopic: &management.WatchTopic{
				Spec: management.WatchTopicSpec{
					Filters: []management.WatchTopicSpecFilters{
						{Group: svcGVK.Group, Kind: svcGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName}},
						{Group: instGVK.Group, Kind: instGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName}},
					},
				},
			},
			expectErr: nil,
		},
		"second kind out of sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				client.resources[inst1.GetKindLink()] = []*apiv1.ResourceInstance{inst1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
				cacheMan.setResources(instGVK.Group, instGVK.Kind, map[string]time.Time{})
			},
			watchTopic: &management.WatchTopic{
				Spec: management.WatchTopicSpec{
					Filters: []management.WatchTopicSpecFilters{
						{Group: svcGVK.Group, Kind: svcGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName}},
						{Group: instGVK.Group, Kind: instGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName}},
					},
				},
			},
			expectErr: errCacheOutOfSync,
		},
		"empty server and cache": {
			setup: func(client *mockResourceClient, _ *mockCacheGetter) {
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
			},
			watchTopic: singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:  nil,
		},
		"ComplianceRuntimeResult out of sync is detected": {
			setup: func(client *mockResourceClient, _ *mockCacheGetter) {
				client.resources[crr1.GetKindLink()] = []*apiv1.ResourceInstance{crr1}
			},
			watchTopic: singleScopeWatchTopic(crrGVK, scopeName),
			expectErr:  errCacheOutOfSync,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := newMockResourceClient()
			cacheMan := newMockCacheGetter()
			if tc.setup != nil {
				tc.setup(client, cacheMan)
			}
			cv := newCacheValidator(client, tc.watchTopic, cacheMan)
			err := cv.Execute()
			assert.Equal(t, tc.expectErr, err)
		})
	}
}
