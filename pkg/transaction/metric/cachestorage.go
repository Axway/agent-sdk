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
	appUsagePrefix     = "app_usage."
	cacheFileName      = "agent-usagemetric.json"
	metricKeyPrefix    = "metric."
	metricStartTimeKey = "metric_start_time"
	usageStartTimeKey  = "usage_start_time"
	usageCountKey      = "usage_count"
	volumeKey          = "usage_volume"
)

type storageCache interface {
	initialize()
	updateUsage(usageCount int)
	updateVolume(bytes int64)
	updateAppUsage(usageCount int, appID string)
	updateMetric(apiStatusMetric metrics.Histogram, metric *APIMetric)
	removeMetric(metric *APIMetric)
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
	c.loadMetrics(storageCache)

	// Not a job as the loop requires signal processing
	if !c.isInitialized && util.IsNotTest() {
		go c.storeCacheJob()
	}
	c.storage = storageCache
	c.isInitialized = true
}

func (c *cacheStorage) loadUsage(storageCache cache.Cache) {
	// update the collector usage start time
	usageStartTime, err := c.parseTimeFromCache(storageCache, usageStartTimeKey)
	if err == nil && !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		// do not load this start time when offline
		c.collector.usageStartTime = usageStartTime
	}
	// update the collector metric start time
	metricStartTime, err := c.parseTimeFromCache(storageCache, metricStartTimeKey)
	if err == nil && !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		// do not load this start time when offline
		c.collector.metricStartTime = metricStartTime
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
	if !c.isInitialized || !agent.GetCentralConfig().GetUsageReportingConfig().CanPublish() {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()
	c.storage.Set(usageStartTimeKey, c.collector.usageStartTime)
	c.storage.Set(metricStartTimeKey, c.collector.metricStartTime)
	c.storage.Set(usageCountKey, usageCount)
}

func (c *cacheStorage) updateVolume(bytes int64) {
	if !c.isInitialized || !agent.GetCentralConfig().IsAxwayManaged() ||
		!agent.GetCentralConfig().GetUsageReportingConfig().CanPublish() {
		// NOT initialized or NOT axway managed or can NOT publish usage
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()
	c.storage.Set(volumeKey, bytes)
}

func (c *cacheStorage) updateAppUsage(usageCount int, appID string) {
	if !c.isInitialized || !agent.GetCentralConfig().GetUsageReportingConfig().CanPublish() {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()
	c.storage.Set(appUsagePrefix+appID, usageCount)
}

func (c *cacheStorage) loadMetrics(storageCache cache.Cache) {
	cacheKeys := storageCache.GetKeys()
	for _, cacheKey := range cacheKeys {
		if strings.Contains(cacheKey, metricKeyPrefix) {
			if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
				// delete metrics from cache in offline mode
				storageCache.Delete(cacheKey)
				continue
			}
			cacheItem, _ := storageCache.Get(cacheKey)

			buffer, _ := json.Marshal(cacheItem)
			var cm cachedMetric
			json.Unmarshal(buffer, &cm)

			var metric *APIMetric
			for _, duration := range cm.Values {
				metricDetail := Detail{
					APIDetails: cm.API,
					AppDetails: cm.App,
					StatusCode: cm.StatusCode,
					Duration:   duration,
				}
				metric = c.collector.createOrUpdateMetric(metricDetail)
			}

			newKey := c.getKey(metric)
			if newKey != cacheKey {
				c.storageLock.Lock()
				storageCache.Delete(cacheKey)
				c.storageLock.Unlock()
			}
			storageCache.Set(newKey, cm)
			if metric != nil {
				metric.StartTime = cm.StartTime
			}
		}
	}
}

func (c *cacheStorage) updateMetric(histogram metrics.Histogram, metric *APIMetric) {
	if !c.isInitialized {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()

	cachedMetric := cachedMetric{
		Subscription:  metric.Subscription,
		App:           metric.App,
		Product:       metric.Product,
		AssetResource: metric.AssetResource,
		ProductPlan:   metric.ProductPlan,
		Quota:         metric.Quota,
		API:           metric.API,
		StatusCode:    metric.StatusCode,
		Count:         histogram.Count(),
		Values:        histogram.Sample().Values(),
		StartTime:     metric.StartTime,
	}

	c.storage.Set(c.getKey(metric), cachedMetric)
}

func (c *cacheStorage) removeMetric(metric *APIMetric) {
	if !c.isInitialized {
		return
	}
	c.storageLock.Lock()
	defer c.storageLock.Unlock()

	c.storage.Delete(c.getKey(metric))
}

func (c *cacheStorage) getKey(metric *APIMetric) string {
	return metricKeyPrefix +
		metric.Subscription.ID + "." +
		metric.App.ID + "." +
		metric.API.ID + "." +
		metric.StatusCode
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
			return
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
