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
	key := ResourceCacheKey("APIService", "env1", "svc1")
	assert.Equal(t, "APIService/env1/svc1", key)

	key = ResourceCacheKey("APIService", "", "svc1")
	assert.Equal(t, "APIService//svc1", key)
}

func TestGetCachedResourcesByKind_APIService(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	ri := makeAPIServiceRI("env1", "svc1", "ext-id-1", modTime)
	err := cm.AddAPIService(ri)
	assert.Nil(t, err)

	result := cm.GetCachedResourcesByKind("management", management.APIServiceGVK().Kind)
	expectedKey := ResourceCacheKey(management.APIServiceGVK().Kind, "env1", "svc1")
	assert.Len(t, result, 1)
	assert.Contains(t, result, expectedKey)
	assert.Equal(t, modTime, result[expectedKey])
}

func TestGetCachedResourcesByKind_APIServiceInstance(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	ri := makeRI("management", management.APIServiceInstanceGVK().Kind, "Environment", "env1", "inst1", "id1", modTime)
	cm.AddAPIServiceInstance(ri)

	result := cm.GetCachedResourcesByKind("management", management.APIServiceInstanceGVK().Kind)
	expectedKey := ResourceCacheKey(management.APIServiceInstanceGVK().Kind, "env1", "inst1")
	assert.Len(t, result, 1)
	assert.Contains(t, result, expectedKey)
}

func TestGetCachedResourcesByKind_ManagedApplication(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	ri := makeRI("management", management.ManagedApplicationGVK().Kind, "Environment", "env1", "app1", "id1", modTime)
	cm.AddManagedApplication(ri)

	result := cm.GetCachedResourcesByKind("management", management.ManagedApplicationGVK().Kind)
	expectedKey := ResourceCacheKey(management.ManagedApplicationGVK().Kind, "env1", "app1")
	assert.Len(t, result, 1)
	assert.Contains(t, result, expectedKey)
}

func TestGetCachedResourcesByKind_AccessRequest(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	ri := makeRI("management", management.AccessRequestGVK().Kind, "Environment", "env1", "ar1", "id1", modTime)
	cm.AddAccessRequest(ri)

	result := cm.GetCachedResourcesByKind("management", management.AccessRequestGVK().Kind)
	expectedKey := ResourceCacheKey(management.AccessRequestGVK().Kind, "env1", "ar1")
	assert.Len(t, result, 1)
	assert.Contains(t, result, expectedKey)
}

func TestGetCachedResourcesByKind_MultipleResourcesSameKind(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	modTime1 := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	modTime2 := time.Date(2026, 3, 12, 11, 0, 0, 0, time.UTC)

	svcKind := management.APIServiceGVK().Kind
	ri1 := makeAPIServiceRI("env1", "svc1", "ext-id-1", modTime1)
	ri2 := makeAPIServiceRI("env1", "svc2", "ext-id-2", modTime2)
	cm.AddAPIService(ri1)
	cm.AddAPIService(ri2)

	result := cm.GetCachedResourcesByKind("management", svcKind)
	assert.Len(t, result, 2)
	assert.Contains(t, result, ResourceCacheKey(svcKind, "env1", "svc1"))
	assert.Contains(t, result, ResourceCacheKey(svcKind, "env1", "svc2"))
}

func TestGetCachedResourcesByKind_DifferentScopes(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	svcKind := management.APIServiceGVK().Kind
	ri1 := makeAPIServiceRI("env1", "svc1", "ext-id-1", modTime)
	ri2 := makeAPIServiceRI("env2", "svc1", "ext-id-2", modTime)
	cm.AddAPIService(ri1)
	cm.AddAPIService(ri2)

	result := cm.GetCachedResourcesByKind("management", svcKind)
	key1 := ResourceCacheKey(svcKind, "env1", "svc1")
	key2 := ResourceCacheKey(svcKind, "env2", "svc1")

	assert.Len(t, result, 2)
	assert.Contains(t, result, key1)
	assert.Contains(t, result, key2)
}

func TestGetCachedResourcesByKind_EmptyCache(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)

	result := cm.GetCachedResourcesByKind("management", management.APIServiceGVK().Kind)
	assert.Empty(t, result)
}

func TestGetCachedResourcesByKind_UnknownKindFallsBackToWatchResource(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	ri := makeRI("catalog", "SomeCustomKind", "Environment", "env1", "custom1", "id1", modTime)
	cm.AddWatchResource(ri)

	result := cm.GetCachedResourcesByKind("catalog", "SomeCustomKind")
	expectedKey := ResourceCacheKey("SomeCustomKind", "env1", "custom1")
	assert.Len(t, result, 1)
	assert.Contains(t, result, expectedKey)
}

func TestGetCachedResourcesByKind_WatchResourceFiltersGroupAndKind(t *testing.T) {
	cm := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	modTime := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)

	ri1 := makeRI("groupA", "KindA", "", "", "res1", "id1", modTime)
	ri2 := makeRI("groupB", "KindB", "", "", "res2", "id2", modTime)
	cm.AddWatchResource(ri1)
	cm.AddWatchResource(ri2)

	result := cm.GetCachedResourcesByKind("groupA", "KindA")
	assert.Len(t, result, 1)
	assert.Contains(t, result, ResourceCacheKey("KindA", "", "res1"))

	result = cm.GetCachedResourcesByKind("groupB", "KindB")
	assert.Len(t, result, 1)
	assert.Contains(t, result, ResourceCacheKey("KindB", "", "res2"))
}
