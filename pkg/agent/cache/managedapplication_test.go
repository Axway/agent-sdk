package cache

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

// add managed application
// get managed application by id
// get managed application by name
// delete managed application
func TestManagedApplicationCache(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	assert.Equal(t, []string{}, m.GetManagedApplicationCacheKeys())

	app1 := createRI("m1", "app-1")
	app2 := createRI("m2", "app-2")

	m.AddManagedApplication(app1)
	assert.ElementsMatch(t, []string{"m1"}, m.GetManagedApplicationCacheKeys())
	m.AddManagedApplication(app2)
	assert.ElementsMatch(t, []string{"m1", "m2"}, m.GetManagedApplicationCacheKeys())

	cachedApp := m.GetManagedApplication("m1")
	assert.Equal(t, app1, cachedApp)

	cachedApp = m.GetManagedApplicationByName("app-2")
	assert.Equal(t, app2, cachedApp)

	err := m.DeleteManagedApplication("m1")
	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"m2"}, m.GetManagedApplicationCacheKeys())

	cachedApp = m.GetManagedApplication("m1")
	assert.Nil(t, cachedApp)

	err = m.DeleteManagedApplication("m1")
	assert.NotNil(t, err)
}
