package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// API service instance management

// AddAPIServiceInstance -  add/update APIServiceInstance resource in cache
func (c *cacheManager) AddAPIServiceInstance(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.instanceMap.SetWithSecondaryKey(resource.Metadata.ID, resource.Name, resource)
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

// GetAPIServiceInstanceByName - returns resource from APIServiceInstance cache based on instance name
func (c *cacheManager) GetAPIServiceInstanceByName(name string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.instanceMap.GetBySecondaryKey(name)
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

func (c *cacheManager) ListAPIServiceInstances() []*v1.ResourceInstance {
	keys := c.GetAPIServiceInstanceKeys()
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	var instances []*v1.ResourceInstance

	for _, key := range keys {
		item, _ := c.instanceMap.Get(key)
		if item != nil {
			instance, ok := item.(*v1.ResourceInstance)
			if ok {
				instances = append(instances, instance)
			}
		}
	}

	return instances
}
