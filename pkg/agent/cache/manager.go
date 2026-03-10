package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"go.etcd.io/bbolt"
)

const defaultCacheStoragePath = "./data/cache"

// dbHandle tracks a shared database file with reference counting
type dbHandle struct {
	db       *bbolt.DB
	refCount atomic.Int32
}

// sharedBoltDBs manages shared database handles across multiple cache managers
// This prevents file locking issues when multiple agents share the same database
var (
	sharedBoltDBsLock sync.RWMutex
	sharedBoltDBs     = make(map[string]*dbHandle)
)

// getSharedDB retrieves or creates a shared database handle, incrementing reference count
func getSharedDB(dbPath string) (*bbolt.DB, error) {
	sharedBoltDBsLock.Lock()
	defer sharedBoltDBsLock.Unlock()

	// Check if handle already exists
	if handle, exists := sharedBoltDBs[dbPath]; exists && handle.db != nil {
		handle.refCount.Add(1)
		return handle.db, nil
	}

	// Try to open in read-write mode first
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		// Try read-only mode
		db, err = bbolt.Open(dbPath, 0600, &bbolt.Options{
			Timeout:  1 * time.Second,
			ReadOnly: true,
		})
		if err != nil {
			return nil, err
		}
	}

	// Store handle with ref count of 1
	sharedBoltDBs[dbPath] = &dbHandle{
		db: db,
	}
	sharedBoltDBs[dbPath].refCount.Store(1)
	return db, nil
}

// releaseSharedDB decrements the reference count and closes the database if count reaches zero
func releaseSharedDB(dbPath string) error {
	sharedBoltDBsLock.Lock()
	defer sharedBoltDBsLock.Unlock()

	handle, exists := sharedBoltDBs[dbPath]
	if !exists || handle.db == nil {
		return nil
	}

	newCount := handle.refCount.Add(-1)
	if newCount <= 0 {
		err := handle.db.Close()
		delete(sharedBoltDBs, dbPath)
		return err
	}

	return nil
}

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
	Flush()
	Close() error

	// API Service cache related methods
	AddAPIService(resource *v1.ResourceInstance) error
	GetAPIServiceCache() cache.Cache
	GetAPIServiceKeys() []string
	GetAPIServiceWithAPIID(apiID string) *v1.ResourceInstance
	GetAPIServiceWithPrimaryKey(primaryKey string) *v1.ResourceInstance
	GetAPIServiceWithName(apiName string) *v1.ResourceInstance
	GetAPIServiceInstanceCount(apiName string) int
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
	logger                  log.FieldLogger
	db                      *bbolt.DB
	dbPath                  string
	apiMap                  *boltStore
	instanceCountMap        *boltStore
	instanceMap             *boltStore
	managedApplicationMap   *boltStore
	accessRequestMap        *boltStore
	watchResourceMap        *boltStore
	subscriptionMap         *boltStore
	sequenceCache           *boltStore
	resourceCacheReadLock   sync.Mutex
	teams                   *boltStore
	ardMap                  *boltStore
	apdMap                  *boltStore
	crdMap                  *boltStore
	crrMap                  *boltStore
	cacheFilename           string
	isPersistedCacheLoaded  bool
	isPersistedCacheEnabled bool
}

// NewAgentCacheManager - Create a new agent cache manager
func NewAgentCacheManager(cfg config.CentralConfig, persistCacheEnabled bool) Manager {
	logger := log.NewFieldLogger().
		WithComponent("cacheManager").
		WithPackage("sdk.agent.cache")
	m := &cacheManager{
		logger:                  logger,
		isPersistedCacheEnabled: persistCacheEnabled,
	}

	m.initializeCache(cfg)

	return m
}

