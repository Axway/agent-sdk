package cache

import (
	"encoding/json"
	"os"
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

// cache keys
const (
	apiServicesKey         = "apiServices"
	apiServiceInstancesKey = "apiServiceInstances"
	instanceCountKey       = "instanceCount"
	credReqDefKey          = "credReqDef"
	accReqDefKey           = "accReqDef"
	appProfDefKey          = "appProfDef"
	teamsKey               = "teams"
	managedAppKey          = "managedApp"
	subscriptionsKey       = "subscriptions"
	accReqKey              = "accReq"
	watchSequenceKey       = "watchSequence"
	watchResourceKey       = "watchResource"
	complianceRuntimeKey   = "compRunRes"
)

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
	GetAPIServiceInstancesByService(apiName string) []*v1.ResourceInstance
	GetTeamsIDsInAPIServices() []string
	DeleteAPIService(apiID string) error

	// API service instance cache related methods
	AddAPIServiceInstance(resource *v1.ResourceInstance)
	GetAPIServiceInstanceKeys() []string
	GetAPIServiceInstanceByID(id string) (*v1.ResourceInstance, error)
	GetAPIServiceInstanceByName(apiName string) (*v1.ResourceInstance, error)
	DeleteAPIServiceInstance(id string) error
	DeleteAllAPIServiceInstance()
	ListAPIServiceInstances() []*v1.ResourceInstance

	// Team and ACL related cache methods
	GetTeamCache() cache.Cache
	AddTeam(team *defs.PlatformTeam)
	GetTeamByName(name string) *defs.PlatformTeam
	GetTeamByID(id string) *defs.PlatformTeam
	GetDefaultTeam() *defs.PlatformTeam
	SetAccessControlList(acl *v1.ResourceInstance)
	GetAccessControlList() *v1.ResourceInstance
	DeleteAccessControlList() error

	// ApplicationProfileDefinition cache related methods
	AddApplicationProfileDefinition(resource *v1.ResourceInstance)
	GetApplicationProfileDefinitionKeys() []string
	GetApplicationProfileDefinitionByName(name string) (*v1.ResourceInstance, error)
	GetApplicationProfileDefinitionByID(id string) (*v1.ResourceInstance, error)
	DeleteApplicationProfileDefinition(id string) error

	// ComplianceRuntimeResult cache related methods
	AddComplianceRuntimeResult(resource *v1.ResourceInstance)
	GetComplianceRuntimeResultKeys() []string
	GetComplianceRuntimeResultByName(name string) (*v1.ResourceInstance, error)
	GetComplianceRuntimeResultByID(id string) (*v1.ResourceInstance, error)
	DeleteComplianceRuntimeResult(id string) error

	// AccessRequestDefinition cache related methods
	AddAccessRequestDefinition(resource *v1.ResourceInstance)
	GetAccessRequestDefinitionKeys() []string
	GetAccessRequestDefinitionByName(name string) (*v1.ResourceInstance, error)
	GetAccessRequestDefinitionByID(id string) (*v1.ResourceInstance, error)
	DeleteAccessRequestDefinition(id string) error

	// CredentialRequestDefinition cache related methods
	AddCredentialRequestDefinition(resource *v1.ResourceInstance)
	GetCredentialRequestDefinitionKeys() []string
	GetCredentialRequestDefinitionByName(name string) (*v1.ResourceInstance, error)
	GetCredentialRequestDefinitionByID(id string) (*v1.ResourceInstance, error)
	DeleteCredentialRequestDefinition(id string) error
	ListCredentialRequestDefinitions() []*v1.ResourceInstance

	// Watch Sequence cache related methods
	AddSequence(watchTopicName string, sequenceID int64)
	GetSequence(watchTopicName string) int64

	// ManagedApplication cache related methods
	GetManagedApplicationCacheKeys() []string
	AddManagedApplication(resource *v1.ResourceInstance)
	GetManagedApplication(id string) *v1.ResourceInstance
	GetManagedApplicationByApplicationID(id string) *v1.ResourceInstance
	GetManagedApplicationByName(name string) *v1.ResourceInstance
	DeleteManagedApplication(id string) error

	// AccessRequest cache related methods
	GetAccessRequestCacheKeys() []string
	AddAccessRequest(resource *v1.ResourceInstance)
	GetAccessRequestByAppAndAPI(managedAppName, remoteAPIID, remoteAPIStage string) *v1.ResourceInstance
	GetAccessRequestByAppAndAPIStageVersion(managedAppName, remoteAPIID, remoteAPIStage, remoteAPIVersion string) *v1.ResourceInstance
	GetAccessRequest(id string) *v1.ResourceInstance
	GetAccessRequestsByApp(managedAppName string) []*v1.ResourceInstance
	DeleteAccessRequest(id string) error
	ListAccessRequests() []*v1.ResourceInstance

	GetWatchResourceCacheKeys(group, kind string) []string
	AddWatchResource(resource *v1.ResourceInstance)
	GetWatchResourceByKey(key string) *v1.ResourceInstance
	GetWatchResourceByID(group, kind, id string) *v1.ResourceInstance
	GetWatchResourceByName(group, kind, name string) *v1.ResourceInstance
	DeleteWatchResource(group, kind, id string) error

	ApplyResourceReadLock()
	ReleaseResourceReadLock()
}

