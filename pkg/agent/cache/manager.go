package cache

import (
	"encoding/json"
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	defaultCacheStoragePath = "./data/cache"
)

// Manager - interface to manage agent resource
type Manager interface {

	// Cache management related methods
	HasLoadedPersistedCache() bool
	SaveCache()

	//API Service cache related methods
	AddAPIService(resource *v1.ResourceInstance) string
	GetAPIServiceCache() cache.Cache
	GetAPIServiceKeys() []string
	GetAPIServiceWithAPIID(externalAPIID string) *v1.ResourceInstance
	GetAPIServiceWithPrimaryKey(primaryKey string) *v1.ResourceInstance
	GetAPIServiceWithName(apiName string) *v1.ResourceInstance
	DeleteAPIService(externalAPIID string) error

	// API service instance cache related methods
	AddAPIServiceInstance(resource *v1.ResourceInstance)
	GetAPIServiceInstanceKeys() []string
	GetAPIServiceInstanceByID(instanceID string) (*v1.ResourceInstance, error)
	DeleteAPIServiceInstance(instanceID string) error
	DeleteAllAPIServiceInstance()

	// Category cache related methods
	AddCategory(resource *v1.ResourceInstance)
	GetCategoryCache() cache.Cache
	GetCategoryKeys() []string
	GetCategory(categoryName string) *v1.ResourceInstance
	GetCategoryWithTitle(title string) *v1.ResourceInstance
	DeleteCategory(categoryName string) error

	// Watch Sequence cache related methods
	AddSequence(watchTopicName string, sequenceID int64)
	GetSequence(watchTopicName string) int64

	GetTeamCache() cache.Cache
	AddTeam(team *definitions.PlatformTeam)
	GetTeamByName(name string) *definitions.PlatformTeam
	GetTeamByID(id string) *definitions.PlatformTeam
	GetDefaultTeam() *definitions.PlatformTeam
}

type cacheManager struct {
	jobs.Job
	apiMap                  cache.Cache
	instanceMap             cache.Cache
	categoryMap             cache.Cache
	sequenceCache           cache.Cache
	cacheLock               sync.Mutex
	persistedCache          cache.Cache
	teams                   cache.Cache
	cacheFilename           string
	hasLoadedPersistedCache bool
	isCacheUpdated          bool
}

// NewAgentCacheManager - Create a new agent cache manager
func NewAgentCacheManager(cfg config.CentralConfig, persistCache bool) Manager {
	// todo - make path configurable
	m := &cacheManager{
		apiMap:         cache.New(),
		instanceMap:    cache.New(),
		categoryMap:    cache.New(),
		sequenceCache:  cache.New(),
		teams:          cache.New(),
		isCacheUpdated: false,
	}

	if cfg.IsUsingGRPC() && persistCache {
		m.initializePersistedCache(cfg)
	}

	return m
}

func (c *cacheManager) initializePersistedCache(cfg config.CentralConfig) {
	c.cacheFilename = c.getCacheFileName(cfg)

	cacheMap := cache.New()
	err := cacheMap.Load(c.cacheFilename)
	if err == nil {
		c.apiMap = c.loadPersistedResourceInstanceCache(cacheMap, "apiServices")
		c.instanceMap = c.loadPersistedResourceInstanceCache(cacheMap, "apiServiceInstances")
		c.categoryMap = c.loadPersistedResourceInstanceCache(cacheMap, "categories")
		c.sequenceCache = c.loadPersistedCache(cacheMap, "watchSequence")
		c.teams = c.loadPersistedCache(cacheMap, "teamCache")
		c.hasLoadedPersistedCache = true
		c.isCacheUpdated = false
	}

	cacheMap.Set("apiServices", c.apiMap)
	cacheMap.Set("apiServiceInstances", c.instanceMap)
	cacheMap.Set("categories", c.categoryMap)
	cacheMap.Set("watchSequence", c.sequenceCache)
	cacheMap.Set("teams", c.teams)
	c.persistedCache = cacheMap
	if util.IsNotTest() {
		jobs.RegisterIntervalJobWithName(c, cfg.GetCacheStorageInterval(), "Agent cache persistence")
	}
}

func (c *cacheManager) getCacheFileName(cfg config.CentralConfig) string {
	cacheStoragePath := cfg.GetCacheStoragePath()
	if cacheStoragePath == "" {
		cacheStoragePath = defaultCacheStoragePath
	}
	util.CreateDirIfNotExist(cacheStoragePath)
	if cfg.GetAgentName() != "" {
		return cacheStoragePath + "/" + cfg.GetAgentName() + ".cache"
	}
	return cacheStoragePath + "/" + cfg.GetEnvironmentName() + ".cache"
}

