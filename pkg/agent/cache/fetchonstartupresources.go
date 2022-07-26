package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const key_td = "fetch-on-startup"

func (c *cacheManager) AddFetchOnStartupResources(resources []*v1.ResourceInstance) {

	var allResources []*v1.ResourceInstance
	if !c.hasFetchOnStartupKey() {
		allResources = resources
	} else {
		cached, err := c.fetchOnStartup.Get(key_td)
		if err != nil {
			log.Errorf("Error fetching key \"%s\" from cache: %v", key_td, err)
			return
		}
		allResources = append(cached.([]*v1.ResourceInstance), resources...)
	}

	err := c.fetchOnStartup.Set(key_td, allResources)
	if err != nil {
		log.Errorf("Error fetching setting \"%s\" from cache: %v", key_td, err)
	}
}

func (c *cacheManager) GetAllFetchOnStartupResources() []*v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	if c.hasFetchOnStartupKey() {
		resources, err := c.fetchOnStartup.Get(key_td)
		if err != nil {
			log.Errorf("Error fetching key \"%s\" from cache: %v", key_td, err)
			return make([]*v1.ResourceInstance, 0)
		}
		return resources.([]*v1.ResourceInstance)
	}
	return make([]*v1.ResourceInstance, 0)
}

func (c *cacheManager) DeleteAllFetchOnStartupResources() error {
	if c.hasFetchOnStartupKey() {
		return c.fetchOnStartup.Delete(key_td)
	}
	return nil
}

func (c *cacheManager) hasFetchOnStartupKey() bool {
	keys := c.fetchOnStartup.GetKeys()
	return len(keys) == 1 && keys[0] == key_td
}