type cacheLoader interface {
	loaded(c cache.Cache)
	unmarshaller(data []byte) (interface{}, error)
	getkey() string
}

type cacheManager struct {
	jobs.Job
	logger                  log.FieldLogger
	apiMap                  cache.Cache
	instanceCountMap        cache.Cache
	instanceMap             cache.Cache
	managedApplicationMap   cache.Cache
	accessRequestMap        cache.Cache
	watchResourceMap        cache.Cache
	subscriptionMap         cache.Cache
	sequenceCache           cache.Cache
	resourceCacheReadLock   sync.Mutex
	cacheLock               sync.Mutex
	persistedCache          cache.Cache
	teams                   cache.Cache
	ardMap                  cache.Cache
	apdMap                  cache.Cache
	crdMap                  cache.Cache
	crrMap                  cache.Cache
	cacheFilename           string
	isPersistedCacheLoaded  bool
	isCacheUpdated          bool
	isPersistedCacheEnabled bool
	migrators               []cacheMigrate
}

// NewAgentCacheManager - Create a new agent cache manager
func NewAgentCacheManager(cfg config.CentralConfig, persistCacheEnabled bool) Manager {
	logger := log.NewFieldLogger().
		WithComponent("cacheManager").
		WithPackage("sdk.agent.cache")
	m := &cacheManager{
		isCacheUpdated:          false,
		logger:                  logger,
		isPersistedCacheEnabled: persistCacheEnabled,
		migrators:               []cacheMigrate{},
	}

	// add migrators here if needed
	m.initializeCache(cfg)

	return m
}

func (c *cacheManager) initializeCache(cfg config.CentralConfig) {
	cacheMap := cache.New()
	if c.isPersistedCacheEnabled {
		c.cacheFilename = c.getCacheFileName(cfg)
		cacheMap.Load(c.cacheFilename)
	}

	cacheLoaders := []cacheLoader{
		createResourceLoader(c.setLoadedCache, apiServicesKey),
		createResourceLoader(c.setLoadedCache, apiServiceInstancesKey),
		createResourceLoader(c.setLoadedCache, credReqDefKey),
		createResourceLoader(c.setLoadedCache, accReqDefKey),
		createResourceLoader(c.setLoadedCache, appProfDefKey),
		createResourceLoader(c.setLoadedCache, managedAppKey),
		createResourceLoader(c.setLoadedCache, subscriptionsKey),
		createResourceLoader(c.setLoadedCache, accReqKey),
		createResourceLoader(c.setLoadedCache, watchResourceKey),
		createInstanceCountLoader(c.setLoadedCache, instanceCountKey),
		createTeamLoader(c.setLoadedCache, teamsKey),
		createSequenceLoader(c.setLoadedCache, watchSequenceKey),
		createResourceLoader(c.setLoadedCache, complianceRuntimeKey),
	}

	c.isPersistedCacheLoaded = true
	c.isCacheUpdated = false
	for _, loader := range cacheLoaders {
		loadedMap, loadNew := c.loadPersistedResourceInstanceCache(cacheMap, loader)
		if loadNew {
			c.isPersistedCacheLoaded = false
		}
		cacheMap.Set(loader.getkey(), loadedMap)
		loader.loaded(loadedMap)
	}

	if c.isPersistedCacheLoaded {
		// after loading, successfully, check for migrations
		for _, loader := range cacheLoaders {
			c.migratePersistentCache(loader.getkey())
		}
	} else {
		// flush all caches if any of the persisted caches failed loaded properly
		c.logger.Info("persisted store failed to load, refreshing cache")
		c.Flush()
	}

	c.persistedCache = cacheMap
	if c.isPersistedCacheEnabled && util.IsNotTest() {
		jobs.RegisterIntervalJobWithName(c, cfg.GetCacheStorageInterval(), "Agent cache persistence")
	}
}

