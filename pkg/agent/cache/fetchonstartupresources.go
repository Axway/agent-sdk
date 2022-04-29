package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const key = "fetch-on-startup"

func (c *cacheManager) AddFetchOnStartupResources(resources []*v1.ResourceInstance) {

	var allResources []*v1.ResourceInstance
	if !c.hasKey() {
		allResources = resources
	} else {
		cached, err := c.fetchOnStartup.Get(key)
		if err != nil {
			log.Errorf("Error fetching key \"%s\" from cache: %v", key, err)
			return
		}
		allResources = append(cached.([]*v1.ResourceInstance), resources...)
	}

	err := c.fetchOnStartup.Set(key, allResources)
	if err != nil {
		log.Errorf("Error fetching setting \"%s\" from cache: %v", key, err)
	}
}

func (c *cacheManager) GetAllFetchOnStartupResources() []*v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	if c.hasKey() {
		resources, err := c.fetchOnStartup.Get(key)
		if err != nil {
			log.Errorf("Error fetching key \"%s\" from cache: %v", key, err)
			return make([]*v1.ResourceInstance, 0)
		}
		return resources.([]*v1.ResourceInstance)
	}
	return make([]*v1.ResourceInstance, 0)
}

func (c *cacheManager) DeleteAllFetchOnStartupResources() error {
	if c.hasKey() {
		return c.fetchOnStartup.Delete(key)
	}
	return nil
}

func (c *cacheManager) hasKey() bool {
	keys := c.fetchOnStartup.GetKeys()
	return len(keys) == 1 && keys[0] == key
}
