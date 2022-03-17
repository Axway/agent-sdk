package cache

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

// add subscription
// get subscription by id
// get subscription by name
// delete subscription
func TestSubscriptionCache(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	cachedSubscription := m.GetSubscription("s1")
	assert.Nil(t, cachedSubscription)

	subscription1 := createRI("s1", "subscription-1")
	subscription2 := createRI("s2", "subscription-2")

	m.AddSubscription(subscription1)
	m.AddSubscription(subscription2)

	cachedApp := m.GetSubscription("s1")
	assert.Equal(t, subscription1, cachedApp)

	cachedApp = m.GetSubscriptionByName("subscription-2")
	assert.Equal(t, subscription2, cachedApp)

	err := m.DeleteSubscription("s1")
	assert.Nil(t, err)

	cachedApp = m.GetSubscription("s1")
	assert.Nil(t, cachedApp)

	cachedApp = m.GetSubscription("s2")
	assert.NotNil(t, cachedApp)

	err = m.DeleteSubscription("s1")
	assert.NotNil(t, err)
}
