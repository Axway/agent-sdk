package cache

import (
	"fmt"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util"
)

// apiServiceToInstanceCount
type apiServiceToInstanceCount struct {
	Count         int
	ApiServiceKey string
}

// API service cache management

// AddAPIService - add/update APIService resource in cache
func (c *cacheManager) AddAPIService(svc *v1.ResourceInstance) error {
	apiID, err := util.GetAgentDetailsValue(svc, defs.AttrExternalAPIID)
	if err != nil {
		return fmt.Errorf("failed to get external API ID from APIService resource: %s", err)
	}
	if apiID != "" {
		defer c.setCacheUpdated(true)
		apiName, _ := util.GetAgentDetailsValue(svc, defs.AttrExternalAPIName)
		primaryKey, _ := util.GetAgentDetailsValue(svc, defs.AttrExternalAPIPrimaryKey)
		cachedRI, _ := c.GetAPIServiceInstanceByName(apiName)
		if primaryKey != "" {
			// Verify secondary key and validate if we need to remove it from the apiMap (cache)
			if _, err := c.apiMap.Get(apiID); err != nil {
				c.apiMap.Delete(apiID)
			}

			c.apiMap.SetWithSecondaryKey(primaryKey, apiID, svc)
			c.apiMap.SetSecondaryKey(primaryKey, apiName)
			c.apiMap.SetSecondaryKey(primaryKey, svc.Name)
		} else {
			c.apiMap.SetWithSecondaryKey(apiID, apiName, svc)
			c.apiMap.SetSecondaryKey(apiID, svc.Name)
		}

		if cachedRI == nil {
			c.countCachedInstancesForAPIService(apiID, primaryKey)
		}

		c.logger.
			WithField("api-name", apiName).
			WithField("api-id", apiID).
			Trace("added api to cache")
	}

	return nil
}

// GetAPIServiceCache - returns the APIService cache
func (c *cacheManager) GetAPIServiceCache() cache.Cache {
	return c.apiMap
}

// GetAPIServiceKeys - returns keys for APIService cache
func (c *cacheManager) GetAPIServiceKeys() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	return c.apiMap.GetKeys()
}

// GetAPIServiceWithAPIID - returns resource from APIService cache based on externalAPIID attribute
func (c *cacheManager) GetAPIServiceWithAPIID(apiID string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	api, _ := c.apiMap.Get(apiID)
	if api == nil {
		api, _ = c.apiMap.GetBySecondaryKey(apiID)
	}

	if api != nil {
		apiSvc, ok := api.(*v1.ResourceInstance)
		if ok {
			apiSvc.CreateHashes()
			return apiSvc
		}
	}
	return nil
}

// GetAPIServiceWithPrimaryKey - returns resource from APIService cache based on externalAPIPrimaryKey attribute
func (c *cacheManager) GetAPIServiceWithPrimaryKey(primaryKey string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	api, _ := c.apiMap.Get(primaryKey)
	if api != nil {
		apiSvc, ok := api.(*v1.ResourceInstance)
		if ok {
			apiSvc.CreateHashes()
			return apiSvc
		}
	}
	return nil
}

// GetAPIServiceWithName - returns resource from APIService cache based on externalAPIName attribute
func (c *cacheManager) GetAPIServiceWithName(apiName string) *v1.ResourceInstance {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	api, _ := c.apiMap.GetBySecondaryKey(apiName)
	if api != nil {
		apiSvc, ok := api.(*v1.ResourceInstance)
		if ok {
			apiSvc.CreateHashes()
			return apiSvc
		}
	}
	return nil
}

// GetTeamsIDsInAPIServices - returns the array of team IDs that have services
func (c *cacheManager) GetTeamsIDsInAPIServices() []string {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	teamNameMap := make(map[string]struct{})
	teamIDs := make([]string, 0)
	for _, key := range c.apiMap.GetKeys() {
		api, _ := c.apiMap.Get(key)
		if apiSvc, ok := api.(*v1.ResourceInstance); ok {
			if apiSvc.Owner != nil && apiSvc.Owner.Type == v1.TeamOwner {
				if _, found := teamNameMap[apiSvc.Owner.ID]; found {
					continue
				}
				teamNameMap[apiSvc.Owner.ID] = struct{}{}
				teamIDs = append(teamIDs, apiSvc.Owner.ID)
			}
		}
	}

	return teamIDs
}

// DeleteAPIService - remove APIService resource from cache based on externalAPIID or externalAPIPrimaryKey
func (c *cacheManager) DeleteAPIService(key string) error {
	defer c.setCacheUpdated(true)

	err := c.apiMap.Delete(key)
	if err != nil {
		err = c.apiMap.DeleteBySecondaryKey(key)
	}
	return err
}

// FormatKey -
func (c *cacheManager) FormatKey(svcName string) string {
	formatKey := fmt.Sprintf("count-%v", svcName)
	return formatKey
}

func (c *cacheManager) addToServiceInstanceCount(apiID, primaryKey string) error {
	svc := c.GetAPIServiceWithPrimaryKey(primaryKey)
	if svc == nil {
		svc = c.GetAPIServiceWithAPIID(apiID)
		if svc == nil {
			// can't increment a count for a service we can't find
			c.logger.
				WithField("primary-key", primaryKey).
				WithField("api-id", apiID).
				Debug("cannot increment a count for a service we cannot find")
			return nil
		}
	}
	key := c.FormatKey(svc.Name)

	svcCountI, _ := c.instanceCountMap.Get(key)
	svcCount := apiServiceToInstanceCount{}
	if svcCountI == nil {
		svcCount = apiServiceToInstanceCount{
			Count:         0,
			ApiServiceKey: svc.Metadata.ID,
		}
	} else {
		svcCount = svcCountI.(apiServiceToInstanceCount)
	}
	svcCount.Count++

	c.instanceCountMap.Set(key, svcCount)
	return nil
}

func (c *cacheManager) removeFromServiceInstanceCount(apiID, primaryKey string) error {
	svc := c.GetAPIServiceWithPrimaryKey(primaryKey)
	if svc == nil {
		svc = c.GetAPIServiceWithAPIID(apiID)
		if svc == nil {
			// can't decrement a count for a service we can't find
			return nil
		}
	}
	key := c.FormatKey(svc.Name)

	svcCountI, err := c.instanceCountMap.Get(key)
	if err != nil {
		return err
	}
	svcCount := apiServiceToInstanceCount{}
	if svcCountI != nil {
		svcCount = svcCountI.(apiServiceToInstanceCount)
		svcCount.Count--
	}

	c.instanceCountMap.Set(key, svcCount)
	return nil
}

func (c *cacheManager) deleteAllServiceInstanceCounts() {
	c.instanceCountMap.Flush()
}

func (c *cacheManager) GetAPIServiceInstanceCount(svcName string) int {
	svc := c.GetAPIServiceWithName(svcName)

	// get apiid and primary key
	apiID, _ := util.GetAgentDetailsValue(svc, defs.AttrExternalAPIID)
	primaryKey, _ := util.GetAgentDetailsValue(svc, defs.AttrExternalAPIPrimaryKey)

	count := 0
	for _, k := range c.instanceMap.GetKeys() {
		item, _ := c.instanceMap.Get(k)
		inst, ok := item.(*v1.ResourceInstance)
		if !ok {
			continue
		}
		instAPIID, _ := util.GetAgentDetailsValue(inst, defs.AttrExternalAPIID)
		instPrimary, _ := util.GetAgentDetailsValue(inst, defs.AttrExternalAPIPrimaryKey)
		if apiID == instAPIID || primaryKey == instPrimary {
			count++
		}
	}

	return count
}
