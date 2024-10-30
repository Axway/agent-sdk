package cache

import (
	"sync"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
)

type cacheMigrate func(key string) error

// migratePersistentCache is the top level migrator for all cache migrations
func (c *cacheManager) migratePersistentCache(key string) error {
	c.logger.Trace("checking if the persisted cache needs migrations")

	wg := sync.WaitGroup{}
	errs := make([]error, len(c.migrators))
	for i, m := range c.migrators {
		wg.Add(1)
		go func(index int, migFunc cacheMigrate) {
			defer wg.Done()
			errs[index] = migFunc(key)
		}(i, m)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *cacheManager) migrateAccessRequest(key string) error {
	if key != accReqKey {
		return nil
	}

	// migrate to set the foreign keys
	if len(c.accessRequestMap.GetKeys()) > 0 && len(c.accessRequestMap.GetForeignKeys()) == 0 {
		c.logger.Trace("migrating access requests to set foreign key of managed application name")

		wg := sync.WaitGroup{}
		errs := make([]error, len(c.accessRequestMap.GetKeys()))
		for i, k := range c.accessRequestMap.GetKeys() {
			wg.Add(1)
			go func(index int, key string) {
				defer wg.Done()
				inst, _ := c.accessRequestMap.Get(key)
				if inst != nil {
					if ri, ok := inst.(*v1.ResourceInstance); ok {
						accessRequest := management.NewAccessRequest("", "")
						errs[index] = accessRequest.FromInstance(ri)
						c.accessRequestMap.SetForeignKey(key, formatAppForeignKey(accessRequest.Spec.ManagedApplication))
					}
				}
			}(i, k)
		}
		wg.Wait()

		for _, err := range errs {
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *cacheManager) migrateInstanceCount(key string) error {
	if key != instanceCountKey {
		return nil
	}

	// run if there are api services, api service instances, and no instance counts
	if len(c.apiMap.GetKeys()) != 0 && len(c.instanceMap.GetKeys()) != 0 && len(c.instanceCountMap.GetKeys()) == 0 {
		c.logger.Trace("migrating instance counts for api service instances")
		wg := sync.WaitGroup{}
		errs := make([]error, len(c.accessRequestMap.GetKeys()))

		for i, k := range c.instanceMap.GetKeys() {
			wg.Add(1)
			go func(index int, key string) {
				defer wg.Done()
				inst, _ := c.instanceMap.Get(key)
				if inst != nil {
					if ri, ok := inst.(*v1.ResourceInstance); ok {
						apiID, _ := util.GetAgentDetailsValue(ri, defs.AttrExternalAPIID)
						primaryKey, _ := util.GetAgentDetailsValue(ri, defs.AttrExternalAPIPrimaryKey)
						c.addToServiceInstanceCount(apiID, primaryKey)
					}
				}
			}(i, k)
		}
		wg.Wait()

		for _, err := range errs {
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *cacheManager) migrateManagedApplications(cacheKey string) error {
	if cacheKey != managedAppKey {
		return nil
	}

	for _, key := range c.managedApplicationMap.GetKeys() {
		cachedManagedApp, _ := c.managedApplicationMap.Get(key)
		if cachedManagedApp == nil {
			continue
		}
		ri, ok := cachedManagedApp.(*v1.ResourceInstance)
		if !ok {
			continue
		}
		manApp := management.ManagedApplication{}
		err := manApp.FromInstance(ri)
		if err != nil {
			continue
		}
		catalogAppRef := manApp.GetReferenceByGVK(catalog.ApplicationGVK())
		c.managedApplicationMap.SetSecondaryKey(key, catalogAppRef.ID)
	}
	return nil
}
