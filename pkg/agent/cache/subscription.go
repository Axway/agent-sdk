package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// Subscription cache related methods - Temporary
func (c *cacheManager) AddSubscription(resource *v1.ResourceInstance) {
	if resource == nil {
		return
	}
	c.subscriptionMap.SetWithSecondaryKey(resource.Metadata.ID, resource.Name, resource)
}

func (c *cacheManager) GetSubscriptionByName(subscriptionName string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	subscription, _ := c.subscriptionMap.GetBySecondaryKey(subscriptionName)
	if subscription != nil {
		ri, ok := subscription.(*v1.ResourceInstance)
		if ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) GetSubscription(id string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	subscription, _ := c.subscriptionMap.Get(id)
	if subscription != nil {
		ri, ok := subscription.(*v1.ResourceInstance)
		if ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) DeleteSubscription(id string) error {
	return c.subscriptionMap.Delete(id)
}
