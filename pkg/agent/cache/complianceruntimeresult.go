package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

func (c *cacheManager) AddComplianceRuntimeResult(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.crrMap.SetWithSecondaryKey(resource.Metadata.ID, resource.Name, resource)
}

func (c *cacheManager) GetComplianceRuntimeResultKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.crrMap.GetKeys()
}

func (c *cacheManager) GetComplianceRuntimeResultByID(id string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.crrMap.Get(id)
	if item != nil {
		instance, ok := item.(*v1.ResourceInstance)
		if ok {
			instance.CreateHashes()
			return instance, nil
		}
	}
	return nil, err
}

func (c *cacheManager) GetComplianceRuntimeResultByName(name string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.crrMap.GetBySecondaryKey(name)
	if item != nil {
		if crr, ok := item.(*v1.ResourceInstance); ok {
			crr.CreateHashes()
			return crr, nil
		}
	}
	return nil, err
}

// DeleteAPIServiceInstance - remove APIServiceInstance resource from cache based on instance ID
func (c *cacheManager) DeleteComplianceRuntimeResult(id string) error {
	defer c.setCacheUpdated(true)

	return c.crrMap.Delete(id)
}
