package cache

import (
	"fmt"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// Environment cache management

// AddEnvironment - add/update Environment resource in cache
func (c *cacheManager) AddEnvironment(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	// get any existing categories with the same title, the oldest one should win the secondary key
	if cachedEnvironment := c.GetEnvironmentWithTitle(resource.Title); cachedEnvironment != nil {
		if time.Time(cachedEnvironment.Metadata.Audit.CreateTimestamp).Before(time.Time(resource.Metadata.Audit.CreateTimestamp)) {
			// add without the title to the cache
			c.environmentMap.Set(resource.Name, resource)
			return
		}
		// remove the current secondary key owner
		c.ApplyResourceReadLock()
		defer c.ReleaseResourceReadLock()
		c.environmentMap.DeleteSecondaryKey(formatTitle(resource.Title))
	}

	c.environmentMap.SetWithSecondaryKey(resource.Name, formatTitle(resource.Title), resource)
}

// GetEnvironmentWithTitle - returns resource from Environment cache based on title
func (c *cacheManager) GetEnvironmentWithTitle(title string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	environment, _ := c.environmentMap.GetBySecondaryKey(formatTitle(title))
	if environment != nil {
		ri, ok := environment.(*v1.ResourceInstance)
		if ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) formatTitle(title string) string {
	return fmt.Sprintf("title-%s", title)
}
