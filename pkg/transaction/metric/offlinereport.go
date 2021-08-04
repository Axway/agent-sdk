package metric

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gorhill/cronexpr"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability"
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
	initialize()
	updateOfflineEvents(lighthouseEvent LighthouseUsageEvent)
	loadOfflineEvents() (LighthouseUsageEvent, bool)
}

type cacheOfflineReport struct {
	jobs.Job
	cronExp         *cronexpr.Expression
	cacheFilePath   string
	reportCache     cache.Cache
	reportCacheLock sync.Mutex
	isInitialized   bool
	jobID           string
	dateFormat      string
}

func newOfflineReportCache() offlineReportCache {
	// Do not initialize if not needed
	if !agent.GetCentralConfig().GetEventAggregationOffline() {
		return nil
	}

	reportManager := &cacheOfflineReport{
		cacheFilePath:   traceability.GetCacheDirPath() + "/" + offlineCacheFileName,
		reportCacheLock: sync.Mutex{},
		reportCache:     cache.New(),
		isInitialized:   false,
		dateFormat:      offlineReportDateFormat,
	}

	reportManager.initialize()
	return reportManager
}

func (c *cacheOfflineReport) initialize() {
	reportCache := cache.Load(c.cacheFilePath)
	c.reportCache = reportCache
	c.isInitialized = true
	c.registerOfflineReportJob()
}

func (c *cacheOfflineReport) registerOfflineReportJob() {
	if flag.Lookup("test.v") != nil {
		return // skip setting up the job in test
	}

	// default schedule to run the report monthly
	schedule := "@monthly"

	// Add QA environment variable to allow to override this behavior
	if qaVar := os.Getenv("QA_CENTRAL_EVENTAGGREGATIONOFFLINE_SCHEDULE"); qaVar != "" {
		_, err := cronexpr.Parse(qaVar)
		if err != nil {
			log.Tracef("Could not use QA_CENTRAL_EVENTAGGREGATIONOFFLINE_SCHEDULE time, %s, it is not a proper schedule", qaVar)
		} else {
			log.Tracef("Using QA_CENTRAL_EVENTAGGREGATIONOFFLINE_SCHEDULE schedule, %s, rather than the monthly schedule for non-QA", qaVar)
			schedule = qaVar
			c.dateFormat = qaOfflineReportDateFormat // set the dateformat to use the hour and minute in QA
		}
	}

	// parse the cron string
	c.cronExp, _ = cronexpr.Parse(schedule)

	// start the job according to teh cron schedule
	jobID, err := jobs.RegisterScheduledJobWithName(c, schedule, "Offline Usage Report")
	if err != nil {
		log.Errorf("could not register usage report creation job: %s", err.Error())
	}
	c.jobID = jobID
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
	if !agent.GetCentralConfig().CanPublishUsageEvent() || !agent.GetCentralConfig().GetEventAggregationOffline() {
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
	if !c.isInitialized || !agent.GetCentralConfig().CanPublishUsageEvent() || !agent.GetCentralConfig().GetEventAggregationOffline() {
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

	notValid := false
	for !notValid {
		// First sort the reports by the keys
		orderedKeys := make([]string, 0, len(savedEvents.Report))
		for k := range savedEvents.Report {
			orderedKeys = append(orderedKeys, k)
		}
		sort.Strings(orderedKeys)

		notValid = true
		prevDateString := ""
		for _, dateString := range orderedKeys {
			if prevDateString == "" {
				prevDateString = dateString
				continue
			}
			prevTime, _ := time.Parse(ISO8601, prevDateString)
			curTime, _ := time.Parse(ISO8601, dateString)
			if curTime.Sub(prevTime) != reportDuration {
				missingTime := prevTime.Add(reportDuration)
				newReport := savedEvents.Report[dateString]
				for usage := range newReport.Usage {
					newReport.Usage[usage] = 0
				}
				savedEvents.Report[missingTime.Format(ISO8601)] = newReport
				notValid = false
				break
			} else {
				prevDateString = dateString
			}
		}
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