func (c *cacheManager) loadPersistedCache(cacheMap cache.Cache, cacheKey string) cache.Cache {
	itemCache, _ := cacheMap.Get(cacheKey)
	if itemCache != nil {
		raw, _ := json.Marshal(itemCache)
		return cache.LoadFromBuffer(raw)
	}
	return cache.New()
}

func (c *cacheManager) loadPersistedResourceInstanceCache(cacheMap cache.Cache, cacheKey string) cache.Cache {
	riCache := c.loadPersistedCache(cacheMap, cacheKey)
	keys := riCache.GetKeys()
	for _, key := range keys {
		item, _ := riCache.Get(key)
		rawResource, _ := json.Marshal(item)
		ri := &v1.ResourceInstance{}
		json.Unmarshal(rawResource, ri)
		riCache.Set(key, ri)
	}
	return riCache
}

func (c *cacheManager) setCacheUpdated(updated bool) {
	c.isCacheUpdated = updated
}

// Cache persistence job

// Ready -
func (c *cacheManager) Ready() bool {
	return true
}

// Status -
func (c *cacheManager) Status() error {
	return nil
}

// Execute - persists the cache to file
func (c *cacheManager) Execute() error {
	if util.IsNotTest() && c.isCacheUpdated {
		log.Trace("executing cache persistence job")
		c.SaveCache()
	}
	return nil
}

// Cache manager

// HasLoadedPersistedCache - returns true if the caches are loaded from file
func (c *cacheManager) HasLoadedPersistedCache() bool {
	return c.hasLoadedPersistedCache
}

func (c *cacheManager) SaveCache() {
	if c.persistedCache != nil {
		c.cacheLock.Lock()
		defer c.cacheLock.Unlock()
		c.persistedCache.Save(c.cacheFilename)
		c.setCacheUpdated(false)
	}
}

// API service cache management

// AddAPIService - add/update APIService resource in cache
func (c *cacheManager) AddAPIService(apiService *v1.ResourceInstance) string {
	externalAPIID, ok := apiService.Attributes[definitions.AttrExternalAPIID]
	if ok {
		defer c.setCacheUpdated(true)
		externalAPIName := apiService.Attributes[definitions.AttrExternalAPIName]
		if externalAPIPrimaryKey, found := apiService.Attributes[definitions.AttrExternalAPIPrimaryKey]; found {
			// Verify secondary key and validate if we need to remove it from the apiMap (cache)
			if _, err := c.apiMap.Get(externalAPIID); err != nil {
				c.apiMap.Delete(externalAPIID)
			}

			c.apiMap.SetWithSecondaryKey(externalAPIPrimaryKey, externalAPIID, apiService)
			c.apiMap.SetSecondaryKey(externalAPIPrimaryKey, externalAPIName)
		} else {
			c.apiMap.SetWithSecondaryKey(externalAPIID, externalAPIName, apiService)
		}
		log.Tracef("added api name: %s, id %s to API cache", externalAPIName, externalAPIID)
	}
	return externalAPIID
}

// GetAPIServiceCache - returns the APIService cache
func (c *cacheManager) GetAPIServiceCache() cache.Cache {
	return c.apiMap
}

// GetAPIServiceKeys - returns keys for APIService cache
func (c *cacheManager) GetAPIServiceKeys() []string {
	return c.apiMap.GetKeys()
}

