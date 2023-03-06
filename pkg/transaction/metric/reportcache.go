package metric

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability"
)

const (
	lighthouseEventsKey       = "lighthouse_events"
	offlineCacheFileName      = "agent-report-working.json"
	offlineReportSuffix       = "usage_report.json"
	offlineReportDateFormat   = "2006_01_02"
	qaOfflineReportDateFormat = "2006_01_02_15_04"
)

type cacheReport struct {
	jobs.Job
	cacheFilePath           string
	reportCache             cache.Cache
	reportCacheLock         sync.Mutex
	isInitialized           bool
	offlineReportDateFormat string
	offline                 bool
}

func newReportCache() *cacheReport {
	reportManager := &cacheReport{
		cacheFilePath:           traceability.GetCacheDirPath() + "/" + offlineCacheFileName,
		reportCacheLock:         sync.Mutex{},
		reportCache:             cache.New(),
		isInitialized:           false,
		offline:                 agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode(),
		offlineReportDateFormat: offlineReportDateFormat,
	}
	if agent.GetCentralConfig().GetUsageReportingConfig().UsingQAVars() {
		reportManager.offlineReportDateFormat = qaOfflineReportDateFormat
	}

	reportManager.initialize()
	return reportManager
}

func (c *cacheReport) initialize() {
	reportCache := cache.Load(c.cacheFilePath)
	c.reportCache = reportCache
	c.isInitialized = true
}

// getEvents - gets the events from the cache, lock before calling this
func (c *cacheReport) getEvents() LighthouseUsageEvent {
	var savedLighthouseEvents LighthouseUsageEvent

	savedEventString, err := c.reportCache.Get(lighthouseEventsKey)
	if err != nil {
		return LighthouseUsageEvent{Report: map[string]LighthouseUsageReport{}}
	}

	err = json.Unmarshal([]byte(savedEventString.(string)), &savedLighthouseEvents)
	if err != nil {
		return LighthouseUsageEvent{Report: map[string]LighthouseUsageReport{}}
	}
	return savedLighthouseEvents
}

// loadEvents - locks the cache before getting the events
func (c *cacheReport) loadEvents() LighthouseUsageEvent {
	if !agent.GetCentralConfig().GetUsageReportingConfig().CanPublishUsage() {
		return LighthouseUsageEvent{Report: map[string]LighthouseUsageReport{}}
	}
	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	return c.getEvents()
}

// setEvents - sets the events in the cache and saves the cache to the disk, lock the cache before calling this
func (c *cacheReport) setEvents(lighthouseEvent LighthouseUsageEvent) {
	eventBytes, err := json.Marshal(lighthouseEvent)
	if err != nil {
		return
	}
	c.reportCache.Set(lighthouseEventsKey, string(eventBytes))
	c.reportCache.Save(c.cacheFilePath)
}

// updateEvents - locks the cache before setting the new light house events in the cache
func (c *cacheReport) updateEvents(lighthouseEvent LighthouseUsageEvent) {
	if !c.isInitialized || !agent.GetCentralConfig().GetUsageReportingConfig().CanPublishUsage() {
		return
	}

	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	c.setEvents(lighthouseEvent)
}

func (c *cacheReport) generateReportPath(timestamp ISO8601Time, index int) string {
	format := "%s_%s"
	if index != 0 {
		format = "%s_" + strconv.Itoa(index) + "_%s"
	}
	return path.Join(traceability.GetReportsDirPath(), fmt.Sprintf(format, time.Time(timestamp).Format(c.offlineReportDateFormat), offlineReportSuffix))
}

// validateReport - copies usage events setting all usages to 0 for any missing time interval
func (c *cacheReport) validateReport(savedEvents LighthouseUsageEvent) LighthouseUsageEvent {
	reportDuration := time.Duration(savedEvents.Granularity * int(time.Millisecond))

	// order all the keys, this will be used to find any missing times
	orderedKeys := make([]string, 0, len(savedEvents.Report))
	for k := range savedEvents.Report {
		orderedKeys = append(orderedKeys, k)
	}
	sort.Strings(orderedKeys)

	// create an empty report to insert when necessary
	emptyReport := LighthouseUsageReport{
		Product: savedEvents.Report[orderedKeys[0]].Product,
		Usage:   make(map[string]int64),
		Meta:    savedEvents.Report[orderedKeys[0]].Meta,
	}
	for usage := range savedEvents.Report[orderedKeys[0]].Usage {
		emptyReport.Usage[usage] = 0
	}

	curDate, _ := time.Parse(ISO8601, orderedKeys[0])
	lastDate, _ := time.Parse(ISO8601, orderedKeys[len(orderedKeys)-1])
	for curDate.Before(lastDate) {
		curDateString := curDate.Format(ISO8601)
		if _, exists := savedEvents.Report[curDateString]; !exists {
			savedEvents.Report[curDateString] = emptyReport
		}
		curDate = curDate.Add(reportDuration)
	}
	return savedEvents
}
