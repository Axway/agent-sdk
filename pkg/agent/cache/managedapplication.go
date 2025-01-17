package cache

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// ManagedApplication cache related methods
func (c *cacheManager) GetManagedApplicationCacheKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.managedApplicationMap.GetKeys()
}

func (c *cacheManager) AddManagedApplication(resource *v1.ResourceInstance) {
	if resource == nil {
		return
	}
	manApp := management.ManagedApplication{}
	err := manApp.FromInstance(resource)
	if err != nil {
		return
	}
	catalogAppRef := manApp.GetReferenceByGVK(catalog.ApplicationGVK())
	c.managedApplicationMap.SetWithSecondaryKey(resource.Metadata.ID, resource.Name, resource)
	c.managedApplicationMap.SetSecondaryKey(resource.Metadata.ID, catalogAppRef.ID)
}

func (c *cacheManager) GetManagedApplication(id string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	managedApp, _ := c.managedApplicationMap.Get(id)
	if managedApp != nil {
		ri, ok := managedApp.(*v1.ResourceInstance)
		if ok {
			ri.CreateHashes()
			return ri
		}
	}
	return nil
}

func (c *cacheManager) GetManagedApplicationByApplicationID(id string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	managedApp, _ := c.managedApplicationMap.GetBySecondaryKey(id)
	if managedApp != nil {
		ri, ok := managedApp.(*v1.ResourceInstance)
		if ok {
			return ri
		}
	}
	return nil
}

func (c *cacheManager) GetManagedApplicationByName(name string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	managedApp, _ := c.managedApplicationMap.GetBySecondaryKey(name)
	if managedApp != nil {
		ri, ok := managedApp.(*v1.ResourceInstance)
		if ok {
			ri.CreateHashes()
			return ri
		}
	}
	return nil
}

func (c *cacheManager) DeleteManagedApplication(id string) error {
	return c.managedApplicationMap.Delete(id)
}
