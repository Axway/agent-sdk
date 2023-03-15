package cache

import (
	"fmt"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
)

// Category cache management

// AddCategory - add/update Category resource in cache
func (c *cacheManager) AddCategory(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	// get any existing categories with the same title, the oldest one should win the secondary key
	if cachedCategory := c.GetCategoryWithTitle(resource.Title); cachedCategory != nil {
		if time.Time(cachedCategory.Metadata.Audit.CreateTimestamp).Before(time.Time(resource.Metadata.Audit.CreateTimestamp)) {
			// add without the title to the cache
			c.categoryMap.Set(resource.Name, resource)
			return
		}
		// remove the current secondary key owner
		c.ApplyResourceReadLock()
		defer c.ReleaseResourceReadLock()
		c.categoryMap.DeleteSecondaryKey(formatTitle(resource.Title))
	}

	c.categoryMap.SetWithSecondaryKey(resource.Name, formatTitle(resource.Title), resource)
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

	category, _ := c.categoryMap.GetBySecondaryKey(formatTitle(title))
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

func formatTitle(title string) string {
	return fmt.Sprintf("title-%s", title)
}
