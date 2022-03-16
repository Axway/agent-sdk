package cache

import v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

// Credential request definition cache management

// AddCredentialRequestDefinition -  add/update CredentialRequestDefinition resource in cache
func (c *cacheManager) AddCredentialRequestDefinition(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.crdMap.Set(resource.Metadata.ID, resource)
}

// GetCredentialRequestDefinitionByName - returns resource from CredentialRequestDefinition cache based on resource name
func (c *cacheManager) GetCredentialRequestDefinitionByName(name string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, _ := c.crdMap.Get(name)
	if item != nil {
		ard, ok := item.(*v1.ResourceInstance)
		if ok {
			return ard
		}
	}
	return nil
}

// DeleteCredentialRequestDefinitionByName - deletes the CredentialRequestDefinition cache based on resource name
func (c *cacheManager) DeleteCredentialRequestDefinitionByName(name string) error {
	defer c.setCacheUpdated(true)

	return c.crdMap.Delete(name)
}
