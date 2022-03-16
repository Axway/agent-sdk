package cache

import v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

// Credential request definition cache management

// AddCredentialRequestDefinition -  add/update CredentialRequestDefinition resource in cache
func (c *cacheManager) AddCredentialRequestDefinition(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.crdMap.Set(resource.Name, resource)
}

// GetCredentialRequestDefinitionByName - returns resource from CredentialRequestDefinition cache based on resource name
func (c *cacheManager) GetCredentialRequestDefinitionByName(name string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.crdMap.Get(name)
	if item != nil {
		if ard, ok := item.(*v1.ResourceInstance); ok {
			return ard, nil
		}
	}
	return nil, err
}

// DeleteCredentialRequestDefinitionByName - deletes the CredentialRequestDefinition cache based on resource name
func (c *cacheManager) DeleteCredentialRequestDefinitionByName(name string) error {
	defer c.setCacheUpdated(true)

	return c.crdMap.Delete(name)
}
