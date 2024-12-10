package cache

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func createWatchResource(group, kind, id, name string) *v1.ResourceInstance {
	ri := &v1.ResourceInstance{
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
	ri.CreateHashes()
	return ri
}

// add watch resource
// get watch resource by key
// get watch resource by id
// get watch resource by name
// delete watch resource
func TestWatchResourceCache(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	cachedRes := m.GetWatchResourceByKey("group:kind:test-id-1")
	assert.Nil(t, cachedRes)

	instance1 := createWatchResource("group-1", "kind-1", "test-id-1", "test-name-1")
	instance2 := createWatchResource("group-2", "kind-2", "test-id-2", "test-name-2")
	m.AddWatchResource(instance1)
	m.AddWatchResource(instance2)

	keys := m.GetWatchResourceCacheKeys("group-1", "kind-1")
	assert.Equal(t, 1, len(keys))

	keys = m.GetWatchResourceCacheKeys("group-2", "kind-2")
	assert.Equal(t, 1, len(keys))

	cachedRes = m.GetWatchResourceByKey("group-1:kind-1:test-id-1")
	assert.Equal(t, instance1, cachedRes)

	cachedRes = m.GetWatchResourceByID("dummy-group", "dummy=kind", "test-id-1")
	assert.Nil(t, cachedRes)

	cachedRes = m.GetWatchResourceByID("group-1", "kind-1", "test-id-1")
	assert.Equal(t, instance1, cachedRes)

	cachedRes = m.GetWatchResourceByName("group-2", "kind-2", "test-name-2")
	assert.Equal(t, instance2, cachedRes)

	err := m.DeleteWatchResource("group-2", "kind-2", "test-id-2")
	assert.Nil(t, err)

	cachedRes = m.GetWatchResourceByID("group-2", "kind-2", "test-id-2")
	assert.Nil(t, cachedRes)

	cachedRes = m.GetWatchResourceByID("group-1", "kind-1", "test-id-1")
	assert.NotNil(t, cachedRes)

	err = m.DeleteWatchResource("group-1", "kind-1", "test-id-1")
	assert.Nil(t, err)

	keys = m.GetWatchResourceCacheKeys("group-1", "kind-1")
	assert.Equal(t, 0, len(keys))

	keys = m.GetWatchResourceCacheKeys("group-2", "kind-2")
	assert.Equal(t, 0, len(keys))
}
