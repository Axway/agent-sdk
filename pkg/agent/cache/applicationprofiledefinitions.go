package cache

import v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

// Access Request Definition cache management

// AddApplicationProfileDefinition -  add/update ApplicationProfileDefinition resource in cache
func (c *cacheManager) AddApplicationProfileDefinition(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.apdMap.SetWithSecondaryKey(resource.Metadata.ID, resource.Name, resource)
}

// GetApplicationProfileDefinitionKeys - returns keys for ApplicationProfileDefinition cache
func (c *cacheManager) GetApplicationProfileDefinitionKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.apdMap.GetKeys()
}

// GetApplicationProfileDefinitionByName - returns resource from ApplicationProfileDefinition cache based on resource name
func (c *cacheManager) GetApplicationProfileDefinitionByName(name string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.apdMap.GetBySecondaryKey(name)
	if item != nil {
		if ard, ok := item.(*v1.ResourceInstance); ok {
			ard.CreateHashes()
			return ard, nil
		}
	}
	return nil, err
}

// GetApplicationProfileDefinitionByID - returns resource from ApplicationProfileDefinition cache based on resource id
func (c *cacheManager) GetApplicationProfileDefinitionByID(id string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.apdMap.Get(id)
	if item != nil {
		if ard, ok := item.(*v1.ResourceInstance); ok {
			ard.CreateHashes()
			return ard, nil
		}
	}
	return nil, err
}

// DeleteApplicationProfileDefinition - deletes the ApplicationProfileDefinition cache based on resource id
func (c *cacheManager) DeleteApplicationProfileDefinition(id string) error {
	defer c.setCacheUpdated(true)

	return c.apdMap.Delete(id)
}
