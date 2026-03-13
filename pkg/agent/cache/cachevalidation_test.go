package cache

import (
	"testing"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func makeRI(group, kind, scopeKind, scopeName, name, id string, modTime time.Time) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1.GroupVersionKind{
				GroupKind: v1.GroupKind{
					Group: group,
					Kind:  kind,
				},
				APIVersion: "v1alpha1",
			},
			Metadata: v1.Metadata{
				ID: id,
				Scope: v1.MetadataScope{
					Kind: scopeKind,
					Name: scopeName,
				},
				Audit: v1.AuditMetadata{
					ModifyTimestamp: v1.Time(modTime),
				},
			},
			Name: name,
		},
	}
}

func makeAPIServiceRI(scopeName, name, apiID string, modTime time.Time) *v1.ResourceInstance {
	ri := makeRI("management", management.APIServiceGVK().Kind, "Environment", scopeName, name, apiID, modTime)
	ri.SubResources = map[string]interface{}{
		defs.XAgentDetails: map[string]interface{}{
			defs.AttrExternalAPIID:   apiID,
			defs.AttrExternalAPIName: name,
		},
	}
	return ri
}

func TestResourceCacheKey(t *testing.T) {
	tests := map[string]struct {
		kind     string
		scope    string
		name     string
		expected string
	}{
		"with scope":  {kind: "APIService", scope: "env1", name: "svc1", expected: "APIService/env1/svc1"},
		"empty scope": {kind: "APIService", scope: "", name: "svc1", expected: "APIService//svc1"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ResourceCacheKey(tc.kind, tc.scope, tc.name))
		})
	}
}

func TestGetCachedResourcesByKind(t *testing.T) {
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	modTime2 := time.Date(2026, 3, 12, 11, 0, 0, 0, time.UTC)

	type testCase struct {
		setup      func(Manager)
		group      string
		kind       string
		expectLen  int
		expectKeys []string
		verify     func(t *testing.T, result map[string]time.Time)
	}

	tests := map[string]testCase{
		"APIService": {
			setup: func(cm Manager) {
				cm.AddAPIService(makeAPIServiceRI("env1", "svc1", "ext-id-1", modTime))
			},
			group:      "management",
			kind:       management.APIServiceGVK().Kind,
			expectLen:  1,
			expectKeys: []string{ResourceCacheKey(management.APIServiceGVK().Kind, "env1", "svc1")},
			verify: func(t *testing.T, result map[string]time.Time) {
				assert.Equal(t, modTime, result[ResourceCacheKey(management.APIServiceGVK().Kind, "env1", "svc1")])
			},
		},
		"APIServiceInstance": {
			setup: func(cm Manager) {
				cm.AddAPIServiceInstance(makeRI("management", management.APIServiceInstanceGVK().Kind, "Environment", "env1", "inst1", "id1", modTime))
			},
			group:      "management",
			kind:       management.APIServiceInstanceGVK().Kind,
			expectLen:  1,
			expectKeys: []string{ResourceCacheKey(management.APIServiceInstanceGVK().Kind, "env1", "inst1")},
		},
		"ManagedApplication": {
			setup: func(cm Manager) {
				cm.AddManagedApplication(makeRI("management", management.ManagedApplicationGVK().Kind, "Environment", "env1", "app1", "id1", modTime))
			},
			group:      "management",
			kind:       management.ManagedApplicationGVK().Kind,
			expectLen:  1,
			expectKeys: []string{ResourceCacheKey(management.ManagedApplicationGVK().Kind, "env1", "app1")},
		},
		"AccessRequest": {
			setup: func(cm Manager) {
				cm.AddAccessRequest(makeRI("management", management.AccessRequestGVK().Kind, "Environment", "env1", "ar1", "id1", modTime))
			},
			group:      "management",
			kind:       management.AccessRequestGVK().Kind,
			expectLen:  1,
			expectKeys: []string{ResourceCacheKey(management.AccessRequestGVK().Kind, "env1", "ar1")},
		},
		"multiple resources same kind": {
			setup: func(cm Manager) {
				cm.AddAPIService(makeAPIServiceRI("env1", "svc1", "ext-id-1", modTime))
				cm.AddAPIService(makeAPIServiceRI("env1", "svc2", "ext-id-2", modTime2))
			},
			group:     "management",
			kind:      management.APIServiceGVK().Kind,
			expectLen: 2,
			expectKeys: []string{
				ResourceCacheKey(management.APIServiceGVK().Kind, "env1", "svc1"),
				ResourceCacheKey(management.APIServiceGVK().Kind, "env1", "svc2"),
			},
		},
		"different scopes": {
			setup: func(cm Manager) {
				cm.AddAPIService(makeAPIServiceRI("env1", "svc1", "ext-id-1", modTime))
				cm.AddAPIService(makeAPIServiceRI("env2", "svc1", "ext-id-2", modTime))
			},
			group:     "management",
			kind:      management.APIServiceGVK().Kind,
			expectLen: 2,
			expectKeys: []string{
				ResourceCacheKey(management.APIServiceGVK().Kind, "env1", "svc1"),
				ResourceCacheKey(management.APIServiceGVK().Kind, "env2", "svc1"),
			},
		},
		"empty cache": {
			group:     "management",
			kind:      management.APIServiceGVK().Kind,
			expectLen: 0,
		},
		"unknown kind falls back to watch resource": {
			setup: func(cm Manager) {
				cm.AddWatchResource(makeRI("catalog", "SomeCustomKind", "Environment", "env1", "custom1", "id1", modTime))
			},
			group:      "catalog",
			kind:       "SomeCustomKind",
			expectLen:  1,
			expectKeys: []string{ResourceCacheKey("SomeCustomKind", "env1", "custom1")},
		},
		"watch resource filters by group and kind - group A": {
			setup: func(cm Manager) {
				cm.AddWatchResource(makeRI("groupA", "KindA", "", "", "res1", "id1", modTime))
				cm.AddWatchResource(makeRI("groupB", "KindB", "", "", "res2", "id2", modTime))
			},
			group:      "groupA",
			kind:       "KindA",
			expectLen:  1,
			expectKeys: []string{ResourceCacheKey("KindA", "", "res1")},
		},
		"watch resource filters by group and kind - group B": {
			setup: func(cm Manager) {
				cm.AddWatchResource(makeRI("groupA", "KindA", "", "", "res1", "id1", modTime))
				cm.AddWatchResource(makeRI("groupB", "KindB", "", "", "res2", "id2", modTime))
			},
			group:      "groupB",
			kind:       "KindB",
			expectLen:  1,
			expectKeys: []string{ResourceCacheKey("KindB", "", "res2")},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
			if tc.setup != nil {
				tc.setup(cm)
			}
			result := cm.GetCachedResourcesByKind(tc.group, tc.kind)
			assert.Len(t, result, tc.expectLen)
			for _, k := range tc.expectKeys {
				assert.Contains(t, result, k)
			}
			if tc.verify != nil {
				tc.verify(t, result)
			}
		})
	}
}
