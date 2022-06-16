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
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	lighthouseEventsKey       = "lighthouse_events"
	offlineCacheFileName      = "agent-report-working.json"
	offlineReportSuffix       = "usage_report.json"
	offlineReportDateFormat   = "2006_01_02"
	qaOfflineReportDateFormat = "2006_01_02_15_04"
)

type offlineReportCache interface {
	isReady() bool
	updateOfflineEvents(lighthouseEvent LighthouseUsageEvent)
	loadOfflineEvents() (LighthouseUsageEvent, bool)
}

type cacheOfflineReport struct {
	jobs.Job
	cacheFilePath   string
	reportCache     cache.Cache
	reportCacheLock sync.Mutex
	isInitialized   bool
	jobID           string
	dateFormat      string
	ready           bool
}

func newOfflineReportCache() offlineReportCache {
	// Do not initialize if not needed
	if !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		return nil
	}

	reportManager := &cacheOfflineReport{
		cacheFilePath:   traceability.GetCacheDirPath() + "/" + offlineCacheFileName,
		reportCacheLock: sync.Mutex{},
		reportCache:     cache.New(),
		isInitialized:   false,
		ready:           false,
		dateFormat:      offlineReportDateFormat,
	}
	if agent.GetCentralConfig().GetUsageReportingConfig().UsingQAVars() {
		reportManager.dateFormat = qaOfflineReportDateFormat
	}

	reportManager.initialize()
	return reportManager
}

func (c *cacheOfflineReport) isReady() bool {
	return c.ready
}

func (c *cacheOfflineReport) initialize() {
	reportCache := cache.Load(c.cacheFilePath)
	c.reportCache = reportCache
	c.registerOfflineReportJob()
	c.isInitialized = true
}

func (c *cacheOfflineReport) registerOfflineReportJob() {
	if !util.IsNotTest() {
		return // skip setting up the job in test
	}

	// start the job according to teh cron schedule
	var err error
	c.jobID, err = jobs.RegisterScheduledJobWithName(c, agent.GetCentralConfig().GetUsageReportingConfig().GetReportSchedule(), "Offline Usage Report")
	if err != nil {
		log.Errorf("could not register usage report creation job: %s", err.Error())
	}
}

// getOfflineEvents - gets the offline events from the cache, lock before calling this
func (c *cacheOfflineReport) getOfflineEvents() (LighthouseUsageEvent, bool) {
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

// loadOfflineEvents - locks the cache before getting the offline events
func (c *cacheOfflineReport) loadOfflineEvents() (LighthouseUsageEvent, bool) {
	if !agent.GetCentralConfig().GetUsageReportingConfig().CanPublishUsage() ||
		!agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		return LighthouseUsageEvent{}, false
	}
	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	return c.getOfflineEvents()
}

// setOfflineEvents - sets the offline events in the cache and saves the cache to the disk, lock the cache before calling this
func (c *cacheOfflineReport) setOfflineEvents(lighthouseEvent LighthouseUsageEvent) {
	eventBytes, err := json.Marshal(lighthouseEvent)
	if err != nil {
		return
	}
	c.reportCache.Set(lighthouseEventsKey, string(eventBytes))
	c.reportCache.Save(c.cacheFilePath)
}

// updateOfflineEvents - locks the cache before setting the new light house events in the cache
func (c *cacheOfflineReport) updateOfflineEvents(lighthouseEvent LighthouseUsageEvent) {
	if !c.isInitialized || !agent.GetCentralConfig().GetUsageReportingConfig().CanPublishUsage() || !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		return
	}

	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()

	c.setOfflineEvents(lighthouseEvent)
}

func (c *cacheOfflineReport) generateReportPath(timestamp ISO8601Time, index int) string {
	format := "%s_%s"
	if index != 0 {
		format = "%s_" + strconv.Itoa(index) + "_%s"
	}
	return path.Join(traceability.GetReportsDirPath(), fmt.Sprintf(format, time.Time(timestamp).Format(c.dateFormat), offlineReportSuffix))
}

// validateReport - copies usage events setting all usages to 0 for any missing time interval
func (c *cacheOfflineReport) validateReport(savedEvents LighthouseUsageEvent) LighthouseUsageEvent {
	reportDuration := time.Duration(savedEvents.Granularity * int(time.Millisecond))

	if len(savedEvents.Report) == 0 {
		return savedEvents
	}

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

// saveReport - creates a new file with the latest cached events then clears all reports from the cache, lock outside of this
func (c *cacheOfflineReport) saveReport() error {
	savedEvents, _ := c.getOfflineEvents()

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
	savedEvents.Report = make(map[string]LighthouseUsageReport)
	c.setOfflineEvents(savedEvents)
	return nil
}

// Status - returns an error if the status of the offline report job is in error
func (c *cacheOfflineReport) Status() error {
	return nil
}

// Ready - indicates that the offline report job is ready to process
//   additionally runs the initial report gen if the last trigger would
//   have ran but the agent was down
func (c *cacheOfflineReport) Ready() bool {
	if agent.GetCentralConfig().GetEnvironmentID() == "" {
		return false
	}

	defer func() { c.ready = true }() // once any existing reports are saved off this isReady

	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()
	savedEvents, loaded := c.getOfflineEvents()
	if loaded && len(savedEvents.Report) > 0 {
		// A report should have ran while agent was down
		err := c.saveReport()
		if err != nil {
			log.Errorf("error hit generating report, report still in cache: %s", err.Error())
		}
		return true
	}
	return true
}

// Execute - process the offline report generation
func (c *cacheOfflineReport) Execute() error {
	c.reportCacheLock.Lock()
	defer c.reportCacheLock.Unlock()
	return c.saveReport()
}
