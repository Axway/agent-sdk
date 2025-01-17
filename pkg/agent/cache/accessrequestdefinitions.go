package cache

import v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

// Access Request Definition cache management

// AddAccessRequestDefinition -  add/update AccessRequestDefinition resource in cache
func (c *cacheManager) AddAccessRequestDefinition(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.ardMap.SetWithSecondaryKey(resource.Metadata.ID, resource.Name, resource)
}

// GetAccessRequestDefinitionKeys - returns keys for AccessRequestDefinition cache
func (c *cacheManager) GetAccessRequestDefinitionKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.ardMap.GetKeys()
}

// GetAccessRequestDefinitionByName - returns resource from AccessRequestDefinition cache based on resource name
func (c *cacheManager) GetAccessRequestDefinitionByName(name string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.ardMap.GetBySecondaryKey(name)
	if item != nil {
		if ard, ok := item.(*v1.ResourceInstance); ok {
			ard.CreateHashes()
			return ard, nil
		}
	}
	return nil, err
}

// GetAccessRequestDefinitionByID - returns resource from AccessRequestDefinition cache based on resource id
func (c *cacheManager) GetAccessRequestDefinitionByID(id string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.ardMap.Get(id)
	if item != nil {
		if ard, ok := item.(*v1.ResourceInstance); ok {
			ard.CreateHashes()
			return ard, nil
		}
	}
	return nil, err
}

// DeleteAccessRequestDefinition - deletes the AccessRequestDefinition cache based on resource id
func (c *cacheManager) DeleteAccessRequestDefinition(id string) error {
	defer c.setCacheUpdated(true)

	return c.ardMap.Delete(id)
}
