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
	Flush()

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
	GetAPIServiceInstanceByName(apiName string) (*v1.ResourceInstance, error)
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
	DeleteAccessControlList() error

	// AccessRequestDefintion cache related methods
	AddAccessRequestDefinition(resource *v1.ResourceInstance)
	GetAccessRequestDefinitionKeys() []string
	GetAccessRequestDefinitionByName(name string) (*v1.ResourceInstance, error)
	GetAccessRequestDefinitionByID(id string) (*v1.ResourceInstance, error)
	DeleteAccessRequestDefinition(id string) error

	// CredentialRequestDefintion cache related methods
	AddCredentialRequestDefinition(resource *v1.ResourceInstance)
	GetCredentialRequestDefinitionKeys() []string
	GetCredentialRequestDefinitionByName(name string) (*v1.ResourceInstance, error)
	GetCredentialRequestDefinitionByID(id string) (*v1.ResourceInstance, error)
	DeleteCredentialRequestDefinition(id string) error

	// Watch Sequence cache related methods
	AddSequence(watchTopicName string, sequenceID int64)
	GetSequence(watchTopicName string) int64
	AddFetchOnStartupResources([]*v1.ResourceInstance)
	GetAllFetchOnStartupResources() []*v1.ResourceInstance
	DeleteAllFetchOnStartupResources() error

	// ManagedApplication cache related methods
	GetManagedApplicationCacheKeys() []string
	AddManagedApplication(resource *v1.ResourceInstance)
	GetManagedApplication(id string) *v1.ResourceInstance
	GetManagedApplicationByName(name string) *v1.ResourceInstance
	DeleteManagedApplication(id string) error

	// AccessRequest cache related methods
	GetAccessRequestCacheKeys() []string
	AddAccessRequest(resource *v1.ResourceInstance)
	GetAccessRequestByAppAndAPI(managedAppName, remoteAPIID, remoteAPIStage string) *v1.ResourceInstance
	GetAccessRequest(id string) *v1.ResourceInstance
	DeleteAccessRequest(id string) error

	ApplyResourceReadLock()
	ReleaseResourceReadLock()
}

type cacheManager struct {
	jobs.Job
	logger                  log.FieldLogger
	apiMap                  cache.Cache
	instanceMap             cache.Cache
	categoryMap             cache.Cache
	managedApplicationMap   cache.Cache
	accessRequestMap        cache.Cache
	subscriptionMap         cache.Cache
	sequenceCache           cache.Cache
	fetchOnStartup          cache.Cache
	resourceCacheReadLock   sync.Mutex
	cacheLock               sync.Mutex
	persistedCache          cache.Cache
	teams                   cache.Cache
	ardMap                  cache.Cache
	crdMap                  cache.Cache
	cacheFilename           string
	isPersistedCacheLoaded  bool
	isCacheUpdated          bool
	isPersistedCacheEnabled bool
}

// NewAgentCacheManager - Create a new agent cache manager
func NewAgentCacheManager(cfg config.CentralConfig, persistCacheEnabled bool) Manager {
	logger := log.NewFieldLogger().
		WithComponent("cacheManager").
		WithPackage("sdk.agent.cache")
	m := &cacheManager{
		apiMap:                  cache.New(),
		instanceMap:             cache.New(),
		categoryMap:             cache.New(),
		managedApplicationMap:   cache.New(),
		accessRequestMap:        cache.New(),
		subscriptionMap:         cache.New(),
		sequenceCache:           cache.New(),
		fetchOnStartup:          cache.New(),
		teams:                   cache.New(),
		ardMap:                  cache.New(),
		crdMap:                  cache.New(),
		isCacheUpdated:          false,
		logger:                  logger,
		isPersistedCacheEnabled: persistCacheEnabled,
	}

	if m.isPersistedCacheEnabled {
		m.initializePersistedCache(cfg)
	}

	return m
}

func (c *cacheManager) initializePersistedCache(cfg config.CentralConfig) {
	c.cacheFilename = c.getCacheFileName(cfg)

	cacheMap := cache.New()
	cacheMap.Load(c.cacheFilename)

	cacheKeys := map[string]func(cache.Cache){
		"apiServices":         func(loaded cache.Cache) { c.apiMap = loaded },
		"apiServiceInstances": func(loaded cache.Cache) { c.instanceMap = loaded },
		"categories":          func(loaded cache.Cache) { c.categoryMap = loaded },
		"credReqDef":          func(loaded cache.Cache) { c.crdMap = loaded },
		"accReqDef":           func(loaded cache.Cache) { c.ardMap = loaded },
		"teams":               func(loaded cache.Cache) { c.teams = loaded },
		"managedApp":          func(loaded cache.Cache) { c.managedApplicationMap = loaded },
		"subscriptions":       func(loaded cache.Cache) { c.subscriptionMap = loaded },
		"accReq":              func(loaded cache.Cache) { c.accessRequestMap = loaded },
		"watchSequence":       func(loaded cache.Cache) { c.sequenceCache = loaded },
		"fetchOnStartup":      func(loaded cache.Cache) { c.fetchOnStartup = loaded },
	}

	c.isPersistedCacheLoaded = true
	c.isCacheUpdated = false
	for key := range cacheKeys {
		loadedMap, isNew := c.loadPersistedResourceInstanceCache(cacheMap, key)
		if isNew {
			c.isPersistedCacheLoaded = false
		}
		cacheKeys[key](loadedMap)
	}

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

func (c *cacheManager) loadPersistedCache(cacheMap cache.Cache, key string) (cache.Cache, bool) {
	itemCache, _ := cacheMap.Get(key)
	if itemCache != nil {
		raw, _ := json.Marshal(itemCache)
		return cache.LoadFromBuffer(raw), false
	}
	return cache.New(), true
}

func (c *cacheManager) loadPersistedResourceInstanceCache(cacheMap cache.Cache, cacheKey string) (cache.Cache, bool) {
	riCache, isNew := c.loadPersistedCache(cacheMap, cacheKey)
	keys := riCache.GetKeys()
	for _, key := range keys {
		item, _ := riCache.Get(key)
		rawResource, _ := json.Marshal(item)
		ri := &v1.ResourceInstance{}
		if json.Unmarshal(rawResource, ri) == nil {
			riCache.Set(key, ri)
		}
	}
	cacheMap.Set(cacheKey, riCache)
	return riCache, isNew
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
		c.logger.Trace("executing cache persistence job")
		c.SaveCache()
	}
	return nil
}

// Cache manager

// HasLoadedPersistedCache - returns true if the caches are loaded from file
func (c *cacheManager) HasLoadedPersistedCache() bool {
	return c.isPersistedCacheLoaded
}

// SaveCache - writes the cache to a file
func (c *cacheManager) SaveCache() {
	if c.persistedCache != nil && c.isPersistedCacheEnabled {
		c.cacheLock.Lock()
		defer c.cacheLock.Unlock()
		c.persistedCache.Save(c.cacheFilename)
		c.setCacheUpdated(false)
		c.logger.Debug("persistent cache has been saved")
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

// Flush empties the persistent cache and all internal caches
func (c *cacheManager) Flush() {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()
	c.logger.Debug("resetting the persistent cache")

	c.accessRequestMap.Flush()
	c.apiMap.Flush()
	c.ardMap.Flush()
	c.categoryMap.Flush()
	c.crdMap.Flush()
	c.instanceMap.Flush()
	c.managedApplicationMap.Flush()
	c.sequenceCache.Flush()
	c.subscriptionMap.Flush()

	c.SaveCache()
}
