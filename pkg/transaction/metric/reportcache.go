package metric

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/gorhill/cronexpr"
)

const (
	eventsKey                 = "lighthouse_events"
	lastPublishTimestampKey   = "timestamp"
	offlineCacheFileName      = "agent-report-working.json"
	offlineReportSuffix       = "usage_report.json"
	offlineReportDateFormat   = "2006_01_02"
	qaOfflineReportDateFormat = "2006_01_02_15_04"
)

type currentTimeFunc func() time.Time

type usageReportCache struct {
	jobs.Job
	logger                  log.FieldLogger
	cacheFilePath           string
	reportCache             cache.Cache
	reportCacheLock         sync.Mutex
	isInitialized           bool
	offlineReportDateFormat string
	currTimeFunc            currentTimeFunc
}

func newReportCache() *usageReportCache {
	reportManager := &usageReportCache{
		logger:                  log.NewFieldLogger().WithPackage("metric").WithComponent("usageReportCache"),
		cacheFilePath:           traceability.GetCacheDirPath() + "/" + offlineCacheFileName,
		reportCacheLock:         sync.Mutex{},
		reportCache:             cache.New(),
		isInitialized:           false,
		offlineReportDateFormat: offlineReportDateFormat,
		currTimeFunc:            time.Now,
	}
	if agent.GetCentralConfig().GetUsageReportingConfig().UsingQAVars() {
		reportManager.offlineReportDateFormat = qaOfflineReportDateFormat
	}

	reportManager.initialize()
	return reportManager
}

func (c *usageReportCache) initialize() {
	reportCache := cache.Load(c.cacheFilePath)
	c.reportCache = reportCache
	c.isInitialized = true
}

// getEvents - gets the events from the cache, lock before calling this
func (c *usageReportCache) getEvents() UsageEvent {
	var savedLighthouseEvents UsageEvent

	savedEventString, err := c.reportCache.Get(eventsKey)
	if err != nil {
		return UsageEvent{Report: map[string]UsageReport{}}
	}

	err = json.Unmarshal([]byte(savedEventString.(string)), &savedLighthouseEvents)
	if err != nil {
		return UsageEvent{Report: map[string]UsageReport{}}
	}
	return savedLighthouseEvents
}

// loadEvents - locks the cache before getting the events
func (c *usageReportCache) loadEvents() UsageEvent {
	if !agent.GetCentralConfig().GetUsageReportingConfig().CanPublish() {
		return UsageEvent{Report: map[string]UsageReport{}}
	}
	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	return c.getEvents()
}

// setEvents - sets the events in the cache and saves the cache to the disk, lock the cache before calling this
func (c *usageReportCache) setEvents(lighthouseEvent UsageEvent) {
	eventBytes, err := json.Marshal(lighthouseEvent)
	if err != nil {
		return
	}
	c.reportCache.Set(eventsKey, string(eventBytes))
	c.reportCache.Save(c.cacheFilePath)
}

// updateEvents - locks the cache before setting the new light house events in the cache
func (c *usageReportCache) updateEvents(lighthouseEvent UsageEvent) {
	if !c.isInitialized || !agent.GetCentralConfig().GetUsageReportingConfig().CanPublish() {
		return
	}

	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	c.setEvents(lighthouseEvent)
}

func (c *usageReportCache) setLastPublishTimestamp(lastPublishTimestamp time.Time) {
	c.reportCache.Set(lastPublishTimestampKey, lastPublishTimestamp)
	c.reportCache.Save(c.cacheFilePath)
}

func (c *usageReportCache) getLastPublishTimestamp() time.Time {
	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	lastPublishTime, err := parseTimeFromCache(c.reportCache, lastPublishTimestampKey)
	if err != nil {
		return time.Time{}
	}

	return lastPublishTime
}

