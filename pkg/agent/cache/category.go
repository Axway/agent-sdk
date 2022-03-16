package cache

import (
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
)

// Category cache management

// AddCategory - add/update Category resource in cache
func (c *cacheManager) AddCategory(resource *v1.ResourceInstance) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()
	defer c.setCacheUpdated(true)

	c.categoryMap.SetWithSecondaryKey(resource.Name, fmt.Sprintf("title-%s", resource.Title), resource)
}

// GetCategoryCache - returns the Category cache
func (c *cacheManager) GetCategoryCache() cache.Cache {
	return c.categoryMap
}

// GetCategoryKeys - returns keys for Category cache
func (c *cacheManager) GetCategoryKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.categoryMap.GetKeys()
}

// GetCategory - returns resource from Category cache based on name
func (c *cacheManager) GetCategory(name string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	category, _ := c.categoryMap.Get(name)
	if category != nil {
		ri, ok := category.(*v1.ResourceInstance)
		if ok {
			return ri
		}
	}
	return nil
}

// GetCategoryWithTitle - returns resource from Category cache based on title
func (c *cacheManager) GetCategoryWithTitle(title string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	category, _ := c.categoryMap.GetBySecondaryKey(fmt.Sprintf("title-%s", title))
	if category != nil {
		ri, ok := category.(*v1.ResourceInstance)
		if ok {
			return ri
		}
	}
	return nil
}

// DeleteCategory - remove Category resource from cache based on name
func (c *cacheManager) DeleteCategory(name string) error {
	defer c.setCacheUpdated(true)

	return c.categoryMap.Delete(name)
}
