package stream

import (
	"fmt"
	"sync"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	cacheMapKey           = "CacheMap"
	apiService            = "APIService"
	apiServiceInstance    = "APIServiceInstance"
	category              = "Category"
	apiServiceKey         = "apiServiceInstanceKey"
	apiServiceInstanceKey = "apiServiceKey"
	categoryKey           = "categoryKey"
	cacheFileName         = "offline-stream-cache.json" //TODO: might not use
)

var cacheLock *sync.Mutex

func init() {
	cacheLock = &sync.Mutex{}
}

type centralCachesCache struct {
	jobs.Job
	apiServiceInstanceCache cache.Cache
	apiServiceCache         cache.Cache
	categoryCache           cache.Cache
}

func (j *centralCachesCache) Ready() bool {
	return true
}

func (j *centralCachesCache) Status() error {
	return nil
}

func (j *centralCachesCache) Execute() error {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	log.Trace("executing cache update job")

	cacheNames := []string{apiServiceKey, apiServiceInstanceKey, categoryKey}
	fmt.Println("cacheNames: ", cacheNames)

	// for _, cache := range cacheNames {
	// 	//TODO: add data
	// 	err := agent.cacheMap.Set(cache, "cache.ID")
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// // err := agent.cacheMap.Load("offlineCache" + ".cache")
	// err := agent.cacheMap.Load(cacheFileName)
	// if err != nil {
	// 	agent.cacheMap.SetWithSecondaryKey("cachesCache", "apiServiceInstancesKey", &agent.apiMap)
	// 	agent.cacheMap.SetWithSecondaryKey("cachesCache", "apiServicesKey", &agent.instanceMap)
	// 	agent.cacheMap.SetWithSecondaryKey("cachesCache", "categoriesKey", &agent.categoryMap)

	// 	if util.IsNotTest() {
	// 		agent.cacheMap.Save("offlineCache" + ".cache")
	// 	}
	// }
	//
	//
	// else {
	// 	fmt.Println("TODO: load caches into offlineCache")

	// 	apiCache, err := agent.cacheMap.GetBySecondaryKey("apiServiceInstancesKey")
	// 	if err != nil {
	// 		agent.apiMap = apiCache
	// 	}
	// 	fmt.Println("err: ", err)
	// 	fmt.Println("apiCache: ", apiCache)

	// 	instanceCache, err := agent.cacheMap.GetBySecondaryKey("apiServicesKey")
	// 	fmt.Println("err: ", err)
	// 	fmt.Println("instanceCache: ", instanceCache)
	// 	categoriesCache, err := agent.cacheMap.GetBySecondaryKey("categoriesKey")
	// 	fmt.Println("err: ", err)
	// 	fmt.Println("categoriesCache: ", categoriesCache)

	// 	if agent.apiMap == nil {
	// 		agent.apiMap = "rawr"
	// 	}
	// 	if agent.instanceMap == nil {
	// 		agent.instanceMap = "rawr"
	// 	}
	// 	if agent.categoryMap == nil {
	// 		agent.categoryMap = "rawr"
	// 	}
	// }

	return nil
}

// startCacheMapCacheJob -
func startCacheMapCacheJob(name string) (string, error) {
	job := &centralCachesCache{}
	return jobs.RegisterSingleRunJobWithName(job, name)
}

// StartCacheJob - starts a single run cache job
func StartCacheJob(c cache.Cache, action proto.Event_Type, resource *v1.ResourceInstance) error {
	if !util.IsNotTest() {
		return nil
	}

	var key, key2 string
	switch resource.Kind {
	case apiServiceInstance:
		key = apiServiceInstanceKey
		key2 = resource.Metadata.ID
		fmt.Println("key2: ", key2)
	case apiService:
		key = apiServiceKey
		key2 = resource.Name
		fmt.Println("key2: ", key2)
	case category:
		key = categoryKey
		key2 = resource.Metadata.ID // TODO: verify
		fmt.Println("key2: ", key2)
	default:
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		// return c.SetWithForeignKey(key, key2, resource)
		return c.SetWithSecondaryKey(key, key2, resource)
	}

	if action == proto.Event_DELETED {
		// return c.DeleteItemsByForeignKey(key2)
		return c.DeleteBySecondaryKey(key2)
	}

	id, err := startCacheMapCacheJob(cacheMapKey)
	if err != nil {
		return err
	}
	log.Tracef("registered cache job: %s", id)
	return nil
}