func (c *usageReportCache) generateReportPath(timestamp ISO8601Time, index int) string {
	format := "%s_%s"
	if index != 0 {
		format = "%s_" + strconv.Itoa(index) + "_%s"
	}
	return path.Join(traceability.GetReportsDirPath(), fmt.Sprintf(format, time.Time(timestamp).Format(c.offlineReportDateFormat), offlineReportSuffix))
}

// validateReport - copies usage events setting all usages to 0 for any missing time interval
func (c *usageReportCache) validateReport(savedEvents UsageEvent) UsageEvent {
	reportDuration := time.Duration(savedEvents.Granularity * int(time.Millisecond))

	// order all the keys, this will be used to find any missing times
	orderedKeys := make([]string, 0, len(savedEvents.Report))
	for k := range savedEvents.Report {
		orderedKeys = append(orderedKeys, k)
	}
	sort.Strings(orderedKeys)

	// create an empty report to insert when necessary
	emptyReport := UsageReport{
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

// addReport - adds a new report to the cache
func (c *usageReportCache) addReport(event UsageEvent) error {
	// Open and load the existing usage file
	savedEvents := c.loadEvents()

	for key, report := range event.Report {
		savedEvents.Report[key] = report
	}
	// Put all reports into the new event
	event.Report = savedEvents.Report

	// Update the cache
	c.updateEvents(event)

	return nil
}

// saveReport - creates a new file with the latest cached events then clears all reports from the cache, lock outside of this
func (c *usageReportCache) saveReport() error {
	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()
	savedEvents := c.getEvents()

	// no reports yet, skip creating the event
	if len(savedEvents.Report) == 0 {
		return nil
	}
	savedEvents = c.validateReport(savedEvents)

	// create the path to save the file
	outputFilePath := ""
	i := 0
	fileExists := true
	for fileExists {
		outputFilePath = c.generateReportPath(savedEvents.Timestamp, i)
		_, err := os.Stat(outputFilePath)
		i++
		fileExists = !os.IsNotExist(err)
	}

	// create the new file to save the events
	file, err := os.Create(filepath.Clean(outputFilePath))
	if err != nil {
		return err
	}

	// marshal the event into json bytes
	cacheBytes, err := json.Marshal(savedEvents)
	if err != nil {
		file.Close()
		return err
	}

	// save the bytes and close the file
	_, err = io.Copy(file, bytes.NewReader(cacheBytes))
	file.Close()
	if err != nil {
		return err
	}

	// clear out all reports
	savedEvents.Report = make(map[string]UsageReport)
	c.setEvents(savedEvents)
	return nil
}

// sendReport - creates a new report with the latest cached events then clears all reports from the cache, lock outside of this
func (c *usageReportCache) sendReport(publishFunc func(event UsageEvent) error) error {
	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()
	savedEvents := c.getEvents()

	// no reports yet, skip creating the event
	if len(savedEvents.Report) == 0 {
		return nil
	}
	savedEvents = c.validateReport(savedEvents)
	if err := publishFunc(savedEvents); err != nil {
		c.logger.Error("could not publish usage, will send at next scheduled publishing")
		return nil
	}

	// update the publish time
	lastPublishTime := time.Now()
	c.setLastPublishTimestamp(lastPublishTime)

	savedEvents.Report = make(map[string]UsageReport)
	c.setEvents(savedEvents)
	return nil
}

func (c *usageReportCache) shouldPublish(schedule string) bool {
	currentTime := c.currTimeFunc()
	lastPublishTimestamp := c.getLastPublishTimestamp()

	// if the last publish was made more than a day ago, publish
	elapsedTimeSinceLastPublish := currentTime.Sub(lastPublishTimestamp)
	if lastPublishTimestamp.IsZero() || elapsedTimeSinceLastPublish >= 24*time.Hour {
		return true
	}

	cronSchedule, err := cronexpr.Parse(schedule)
	if err != nil {
		return false
	}
	// publish if last scheduled time is past
	nextPublishTime := cronSchedule.Next(lastPublishTimestamp)
	return nextPublishTime.Before(currentTime)
}
