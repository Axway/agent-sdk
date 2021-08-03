package metric

import (
	"encoding/json"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/traceability"
)

const (
	lighthouseEventsKey  = "lighthouse_events"
	offlineCacheFileName = "agent-report-working.json"
)

type offlineReportCache interface {
	initialize()
	updateOfflineEvents(lighthouseEvent LighthouseUsageEvent)
	loadOfflineEvents() (LighthouseUsageEvent, bool)
}

type cacheOfflineReport struct {
	cacheFilePath   string
	reportCache     cache.Cache
	reportCacheLock sync.Mutex
	isInitialized   bool
}

func newOfflineReportCache() offlineReportCache {
	// Do not initialize if not needed
	if !agent.GetCentralConfig().GetEventAggregationOffline() {
		return nil
	}

	reportCache := &cacheOfflineReport{
		cacheFilePath:   traceability.GetCacheDirPath() + "/" + offlineCacheFileName,
		reportCacheLock: sync.Mutex{},
		reportCache:     cache.New(),
		isInitialized:   false,
	}

	reportCache.initialize()
	return reportCache
}

func (c *cacheOfflineReport) initialize() {
	reportCache := cache.Load(c.cacheFilePath)
	c.reportCache = reportCache
	c.isInitialized = true
}

func (c *cacheOfflineReport) loadOfflineEvents() (LighthouseUsageEvent, bool) {
	if !agent.GetCentralConfig().CanPublishUsageEvent() || !agent.GetCentralConfig().GetEventAggregationOffline() {
		return LighthouseUsageEvent{}, false
	}
	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	var savedLighthouseEvents LighthouseUsageEvent

	savedEventString, err := c.reportCache.Get(lighthouseEventsKey)
	if err != nil {
		return LighthouseUsageEvent{}, false
	}

	err = json.Unmarshal([]byte(savedEventString.(string)), &savedLighthouseEvents)
	if err != nil {
		return LighthouseUsageEvent{}, false
	}
	return savedLighthouseEvents, true
}

func (c *cacheOfflineReport) updateOfflineEvents(lighthouseEvent LighthouseUsageEvent) {
	if !c.isInitialized || !agent.GetCentralConfig().CanPublishUsageEvent() || !agent.GetCentralConfig().GetEventAggregationOffline() {
		return
	}

	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	eventBytes, err := json.Marshal(lighthouseEvent)
	if err != nil {
		return
	}
	c.reportCache.Set(lighthouseEventsKey, string(eventBytes))
	c.reportCache.Save(c.cacheFilePath)
}