func (c *cacheManager) initializeCache(cfg config.CentralConfig) {
	c.cacheFilename = c.getCacheFileName(cfg)

	// Open or create bbolt database
	var err error
	var dbPath string

	if c.isPersistedCacheEnabled {
		if !util.IsNotTest() && cfg.GetAgentName() == "" && cfg.GetCacheStoragePath() == "" {
			cachePath := defaultCacheStoragePath
			util.CreateDirIfNotExist(cachePath)
			dbPath = cachePath + "/cache_test_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".db"
		} else {
			dbPath = c.cacheFilename
		}
	} else {
		// Use temporary file for cache during testing
		// These databases are not persisted across tests
		cachePath := defaultCacheStoragePath
		util.CreateDirIfNotExist(cachePath)
		dbPath = cachePath + "/cache_test_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".db"
	}
	c.dbPath = dbPath

	// Use shared database handle management to prevent file locking issues
	c.db, err = getSharedDB(dbPath)

	if err != nil {
		if c.isPersistedCacheEnabled {
			c.logger.WithError(err).Error("failed to open database, cache will not persist")
			c.isPersistedCacheLoaded = false
			c.isPersistedCacheEnabled = false
		} else {
			c.logger.WithError(err).Debug("failed to open temporary cache database")
		}
	} else {
		if c.db.IsReadOnly() {
			c.logger.WithField("dbFile", dbPath).Info("database opened in read-only mode")
		} else if c.isPersistedCacheEnabled {
			c.logger.WithField("dbFile", c.cacheFilename).Info("database opened successfully")
		} else {
			c.logger.Debug("temporary database created for cache (test mode)")
		}

		if c.isPersistedCacheEnabled {
			c.isPersistedCacheLoaded = true
		}
	}

	// Create bolt caches for each bucket
	bucketNames := []string{
		apiServicesKey,
		apiServiceInstancesKey,
		instanceCountKey,
		credReqDefKey,
		accReqDefKey,
		appProfDefKey,
		teamsKey,
		managedAppKey,
		subscriptionsKey,
		accReqKey,
		watchSequenceKey,
		watchResourceKey,
		complianceRuntimeKey,
	}

	if c.db != nil {
		if c.db.IsReadOnly() {
			for _, bucketName := range bucketNames {
				c.setBoltCache(&boltStore{db: c.db, bucketName: bucketName}, bucketName)
			}
			return
		}

		// Initialize all caches with bbolt
		for _, bucketName := range bucketNames {
			bc, err := newBoltStore(c.db, bucketName)
			if err != nil {
				c.logger.WithError(err).WithField("bucket", bucketName).Error("failed to create cache")
				continue
			}
			c.setBoltCache(bc, bucketName)
		}

		// Initialize special caches with default values for empty databases
		c.initializeSpecialCaches()

		// Check if database has any cached data
		// If empty, we need to sync from server
		if c.isPersistedCacheEnabled && !c.hasCachedData() {
			c.isPersistedCacheLoaded = false
			c.logger.Info("empty database detected, will sync from server")
		}
	} else {
		c.logger.Warn("database not available, caches will not be persisted")
		for _, bucketName := range bucketNames {
			c.setBoltCache(&boltStore{bucketName: bucketName}, bucketName)
		}
	}
}

// initializeSpecialCaches sets up loaders for special caches to ensure proper initialization
// of empty databases with required default structures
func (c *cacheManager) initializeSpecialCaches() {
	// Initialize instance count cache with an empty structure if needed
	// This ensures the cache is properly set up for counting API service instances
	instanceCountSetter := func(cache cache.Cache, key string) {
		// Setter is called when the loader is processed
		c.logger.WithField("cacheKey", key).Debug("instanceCount cache loader initialized")
	}
	_ = createInstanceCountLoader(instanceCountSetter, instanceCountKey)

	// Initialize sequence cache for watch sequences
	// This ensures sequence tracking starts properly for watch resources
	sequenceSetter := func(cache cache.Cache, key string) {
		// Setter is called when the loader is processed
		c.logger.WithField("cacheKey", key).Debug("sequence cache loader initialized")
	}
	_ = createSequenceLoader(sequenceSetter, watchSequenceKey)

	// Initialize team cache for platform teams
	// This ensures team metadata is properly structured
	teamSetter := func(cache cache.Cache, key string) {
		// Setter is called when the loader is processed
		c.logger.WithField("cacheKey", key).Debug("team cache loader initialized")
	}
	_ = createTeamLoader(teamSetter, teamsKey)
}

