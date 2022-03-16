package cache

import (
	"encoding/json"
	"sync"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const defaultCacheStoragePath = "./data/cache"

// Manager - interface to manage agent resource
type Manager interface {

	// Cache management related methods
	HasLoadedPersistedCache() bool
	SaveCache()

	// API Service cache related methods
	AddAPIService(resource *v1.ResourceInstance) error
	GetAPIServiceCache() cache.Cache
	GetAPIServiceKeys() []string
	GetAPIServiceWithAPIID(apiID string) *v1.ResourceInstance
	GetAPIServiceWithPrimaryKey(primaryKey string) *v1.ResourceInstance
	GetAPIServiceWithName(apiName string) *v1.ResourceInstance
	GetTeamsIDsInAPIServices() []string
	DeleteAPIService(apiID string) error

	// API service instance cache related methods
	AddAPIServiceInstance(resource *v1.ResourceInstance)
	GetAPIServiceInstanceKeys() []string
	GetAPIServiceInstanceByID(id string) (*v1.ResourceInstance, error)
	DeleteAPIServiceInstance(id string) error
	DeleteAllAPIServiceInstance()

	// Category cache related methods
	AddCategory(resource *v1.ResourceInstance)
	GetCategoryCache() cache.Cache
	GetCategoryKeys() []string
	GetCategory(name string) *v1.ResourceInstance
	GetCategoryWithTitle(title string) *v1.ResourceInstance
	DeleteCategory(name string) error

	// Team and ACL related cache methods
	GetTeamCache() cache.Cache
	AddTeam(team *defs.PlatformTeam)
	GetTeamByName(name string) *defs.PlatformTeam
	GetTeamByID(id string) *defs.PlatformTeam
	GetDefaultTeam() *defs.PlatformTeam
	SetAccessControlList(acl *v1.ResourceInstance)
	GetAccessControlList() *v1.ResourceInstance

	// AccessRequestDefintion cache related methods
	AddAccessRequestDefinition(resource *v1.ResourceInstance)
	GetAccessRequestDefinitionByName(name string) *v1.ResourceInstance
	DeleteAccessRequestDefinitionByName(name string) error

	// CredentialRequestDefintion cache related methods
	AddCredentialRequestDefinition(resource *v1.ResourceInstance)
	GetCredentialRequestDefinitionByName(name string) *v1.ResourceInstance
	DeleteCredentialRequestDefinitionByName(name string) error

	// Watch Sequence cache related methods
	AddSequence(watchTopicName string, sequenceID int64)
	GetSequence(watchTopicName string) int64

	ApplyResourceReadLock()
	ReleaseResourceReadLock()
}

type cacheManager struct {
	jobs.Job
	apiMap                  cache.Cache
	instanceMap             cache.Cache
	categoryMap             cache.Cache
	sequenceCache           cache.Cache
	resourceCacheReadLock   sync.Mutex
	cacheLock               sync.Mutex
	persistedCache          cache.Cache
	teams                   cache.Cache
	ardMap                  cache.Cache
	crdMap                  cache.Cache
	cacheFilename           string
	hasLoadedPersistedCache bool
	isCacheUpdated          bool
}

// NewAgentCacheManager - Create a new agent cache manager
func NewAgentCacheManager(cfg config.CentralConfig, persistCache bool) Manager {
	m := &cacheManager{
		apiMap:         cache.New(),
		instanceMap:    cache.New(),
		categoryMap:    cache.New(),
		sequenceCache:  cache.New(),
		teams:          cache.New(),
		ardMap:         cache.New(),
		crdMap:         cache.New(),
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
	cachePath := cfg.GetCacheStoragePath()
	if cachePath == "" {
		cachePath = defaultCacheStoragePath
	}
	util.CreateDirIfNotExist(cachePath)
	if cfg.GetAgentName() != "" {
		return cachePath + "/" + cfg.GetAgentName() + ".cache"
	}
	return cachePath + "/" + cfg.GetEnvironmentName() + ".cache"
}

func (c *cacheManager) loadPersistedCache(cacheMap cache.Cache, key string) cache.Cache {
	itemCache, _ := cacheMap.Get(key)
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

func (c *cacheManager) ApplyResourceReadLock() {
	c.resourceCacheReadLock.Lock()
}

func (c *cacheManager) ReleaseResourceReadLock() {
	c.resourceCacheReadLock.Unlock()
}
