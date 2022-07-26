package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const key = "runtimeconfig"

func (c *cacheManager) UpdateRuntimeconfigResource(resource *v1.ResourceInstance) {

	if resource == nil {
		return
	}

	arc := &mv1.AmplifyRuntimeConfig{}
	if arc.FromInstance(resource) != nil {
		return
	}

	err := c.runtimeConfig.Set(key, resource)

	if err != nil {
		log.Errorf("Error fetching setting \"%s\" from cache: %v", key, err)
	}
}

func (c *cacheManager) GetRuntimeconfigResource() *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	rtc, err := c.runtimeConfig.Get(key)
	if err != nil {
		log.Errorf("Error fetching key \"%s\" from cache: %v", key, err)
		return &v1.ResourceInstance{}
	}
	return rtc.(*v1.ResourceInstance)

}
