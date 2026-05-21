package cache

import v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

func (c *cacheManager) AddPublishedProduct(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.publishedProductMap.SetWithSecondaryKey(resource.Metadata.ID, resource.Name, resource)
}

func (c *cacheManager) GetPublishedProductByID(id string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.publishedProductMap.Get(id)
	if item != nil {
		if pp, ok := item.(*v1.ResourceInstance); ok {
			pp.CreateHashes()
			return pp, nil
		}
	}
	return nil, err
}

func (c *cacheManager) GetPublishedProductByName(name string) (*v1.ResourceInstance, error) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	item, err := c.publishedProductMap.GetBySecondaryKey(name)
	if item != nil {
		if pp, ok := item.(*v1.ResourceInstance); ok {
			pp.CreateHashes()
			return pp, nil
		}
	}
	return nil, err
}

func (c *cacheManager) DeletePublishedProduct(id string) error {
	defer c.setCacheUpdated(true)

	return c.publishedProductMap.Delete(id)
}
