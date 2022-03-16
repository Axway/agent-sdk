package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// API service instance management

// AddAPIServiceInstance -  add/update APIServiceInstance resource in cache
func (c *cacheManager) AddAPIServiceInstance(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.instanceMap.Set(resource.Metadata.ID, resource)
}

// GetAPIServiceInstanceKeys - returns keys for APIServiceInstance cache
func (c *cacheManager) GetAPIServiceInstanceKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.instanceMap.GetKeys()
}

// GetAPIServiceInstanceByID - returns resource from APIServiceInstance cache based on instance ID
func (c *cacheManager) GetAPIServiceInstanceByID(id string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.instanceMap.Get(id)
	if item != nil {
		instance, ok := item.(*v1.ResourceInstance)
		if ok {
			return instance, nil
		}
	}
	return nil, err
}

// DeleteAPIServiceInstance - remove APIServiceInstance resource from cache based on instance ID
func (c *cacheManager) DeleteAPIServiceInstance(id string) error {
	defer c.setCacheUpdated(true)

	return c.instanceMap.Delete(id)
}

// DeleteAllAPIServiceInstance - remove all APIServiceInstance resource from cache
func (c *cacheManager) DeleteAllAPIServiceInstance() {
	defer c.setCacheUpdated(true)

	c.instanceMap.Flush()
}
