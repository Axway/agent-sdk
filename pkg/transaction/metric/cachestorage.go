package metric

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/util"
	metrics "github.com/rcrowley/go-metrics"
)

const (
	cacheFileName     = "agent-usagemetric.json"
	usageStartTimeKey = "usage_start_time"
	usageCountKey     = "usage_count"
	volumeKey         = "usage_volume"
	metricKeyPrefix   = "metric."
)

type storageCache interface {
	initialize()
	updateUsage(usageCount int)
	updateVolume(bytes int64)
	updateMetric(apiStatusMetric metrics.Histogram, apiMetric *APIMetric)
	removeMetric(apiMetric *APIMetric)
	save()
}

type cacheStorage struct {
	cacheFilePath    string
	oldCacheFilePath string
	collector        *collector
	storage          cache.Cache
	storageLock      sync.Mutex
	isInitialized    bool
}

func newStorageCache(collector *collector) storageCache {
	storageCache := &cacheStorage{
		cacheFilePath:    traceability.GetCacheDirPath() + "/" + cacheFileName,
		oldCacheFilePath: traceability.GetDataDirPath() + "/" + cacheFileName,
		collector:        collector,
		storageLock:      sync.Mutex{},
		storage:          cache.New(),
		isInitialized:    false,
	}

	return storageCache
}

func (c *cacheStorage) moveCacheFile() {
	// to remove for next major release
	_, err := os.Stat(c.oldCacheFilePath)
	if os.IsNotExist(err) {
		return
	}
	// file exists, move it over
	os.Rename(c.oldCacheFilePath, c.cacheFilePath)
}

func (c *cacheStorage) initialize() {
	c.moveCacheFile() // to remove for next major release
	storageCache := cache.Load(c.cacheFilePath)
	c.loadUsage(storageCache)
	c.loadAPIMetric(storageCache)

	// Not a job as the loop requires signal processing
	if !c.isInitialized && util.IsNotTest() {
		go c.storeCacheJob()
	}
	c.storage = storageCache
	c.isInitialized = true
}

func (c *cacheStorage) loadUsage(storageCache cache.Cache) {
	// update the collector start time
	usageStartTime, err := c.parseTimeFromCache(storageCache, usageStartTimeKey)
	if err == nil && !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		// do not load this start time when offline
		c.collector.startTime = usageStartTime
	}

	// update transaction counter in registry.
	usageCount, err := storageCache.Get(usageCountKey)
	if err == nil {
		// un-marshalling the cache defaults the serialization of numeric values to float64
		c.collector.updateUsage(int64(usageCount.(float64)))
	}

	// update transaction volume in registry.
	usageVolume, err := storageCache.Get(volumeKey)
	if err == nil {
		// un-marshalling the cache defaults the serialization of numeric values to float64
		c.collector.updateVolume(int64(usageVolume.(float64)))
	}
}

func (c *cacheStorage) updateUsage(usageCount int) {
	if !c.isInitialized || !agent.GetCentralConfig().GetUsageReportingConfig().CanPublishUsage() {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()
	c.storage.Set(usageStartTimeKey, c.collector.startTime)
	c.storage.Set(usageCountKey, usageCount)
}

func (c *cacheStorage) updateVolume(bytes int64) {
	if !c.isInitialized || !agent.GetCentralConfig().IsAxwayManaged() ||
		!agent.GetCentralConfig().GetUsageReportingConfig().CanPublishUsage() {
		// NOT initialized or NOT axway managed or can NOT publish usage
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()
	c.storage.Set(volumeKey, bytes)
}

func (c *cacheStorage) loadAPIMetric(storageCache cache.Cache) {
	cacheKeys := storageCache.GetKeys()
	for _, cacheKey := range cacheKeys {
		if strings.Contains(cacheKey, metricKeyPrefix) {
			cacheItem, _ := storageCache.Get(cacheKey)

			buffer, _ := json.Marshal(cacheItem)
			var apiMetric cachedMetric
			json.Unmarshal(buffer, &apiMetric)

			storageCache.Set(cacheKey, apiMetric)

			var apiStatusMetric *APIMetric
			for _, duration := range apiMetric.Values {
				apiStatusMetric = c.collector.updateMetric(APIDetails{apiMetric.API.ID, apiMetric.API.Name, apiMetric.API.Revision}, apiMetric.StatusCode, duration)
			}
			if apiStatusMetric != nil {
				apiStatusMetric.StartTime = apiMetric.StartTime
			}
		}
	}
}

func (c *cacheStorage) updateMetric(apiStatusMetric metrics.Histogram, apiMetric *APIMetric) {
	if !c.isInitialized {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()

	cachedAPIMetric := cachedMetric{
		API:        apiMetric.API,
		StatusCode: apiMetric.StatusCode,
		Count:      apiStatusMetric.Count(),
		Values:     apiStatusMetric.Sample().Values(),
		StartTime:  apiMetric.StartTime,
	}
	c.storage.Set(metricKeyPrefix+apiMetric.API.ID+"."+apiMetric.StatusCode, cachedAPIMetric)
}

func (c *cacheStorage) removeMetric(apiMetric *APIMetric) {
	if !c.isInitialized {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()
	c.storage.Delete(metricKeyPrefix + apiMetric.API.ID + "." + apiMetric.StatusCode)
}

func (c *cacheStorage) save() {
	if !c.isInitialized {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()

	c.storage.Save(c.cacheFilePath)
}

func (c *cacheStorage) storeCacheJob() {
	cachetimeTicker := time.NewTicker(5 * time.Second)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		select {
		case <-cachetimeTicker.C:
			c.save()
		case <-signals:
			c.save()
			break
		}
	}
}

func (c *cacheStorage) parseTimeFromCache(storage cache.Cache, key string) (time.Time, error) {
	resultTime := now()
	item, err := storage.Get(key)
	if err != nil {
		return now(), err
	}
	cachedTimeStr, ok := item.(string)
	if ok {
		resultTime, _ = time.Parse(time.RFC3339, cachedTimeStr)
	} else {
		cachedTime, ok := item.(time.Time)
		if ok {
			resultTime = cachedTime
		}
	}
	return resultTime, nil
}