// hasCachedData checks if the database contains any cached data
// Returns true if data exists, false if database is empty
func (c *cacheManager) hasCachedData() bool {
	if c.db == nil {
		return false
	}

	hasData := false
	// Check if any critical buckets have data
	// We check the watch sequence as it's always set during initial sync
	err := c.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(watchSequenceKey))
		if b != nil {
			// Check if bucket has any keys
			cursor := b.Cursor()
			k, _ := cursor.First()
			if k != nil {
				hasData = true
			}
		}
		return nil
	})

	if err != nil {
		c.logger.WithError(err).Warn("failed to check for cached data")
		return false
	}

	return hasData
}

func (c *cacheManager) setBoltCache(bc *boltStore, key string) {
	c.logger.WithField("cacheKey", key).Debug("cache initialized")
	switch key {
	case apiServicesKey:
		c.apiMap = bc
	case apiServiceInstancesKey:
		c.instanceMap = bc
	case instanceCountKey:
		c.instanceCountMap = bc
	case credReqDefKey:
		c.crdMap = bc
	case accReqDefKey:
		c.ardMap = bc
	case appProfDefKey:
		c.apdMap = bc
	case teamsKey:
		c.teams = bc
	case managedAppKey:
		c.managedApplicationMap = bc
	case subscriptionsKey:
		c.subscriptionMap = bc
	case accReqKey:
		c.accessRequestMap = bc
	case watchResourceKey:
		c.watchResourceMap = bc
	case watchSequenceKey:
		c.sequenceCache = bc
	case complianceRuntimeKey:
		c.crrMap = bc
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
		return cachePath + "/" + cfg.GetAgentName() + ".db"
	}
	c.logger = c.logger.WithField("cachePath", cachePath)
	return cachePath + "/" + cfg.GetEnvironmentName() + ".db"
}

// HasLoadedPersistedCache - returns true if the caches are loaded from file
func (c *cacheManager) HasLoadedPersistedCache() bool {
	return c.isPersistedCacheLoaded
}

// Watch Sequence cache

// AddSequence - add/updates the sequenceID for the watch topic in cache
func (c *cacheManager) AddSequence(watchTopicName string, sequenceID int64) {
	c.sequenceCache.Set(watchTopicName, sequenceID)
}

// GetSequence - returns the sequenceID for the watch topic in cache
func (c *cacheManager) GetSequence(watchTopicName string) int64 {
	cachedSeqID, err := c.sequenceCache.Get(watchTopicName)
	if err == nil {
		if seqID, ok := cachedSeqID.(int64); ok {
			return seqID
		} else if seqID, ok := cachedSeqID.(int); ok {
			return int64(seqID)
		} else if seqID, ok := cachedSeqID.(int32); ok {
			return int64(seqID)
		} else if seqID, ok := cachedSeqID.(uint64); ok {
			return int64(seqID)
		} else if seqID, ok := cachedSeqID.(uint32); ok {
			return int64(seqID)
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

// Flush empties the persistent cache and all internal caches.
// In read-only mode, flush operations are silently skipped.
func (c *cacheManager) Flush() {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()
	c.logger.Debug("resetting the persistent cache")

	caches := []*boltStore{
		c.accessRequestMap,
		c.apiMap,
		c.ardMap,
		c.apdMap,
		c.crrMap,
		c.crdMap,
		c.instanceMap,
		c.managedApplicationMap,
		c.sequenceCache,
		c.subscriptionMap,
		c.watchResourceMap,
		c.teams,
		c.instanceCountMap,
	}

	for _, cache := range caches {
		if cache != nil {
			cache.Flush()
		}
	}
}

// Close closes the bbolt database and releases the shared database handle.
// If this is the last reference to the database, it will be closed and cleaned up.
func (c *cacheManager) Close() error {
	if c.dbPath == "" {
		return nil
	}

	c.logger.WithField("dbPath", c.dbPath).Debug("releasing database handle")
	return releaseSharedDB(c.dbPath)
}