func (c *cacheManager) setLoadedCache(lc cache.Cache, key string) {
	c.logger.WithField("cacheKey", key).Debug("cache loaded")
	switch key {
	case apiServicesKey:
		c.apiMap = lc
	case apiServiceInstancesKey:
		c.instanceMap = lc
	case instanceCountKey:
		c.instanceCountMap = lc
	case credReqDefKey:
		c.crdMap = lc
	case accReqDefKey:
		c.ardMap = lc
	case appProfDefKey:
		c.apdMap = lc
	case teamsKey:
		c.teams = lc
	case managedAppKey:
		c.managedApplicationMap = lc
	case subscriptionsKey:
		c.subscriptionMap = lc
	case accReqKey:
		c.accessRequestMap = lc
	case watchResourceKey:
		c.watchResourceMap = lc
	case watchSequenceKey:
		c.sequenceCache = lc
	case complianceRuntimeKey:
		c.crrMap = lc
	default:
		c.logger.WithField("cacheKey", key).Error("unknown cache key")
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
	c.logger = c.logger.WithField("cachePath", cachePath)
	return cachePath + "/" + cfg.GetEnvironmentName() + ".cache"
}

func (c *cacheManager) loadPersistedCache(cacheMap cache.Cache, key string) (cache.Cache, bool) {
	if !c.isPersistedCacheLoaded {
		// return as soon as possible
		return cache.New(), true
	}
	itemCache, _ := cacheMap.Get(key)
	if itemCache != nil {
		raw, _ := json.Marshal(itemCache)
		return cache.LoadFromBuffer(raw), false
	}
	return cache.New(), true
}

func (c *cacheManager) loadPersistedResourceInstanceCache(cacheMap cache.Cache, loader cacheLoader) (cache.Cache, bool) {
	riCache, isNew := c.loadPersistedCache(cacheMap, loader.getkey())
	if isNew {
		return riCache, isNew
	}

	// If the cache is not new, we need to load the data from the persisted store
	keys := riCache.GetKeys()
	logger := c.logger.WithField("cacheKey", loader.getkey())
	logger.Debug("loading cache from persisted store")
	for _, key := range keys {
		logger = logger.WithField("key", key)
		logger.Trace("loading data for key")
		item, err := riCache.Get(key)
		if err != nil {
			logger.WithError(err).Error("reading item from cache, refreshing cache")
			riCache = cache.New()
			return riCache, true
		}
		rawResource, err := json.Marshal(item)
		if err != nil {
			logger.WithError(err).Error("reading data from cache, refreshing cache")
			riCache = cache.New()
			return riCache, true
		}
		toCache, err := loader.unmarshaller(rawResource)
		if err != nil {
			c.logger.WithError(err).Errorf("failed to load data into cache")
			riCache = cache.New()
			return riCache, true
		}
		riCache.Set(key, toCache)
	}

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
	c.apdMap.Flush()
	c.crrMap.Flush()
	c.crdMap.Flush()
	c.instanceMap.Flush()
	c.managedApplicationMap.Flush()
	c.sequenceCache.Flush()
	c.subscriptionMap.Flush()
	c.watchResourceMap.Flush()
	c.SaveCache()
	// delete the cache file in case the agent is restarted here
	os.Remove(c.cacheFilename)
}
