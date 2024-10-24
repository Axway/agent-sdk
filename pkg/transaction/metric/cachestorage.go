package metric

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/rcrowley/go-metrics"
)

const (
	appUsagePrefix     = "app_usage."
	cacheFileName      = "agent-usagemetric.json"
	metricKeyPrefix    = "metric"
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
	updateMetric(cachedMetric cachedMetricInterface, metric *centralMetric)
	removeMetric(metric *centralMetric)
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
	usageStartTime, err := parseTimeFromCache(storageCache, usageStartTimeKey)
	if err == nil && !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		// do not load this start time when offline
		c.collector.usageStartTime = usageStartTime
	}
	// update the collector metric start time
	metricStartTime, err := parseTimeFromCache(storageCache, metricStartTimeKey)
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
		if strings.HasPrefix(cacheKey, fmt.Sprintf("%s.", metricKeyPrefix)) {
			if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
				// delete metrics from cache in offline mode
				storageCache.Delete(cacheKey)
				continue
			}
			cacheItem, _ := storageCache.Get(cacheKey)

			buffer, _ := json.Marshal(cacheItem)
			var cm cachedMetric
			json.Unmarshal(buffer, &cm)

			var metric *centralMetric
			for _, duration := range cm.Values {
				metricDetail := Detail{
					StatusCode: cm.StatusCode,
					Duration:   duration,
				}
				if cm.API != nil {
					metricDetail.APIDetails = models.APIDetails{
						ID:   cm.API.ID,
						Name: cm.API.Name,
					}
				}
				if cm.App != nil {
					metricDetail.AppDetails = models.AppDetails{
						ID:            cm.App.ID,
						ConsumerOrgID: cm.App.ConsumerOrgID,
					}
				}
				if cm.Unit != nil {
					metricDetail.UnitName = cm.Unit.Name
				}
				metric = c.collector.createOrUpdateMetric(metricDetail)
			}

			newKey := metric.getKey()
			if newKey != cacheKey {
				c.storageLock.Lock()
				storageCache.Delete(cacheKey)
				c.storageLock.Unlock()
			}
			storageCache.Set(newKey, cm)
			if metric != nil {
				metric.Observation.Start = cm.StartTime.UnixMilli()
			}
		}
	}
}

func (c *cacheStorage) updateMetric(cached cachedMetricInterface, metric *centralMetric) {
	if !c.isInitialized {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()

	c.storage.Set(metric.getKey(), metric.createCachedMetric(cached))
}

func (c *cacheStorage) removeMetric(metric *centralMetric) {
	if !c.isInitialized {
		return
	}

	c.storageLock.Lock()
	defer c.storageLock.Unlock()

	c.storage.Delete(metric.getKey())
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

func parseTimeFromCache(storage cache.Cache, key string) (time.Time, error) {
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

type cachedMetricInterface interface {
	Count() int64
	Values() []int64
}

type customCounter struct {
	c metrics.Counter
}

func newCustomCounter(c metrics.Counter) *customCounter {
	return &customCounter{c: c}
}

func (c customCounter) Count() int64 {
	return c.c.Count()
}

func (c customCounter) Values() []int64 {
	return nil
}

type customHistogram struct {
	h metrics.Histogram
}

func newCustomHistogram(h metrics.Histogram) *customHistogram {
	return &customHistogram{h: h}
}

func (c customHistogram) Count() int64 {
	return c.h.Count()
}

func (c customHistogram) Values() []int64 {
	return c.h.Sample().Values()
}