// GetAPIServiceWithAPIID - returns resource from APIService cache based on externalAPIID attribute
func (c *cacheManager) GetAPIServiceWithAPIID(externalAPIID string) *v1.ResourceInstance {
	api, _ := c.apiMap.Get(externalAPIID)
	if api == nil {
		api, _ = c.apiMap.GetBySecondaryKey(externalAPIID)
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
	api, _ := c.apiMap.GetBySecondaryKey(apiName)
	if api != nil {
		apiSvc, ok := api.(*v1.ResourceInstance)
		if ok {
			return apiSvc
		}
	}
	return nil
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

// API service instance management

// AddAPIServiceInstance -  add/update APIServiceInstance resource in cache
func (c *cacheManager) AddAPIServiceInstance(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.instanceMap.Set(resource.Metadata.ID, resource)
}

// GetAPIServiceInstanceKeys - returns keys for APIServiceInstance cache
func (c *cacheManager) GetAPIServiceInstanceKeys() []string {
	return c.instanceMap.GetKeys()
}

// GetAPIServiceInstanceByID - returns resource from APIServiceInstance cache based on instance ID
func (c *cacheManager) GetAPIServiceInstanceByID(instanceID string) (*v1.ResourceInstance, error) {
	item, err := c.instanceMap.Get(instanceID)
	if item != nil {
		instance, ok := item.(*v1.ResourceInstance)
		if ok {
			return instance, nil
		}
	}
	return nil, err
}

// DeleteAPIServiceInstance - remove APIServiceInstance resource from cache based on instance ID
func (c *cacheManager) DeleteAPIServiceInstance(instanceID string) error {
	defer c.setCacheUpdated(true)

	return c.instanceMap.Delete(instanceID)
}

// DeleteAllAPIServiceInstance - remove all APIServiceInstance resource from cache
func (c *cacheManager) DeleteAllAPIServiceInstance() {
	defer c.setCacheUpdated(true)

	c.instanceMap.Flush()
}

// Category cache management

// AddCategory - add/update Category resource in cache
func (c *cacheManager) AddCategory(resource *v1.ResourceInstance) {
	defer c.setCacheUpdated(true)

	c.categoryMap.SetWithSecondaryKey(resource.Name, resource.Title, resource)
}

// GetCategoryCache - returns the Category cache
func (c *cacheManager) GetCategoryCache() cache.Cache {
	return c.categoryMap
}

// GetCategoryKeys - returns keys for Category cache
func (c *cacheManager) GetCategoryKeys() []string {
	return c.categoryMap.GetKeys()
}

// GetCategory - returns resource from Category cache based on name
func (c *cacheManager) GetCategory(categoryName string) *v1.ResourceInstance {
	category, _ := c.categoryMap.Get(categoryName)
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
	category, _ := c.categoryMap.GetBySecondaryKey(title)
	if category != nil {
		ri, ok := category.(*v1.ResourceInstance)
		if ok {
			return ri
		}
	}
	return nil
}

// DeleteCategory - remove Category resource from cache based on name
func (c *cacheManager) DeleteCategory(categoryName string) error {
	defer c.setCacheUpdated(true)

	return c.categoryMap.Delete(categoryName)
}

// Watch Sequence cache

// AddSequence - add/updates the sequenceID for the watch topic in cache
func (c *cacheManager) AddSequence(watchTopicName string, sequenceID int64) {
	defer c.setCacheUpdated(true)

	c.sequenceCache.Set(watchTopicName, sequenceID)
}

// GetSequence - returns the sequenceID for the watch topic in cache
func (c *cacheManager) GetSequence(watchTopicName string) int64 {
	cachedSeqID, err := c.sequenceCache.Get(watchTopicName)
	if err == nil {
		if seqID, ok := cachedSeqID.(int64); ok {
			return seqID
		} else if seqID, ok := cachedSeqID.(float64); ok {
			return int64(seqID)
		}
	}
	return 0
}

// GetTeamCache - returns the team cache
func (c *cacheManager) GetTeamCache() cache.Cache {
	return c.teams
}

// AddTeam saves a team to the cache
func (c *cacheManager) AddTeam(team *definitions.PlatformTeam) {
	defer c.setCacheUpdated(true)
	c.teams.SetWithSecondaryKey(team.Name, team.ID, team)
}

// GetTeamByName gets a team by name
func (c *cacheManager) GetTeamByName(name string) *definitions.PlatformTeam {
	item, err := c.teams.Get(name)
	if err != nil {
		return nil
	}
	team, ok := item.(*definitions.PlatformTeam)
	if !ok {
		return nil
	}
	return team
}

// GetDefaultTeam gets the default team
func (c *cacheManager) GetDefaultTeam() *definitions.PlatformTeam {
	names := c.teams.GetKeys()

	var defaultTeam *definitions.PlatformTeam
	for _, name := range names {
		item, _ := c.teams.Get(name)
		team, ok := item.(*definitions.PlatformTeam)
		if !ok {
			continue
		}

		if team.Default {
			defaultTeam = team
			break
		}

		continue
	}

	return defaultTeam
}

// GetTeamByID gets a team by id
func (c *cacheManager) GetTeamByID(id string) *definitions.PlatformTeam {
	item, err := c.teams.GetBySecondaryKey(id)
	if err != nil {
		return nil
	}
	team, ok := item.(*definitions.PlatformTeam)
	if !ok {
		return nil
	}
	return team
}
