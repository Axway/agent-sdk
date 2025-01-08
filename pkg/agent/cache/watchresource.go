package cache

import (
	"fmt"
	"strings"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// Custom watch resource cache related methods
func (c *cacheManager) GetWatchResourceCacheKeys(group, kind string) []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()
	groupKindKeyPrefix := fmt.Sprintf("%s:%s", group, kind)

	keys := make([]string, 0)
	for _, key := range c.watchResourceMap.GetKeys() {
		if strings.HasPrefix(key, groupKindKeyPrefix) {
			keys = append(keys, key)
		}
	}
	return keys
}

func (c *cacheManager) getWatchResourceKey(group, kind, key string) string {
	return fmt.Sprintf("%s:%s:%s", group, kind, key)
}

func (c *cacheManager) AddWatchResource(ri *v1.ResourceInstance) {
	if ri == nil {
		return
	}
	group := ri.Group
	kind := ri.Kind

	c.watchResourceMap.SetWithSecondaryKey(c.getWatchResourceKey(group, kind, ri.Metadata.ID), c.getWatchResourceKey(group, kind, ri.Name), ri)
}

func (c *cacheManager) GetWatchResourceByKey(key string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	resource, _ := c.watchResourceMap.Get(key)
	if resource != nil {
		if ri, ok := resource.(*v1.ResourceInstance); ok {
			ri.CreateHashes()
			return ri
		}
	}
	return nil
}

func (c *cacheManager) GetWatchResourceByID(group, kind, id string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	resource, _ := c.watchResourceMap.Get(c.getWatchResourceKey(group, kind, id))
	if resource != nil {
		if ri, ok := resource.(*v1.ResourceInstance); ok {
			ri.CreateHashes()
			return ri
		}
	}
	return nil
}

func (c *cacheManager) GetWatchResourceByName(group, kind, name string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	resource, _ := c.watchResourceMap.GetBySecondaryKey(c.getWatchResourceKey(group, kind, name))
	if resource != nil {
		if ri, ok := resource.(*v1.ResourceInstance); ok {
			ri.CreateHashes()
			return ri
		}
	}
	return nil
}

func (c *cacheManager) DeleteWatchResource(group, kind, id string) error {
	return c.watchResourceMap.Delete(c.getWatchResourceKey(group, kind, id))
}
