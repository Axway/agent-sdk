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
	count         int
	apiServiceKey string
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
		if primaryKey != "" {
			// Verify secondary key and validate if we need to remove it from the apiMap (cache)
			if _, err := c.apiMap.Get(apiID); err != nil {
				c.apiMap.Delete(apiID)
			}

			c.apiMap.SetWithSecondaryKey(primaryKey, apiID, svc)
			c.apiMap.SetSecondaryKey(primaryKey, apiName)
		} else {
			c.apiMap.SetWithSecondaryKey(apiID, apiName, svc)
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

func (c *cacheManager) addToServiceInstanceCount(primaryKey string) error {
	svc := c.GetAPIServiceWithPrimaryKey(primaryKey)
	key := fmt.Sprintf("count-%v", svc.Name)

	svcCountI, _ := c.instanceCountMap.Get(key)
	svcCount := apiServiceToInstanceCount{}
	if svcCountI == nil {
		svcCount = apiServiceToInstanceCount{
			count:         0,
			apiServiceKey: svc.Metadata.ID,
		}
	} else {
		svcCount = svcCountI.(apiServiceToInstanceCount)
	}
	svcCount.count++

	c.apiMap.Set(key, svcCount)
	return nil
}

func (c *cacheManager) removeFromServiceInstanceCount(primaryKey string) error {
	svc := c.GetAPIServiceWithPrimaryKey(primaryKey)
	key := fmt.Sprintf("count-%v", svc.Name)

	svcCountI, err := c.instanceCountMap.Get(key)
	if err != nil {
		return err
	}
	svcCount := apiServiceToInstanceCount{}
	if svcCountI != nil {
		svcCount = svcCountI.(apiServiceToInstanceCount)
		svcCount.count--
	}

	return nil
}

func (c *cacheManager) deleteAllServiceInstanceCounts() {
	c.instanceCountMap.Flush()
}

func (c *cacheManager) GetAPIServiceInstanceCount(svcName string) int {
	key := fmt.Sprintf("count-%v", svcName)

	svcCountI, err := c.apiMap.Get(key)
	if err != nil {
		return 0
	}
	svcCount := apiServiceToInstanceCount{}
	if svcCountI != nil {
		svcCount = svcCountI.(apiServiceToInstanceCount)
		return svcCount.count
	}

	return 0
}
