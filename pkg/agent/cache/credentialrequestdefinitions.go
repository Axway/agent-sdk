package cache

import v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

// Credential request definition cache management

// AddCredentialRequestDefinition -  add/update CredentialRequestDefinition resource in cache
func (c *cacheManager) AddCredentialRequestDefinition(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.crdMap.SetWithSecondaryKey(resource.Metadata.ID, resource.Name, resource)
}

// GetCredentialRequestDefinitionKeys - returns keys for CredentialRequestDefinition cache
func (c *cacheManager) GetCredentialRequestDefinitionKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.crdMap.GetKeys()
}

// GetCredentialRequestDefinitionByName - returns resource from CredentialRequestDefinition cache based on resource name
func (c *cacheManager) GetCredentialRequestDefinitionByName(name string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.crdMap.GetBySecondaryKey(name)
	if item != nil {
		if ard, ok := item.(*v1.ResourceInstance); ok {
			return ard, nil
		}
	}
	return nil, err
}

// GetCredentialRequestDefinitionByID - returns resource from CredentialRequestDefinition cache based on resource id
func (c *cacheManager) GetCredentialRequestDefinitionByID(id string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.crdMap.Get(id)
	if item != nil {
		if ard, ok := item.(*v1.ResourceInstance); ok {
			return ard, nil
		}
	}
	return nil, err
}

// DeleteCredentialRequestDefinition - deletes the CredentialRequestDefinition cache based on resource id
func (c *cacheManager) DeleteCredentialRequestDefinition(id string) error {
	defer c.setCacheUpdated(true)

	return c.crdMap.Delete(id)
}
