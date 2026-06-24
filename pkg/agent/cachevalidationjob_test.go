package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	resources map[string][]*apiv1.ResourceInstance // keyed by URL
	err       error
}

func newMockResourceClient() *mockResourceClient {
	return &mockResourceClient{
		resources: make(map[string][]*apiv1.ResourceInstance),
	}
}

func (m *mockResourceClient) GetAPIV1ResourceCount(_ string) (int, error) {
	return 0, nil
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

// mockCVHarvester is a harvester mock with a configurable latest sequence ID,
// used to control the sequence pre-check inside cacheValidator.
type mockCVHarvester struct {
	latestSeq int64
	err       error
}

func (m *mockCVHarvester) ReceiveSyncEvents(_ context.Context, _ string, _ int64, _ chan *proto.Event) (int64, error) {
	return m.latestSeq, m.err
}

func (m *mockCVHarvester) EventCatchUp(_ context.Context, _ string, _ chan *proto.Event) error {
	return nil
}

type mockSeqProvider struct {
	seq int64
}

func (m *mockSeqProvider) GetSequence() int64   { return m.seq }
func (m *mockSeqProvider) SetSequence(id int64) { m.seq = id }

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

	svcGVK := management.APIServiceGVK()
	instGVK := management.APIServiceInstanceGVK()
	crrGVK := management.ComplianceRuntimeResultGVK()
	managedAppGVK := management.ManagedApplicationGVK()
	crdGVK := management.CredentialRequestDefinitionGVK()
	scopeName := "testEnv"

	svc1 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc1", modTime)
	svc2 := makeServerResource(svcGVK.Kind, "Environment", scopeName, "svc2", modTime)
	inst1 := makeServerResource(instGVK.Kind, "Environment", scopeName, "inst1", modTime)
	crr1 := makeServerResource(crrGVK.Kind, "Environment", scopeName, "crr1", modTime)
	managedApp1 := makeServerResource(managedAppGVK.Kind, "Environment", scopeName, "managedApp1", modTime)
	crd1 := makeServerResource(crdGVK.Kind, "Environment", scopeName, "crd1", modTime)

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
		harvester  harvester.Harvest
		sequence   events.SequenceProvider
		expectErr  error
		// expectedKinds is the set of kinds in the slice Execute returns: the
		// out-of-sync (failed) kinds when expectErr is errCacheOutOfSync, or the
		// validated (in-sync) kinds when expectErr is nil.
		expectedKinds []string
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
			expectErr:     nil,
			expectedKinds: nil,
		},
		"in sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:     nil,
			expectedKinds: []string{svcGVK.Kind},
		},
		"count mismatch": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1, svc2}
				cacheMan.setResources(crrGVK.Group, crrGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(crrGVK.Kind, scopeName, "crr1"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(crrGVK, scopeName),
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{crrGVK.Kind},
		},
		// APIService count validation is intentionally relaxed: validateKind always
		// returns true for APIService because the cache deduplicates by externalAPIID,
		// so count mismatches are expected and not treated as out-of-sync.
		"APIService higher count mismatch - not treated as out of sync will be treated with APIServiceInstances": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1, svc2}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:     nil,
			expectedKinds: []string{svcGVK.Kind},
		},
		"extra resource in cache - count mismatch detected": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[crd1.GetKindLink()] = []*apiv1.ResourceInstance{crd1}
				cacheMan.setResources(crdGVK.Group, crdGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(crdGVK.Kind, scopeName, "crd1"): modTime,
					agentcache.ResourceCacheKey(crdGVK.Kind, scopeName, "crd2"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(crdGVK, scopeName),
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{crdGVK.Kind},
		},
		// When the cache holds more APIService entries than the server, the count
		// mismatch is logged but still not treated as out-of-sync (returns true).
		"APIService extra resource in cache - still not treated as out of sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc3"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{svcGVK.Kind},
		},
		// Count-only validation cannot detect drift when the server and cache hold the
		// same number of resources but different ones (metadata/timestamp checks were
		// removed). This documents that limitation: a swapped resource is treated as in sync.
		"same count, different resource is not detected by count-only validation": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc-other"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:     nil,
			expectedKinds: []string{svcGVK.Kind},
		},
		"fetch error": {
			setup: func(client *mockResourceClient, _ *mockCacheGetter) {
				client.err = fmt.Errorf("network error")
			},
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{svcGVK.Kind},
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
			expectErr:     nil,
			expectedKinds: []string{instGVK.Kind, instGVK.Kind},
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
			expectErr:     nil,
			expectedKinds: []string{svcGVK.Kind, instGVK.Kind},
		},
		"second kind out of sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[managedApp1.GetKindLink()] = []*apiv1.ResourceInstance{managedApp1}
				client.resources[crd1.GetKindLink()] = []*apiv1.ResourceInstance{crd1}
				cacheMan.setResources(managedAppGVK.Group, managedAppGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(managedAppGVK.Kind, scopeName, "managedApp1"): modTime,
				})
				cacheMan.setResources(crdGVK.Group, crdGVK.Kind, map[string]time.Time{})
			},
			watchTopic: &management.WatchTopic{
				Spec: management.WatchTopicSpec{
					Filters: []management.WatchTopicSpecFilters{
						{Group: managedAppGVK.Group, Kind: managedAppGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName}},
						{Group: crdGVK.Group, Kind: crdGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName}},
					},
				},
			},
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{crdGVK.Kind},
		},
		// When APIServiceInstance is out of sync, the APIService filter is also
		// included in the failed set (cache is keyed by externalAPIID, so the
		// full rebuild must include both kinds).
		"second kind out of sync - APIServiceInstance fetches in APIService": {
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
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{instGVK.Kind, svcGVK.Kind},
		},
		"both kinds out of sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				client.resources[inst1.GetKindLink()] = []*apiv1.ResourceInstance{inst1}
				// both empty in cache
			},
			watchTopic: &management.WatchTopic{
				Spec: management.WatchTopicSpec{
					Filters: []management.WatchTopicSpecFilters{
						{Group: svcGVK.Group, Kind: svcGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName}},
						{Group: instGVK.Group, Kind: instGVK.Kind, Name: "*", Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName}},
					},
				},
			},
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{svcGVK.Kind, instGVK.Kind},
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
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			expectErr:     nil,
			expectedKinds: []string{svcGVK.Kind},
		},
		"ComplianceRuntimeResult out of sync is detected": {
			setup: func(client *mockResourceClient, _ *mockCacheGetter) {
				client.resources[crr1.GetKindLink()] = []*apiv1.ResourceInstance{crr1}
			},
			watchTopic:    singleScopeWatchTopic(crrGVK, scopeName),
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{crrGVK.Kind},
		},
		// Sequence in sync: per-kind count validation runs and passes, returning the
		// validated filters with no error.
		"sequence in sync - per-kind validation passes": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			harvester:     &mockCVHarvester{latestSeq: 10},
			sequence:      &mockSeqProvider{seq: 10},
			expectErr:     nil,
			expectedKinds: []string{svcGVK.Kind},
		},
		// Sequence out of sync: Execute reports the cache as out of sync without
		// running per-kind validation, returning no filters.
		"sequence out of sync - reported out of sync without per-kind validation": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			harvester:     &mockCVHarvester{latestSeq: 20},
			sequence:      &mockSeqProvider{seq: 10},
			expectErr:     errCacheOutOfSync,
			expectedKinds: nil,
		},
		// Harvester unreachable (ReceiveSyncEvents returns an error): Execute falls back
		// to per-kind count validation instead of immediately reporting out of sync.
		// Cache and server are in sync, so no error is returned.
		"harvester unreachable - falls back to per-kind validation, cache in sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[svc1.GetKindLink()] = []*apiv1.ResourceInstance{svc1}
				cacheMan.setResources(svcGVK.Group, svcGVK.Kind, map[string]time.Time{
					agentcache.ResourceCacheKey(svcGVK.Kind, scopeName, "svc1"): modTime,
				})
			},
			watchTopic:    singleScopeWatchTopic(svcGVK, scopeName),
			harvester:     &mockCVHarvester{err: fmt.Errorf("connection refused")},
			sequence:      &mockSeqProvider{seq: 10},
			expectErr:     nil,
			expectedKinds: []string{svcGVK.Kind},
		},
		// Harvester unreachable and cache is out of sync: per-kind count validation
		// runs and catches the mismatch, returning the failed filter.
		// Uses APIServiceInstance (not APIService) because APIService count validation
		// is intentionally relaxed and never reports out-of-sync.
		"harvester unreachable - falls back to per-kind validation, cache out of sync": {
			setup: func(client *mockResourceClient, cacheMan *mockCacheGetter) {
				client.resources[inst1.GetKindLink()] = []*apiv1.ResourceInstance{inst1}
				cacheMan.setResources(instGVK.Group, instGVK.Kind, map[string]time.Time{})
			},
			watchTopic:    singleScopeWatchTopic(instGVK, scopeName),
			harvester:     &mockCVHarvester{err: fmt.Errorf("connection refused")},
			sequence:      &mockSeqProvider{seq: 10},
			expectErr:     errCacheOutOfSync,
			expectedKinds: []string{instGVK.Kind},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := newMockResourceClient()
			cacheMan := newMockCacheGetter()
			if tc.setup != nil {
				tc.setup(client, cacheMan)
			}

			// Default to an in-sync harvester/sequence so per-kind count validation
			// runs. Cases that exercise the sequence pre-check set their own.
			harv := tc.harvester
			seq := tc.sequence
			if harv == nil && seq == nil {
				harv = &mockCVHarvester{latestSeq: 1}
				seq = &mockSeqProvider{seq: 1}
			}

			cv := newCacheValidator(client, tc.watchTopic, cacheMan, harv, seq)
			filters, err := cv.Execute()
			assert.Equal(t, tc.expectErr, err)

			if len(tc.expectedKinds) == 0 {
				assert.Empty(t, filters)
			} else {
				require.Len(t, filters, len(tc.expectedKinds))
				actualKinds := make([]string, len(filters))
				for i, f := range filters {
					actualKinds[i] = f.Kind
				}
				assert.ElementsMatch(t, tc.expectedKinds, actualKinds)
			}
		})
	}
}
