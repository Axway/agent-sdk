package cache

import v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

// Access Request Definition cache management

// AddAccessRequestDefinition -  add/update AccessRequestDefinition resource in cache
func (c *cacheManager) AddAccessRequestDefinition(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.ardMap.Set(resource.Metadata.ID, resource)
}

// GetAccessRequestDefinitionByName - returns resource from AccessRequestDefinition cache based on resource name
func (c *cacheManager) GetAccessRequestDefinitionByName(name string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, _ := c.ardMap.Get(name)
	if item != nil {
		ard, ok := item.(*v1.ResourceInstance)
		if ok {
			return ard
		}
	}
	return nil
}

// DeleteAccessRequestDefinitionByName - deletes the AccessRequestDefinition cache based on resource name
func (c *cacheManager) DeleteAccessRequestDefinitionByName(name string) error {
	defer c.setCacheUpdated(true)

	return c.ardMap.Delete(name)
}
