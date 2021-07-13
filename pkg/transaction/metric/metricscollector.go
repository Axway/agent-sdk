package metric

import (
	"flag"
	"strings"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	metrics "github.com/rcrowley/go-metrics"
)

// Collector - interface for collecting metrics
type Collector interface {
	AddMetric(apiID, apiName, statusCode string, duration int64, appName, teamName string)
}

// collector - collects the metrics for transactions events
type collector struct {
	jobs.Job
	startTime        time.Time
	endTime          time.Time
	orgGUID          string
	eventChannel     chan interface{}
	lock             *sync.Mutex
	batchLock        *sync.Mutex
	registry         metrics.Registry
	metricBatch      *EventBatch
	metricMap        map[string]map[string]*APIMetric
	publishItemQueue []publishQueueItem
	jobID            string
	publisher        publisher
	storage          storageCache
}

type publishQueueItem interface {
	GetEvent() interface{}
	GetMetric() interface{}
}

type usageEventPublishItem interface {
	publishQueueItem
}

type usageEventQueueItem struct {
	usageEventPublishItem
	event  LighthouseUsageEvent
	metric metrics.Counter
}

func (qi *usageEventQueueItem) GetEvent() interface{} {
	return qi.event
}

func (qi *usageEventQueueItem) GetMetric() interface{} {
	return qi.metric
}

type metricEventPublishItem interface {
	publishQueueItem
	GetAPIID() string
	GetStatusCode() string
}

var globalMetricCollector Collector

// GetMetricCollector - Create metric collector
func GetMetricCollector() Collector {
	if globalMetricCollector == nil && flag.Lookup("test.v") == nil {
		globalMetricCollector = createMetricCollector()
	}
	return globalMetricCollector
}

func createMetricCollector() Collector {
	metricCollector := &collector{
		// Set the initial start time to be minimum 1m behind, so that the job can generate valid event
		// if any usage event are to be generated on startup
		startTime:        time.Now().Add(-1 * time.Minute),
		lock:             &sync.Mutex{},
		batchLock:        &sync.Mutex{},
		registry:         metrics.NewRegistry(),
		metricMap:        make(map[string]map[string]*APIMetric),
		publishItemQueue: make([]publishQueueItem, 0),
		publisher:        newMetricPublisher(),
	}

	// Create and initialize the storage cache for usage/metric by loading from disk
	metricCollector.storage = newStorageCache(metricCollector, traceability.GetDataDirPath()+"/"+cacheFileName)
	metricCollector.storage.initialize()

	if flag.Lookup("test.v") == nil {
		var err error
		metricCollector.jobID, err = jobs.RegisterIntervalJob(metricCollector, agent.GetCentralConfig().GetEventAggregationInterval())
		if err != nil {
			panic(err)
		}
	}

	return metricCollector
}

// Status - returns the status of the metric collector
func (c *collector) Status() error {
	return nil
}

// Ready - indicates that the collector job is ready to process
func (c *collector) Ready() bool {
	return agent.GetCentralConfig().GetEnvironmentID() != ""
}

// Execute - process the metric collection and generation of usage/metric event
func (c *collector) Execute() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.endTime = time.Now()
	c.orgGUID = c.getOrgGUID()
	log.Debugf("Generating usage/metric event [start timestamp: %d, end timestamp: %d]", util.ConvertTimeToMillis(c.startTime), util.ConvertTimeToMillis(c.endTime))
	defer func() {
		c.cleanup()
	}()

	c.generateEvents()
	c.publishEvents()
	return nil
}

// AddMetric - add metric for API transaction to collection
func (c *collector) AddMetric(apiID, apiName, statusCode string, duration int64, appName, teamName string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.batchLock.Lock()
	defer c.batchLock.Unlock()
	c.updateUsage(1)
	c.updateMetric(apiID, apiName, statusCode, duration)
}

func (c *collector) updateUsage(count int64) {
	transactionCount := c.getOrRegisterCounter("transaction.count")
	transactionCount.Inc(count)
	c.storage.updateUsage(int(transactionCount.Count()))
}

func (c *collector) updateMetric(apiID, apiName, statusCode string, duration int64) *APIMetric {
	if !agent.GetCentralConfig().CanPublishMetricEvent() {
		return nil // no need to update metrics with publish off
	}
	apiStatusDuration := c.getOrRegisterHistogram("transaction.status." + apiID + "." + statusCode)

	apiStatusMap, ok := c.metricMap[apiID]
	if !ok {
		apiStatusMap = make(map[string]*APIMetric)
		c.metricMap[apiID] = apiStatusMap
	}

	if _, ok := apiStatusMap[statusCode]; !ok {
		// First api metric for api+statuscode,
		// setup the start time to be used for reporting metric event
		apiStatusMap[statusCode] = &APIMetric{
			API: APIDetails{
				Name: apiName,
				ID:   apiID,
			},
			StatusCode: statusCode,
			StartTime:  time.Now(),
		}
	}

	apiStatusDuration.Update(duration)
	c.storage.updateMetric(apiStatusDuration, apiStatusMap[statusCode])
	return apiStatusMap[statusCode]
}

func (c *collector) cleanup() {
	c.publishItemQueue = make([]publishQueueItem, 0)
}

func (c *collector) getOrgGUID() string {
	authToken, _ := agent.GetCentralAuthToken()
	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = true

	claims := jwt.MapClaims{}
	_, _, _ = parser.ParseUnverified(authToken, claims)

	claim, ok := claims["org_guid"]
	if ok {
		return claim.(string)
	}
	return ""
}

func (c *collector) generateEvents() {
	if agent.GetCentralConfig().GetEnvironmentID() == "" ||
		cmd.GetBuildDataPlaneType() == "" {
		log.Warn("Unable to process usage and metric event generation. Please verify the agent config")
		return
	}

	if len(c.publishItemQueue) == 0 {
		log.Infof("No usage/metric event generated as no transactions recorded [start timestamp: %d, end timestamp: %d]", util.ConvertTimeToMillis(c.startTime), util.ConvertTimeToMillis(c.endTime))
	}

	c.metricBatch = NewEventBatch(c)
	c.registry.Each(c.processUsageFromRegistry)
	if agent.GetCentralConfig().CanPublishMetricEvent() {
		err := c.metricBatch.Publish()
		if err != nil {
			log.Errorf("Could not send metric event: %s, current metric data is kept and will be added to the next trigger interval.", err.Error())
		}
	}
}

func (c *collector) processUsageFromRegistry(name string, metric interface{}) {
	if name == "transaction.count" {
		counterMetric := metric.(metrics.Counter)
		if agent.GetCentralConfig().CanPublishUsageEvent() {
			c.generateUsageEvent(counterMetric, c.orgGUID)
		} else {
			log.Info("Publishing the usage event is turned off")
		}
	}

	c.processTransactionMetric(name, metric)
}

func (c *collector) generateUsageEvent(transactionCounter metrics.Counter, orgGUID string) {
	if transactionCounter.Count() != 0 {
		c.generateLighthouseUsageEvent(transactionCounter, orgGUID)
	}
}

func (c *collector) generateLighthouseUsageEvent(transactionCount metrics.Counter, orgGUID string) {
	lightHouseUsageEvent := LighthouseUsageEvent{
		OrgGUID:     orgGUID,
		EnvID:       agent.GetCentralConfig().GetEnvironmentID(),
		Timestamp:   ISO8601Time(c.endTime),
		SchemaID:    agent.GetCentralConfig().GetLighthouseURL() + "/api/v1/report.schema.json",
		Granularity: int(c.endTime.Sub(c.startTime).Milliseconds()),
		Report: map[string]LighthouseUsageReport{
			c.startTime.Format(ISO8601): {
				Product: cmd.GetBuildDataPlaneType(),
				Usage: map[string]int64{
					cmd.GetBuildDataPlaneType() + ".Transactions": transactionCount.Count(),
				},
				Meta: make(map[string]interface{}),
			},
		},
		Meta: make(map[string]interface{}),
	}
	log.Infof("Creating usage event with %d transactions [start timestamp: %d, end timestamp: %d]", transactionCount.Count(), util.ConvertTimeToMillis(c.startTime), util.ConvertTimeToMillis(c.endTime))
	queueItem := &usageEventQueueItem{
		event:  lightHouseUsageEvent,
		metric: transactionCount,
	}
	c.publishItemQueue = append(c.publishItemQueue, queueItem)
}

func (c *collector) processTransactionMetric(metricName string, metric interface{}) {
	elements := strings.Split(metricName, ".")
	if len(elements) > 2 {
		apiID := elements[2]
		if apiStatusMap, ok := c.metricMap[apiID]; ok && strings.HasPrefix(metricName, "transaction.status") {
			statusCode := elements[3]
			if statusCodeDetail, ok := apiStatusMap[statusCode]; ok {
				statusMetric := (metric.(metrics.Histogram))
				c.setEventMetricsFromHistogram(statusCodeDetail, statusMetric)
				c.generateAPIStatusMetricEvent(statusMetric, statusCodeDetail, apiID)
			}
		}
	}
}

func (c *collector) setEventMetricsFromHistogram(apiStatusDetails *APIMetric, histogram metrics.Histogram) {
	apiStatusDetails.Count = histogram.Count()
	apiStatusDetails.Response.Max = histogram.Max()
	apiStatusDetails.Response.Min = histogram.Min()
	apiStatusDetails.Response.Avg = histogram.Mean()
}

func (c *collector) generateAPIStatusMetricEvent(histogram metrics.Histogram, apiStatusMetric *APIMetric, apiID string) {
	if apiStatusMetric.Count == 0 {
		return
	}

	apiStatusMetric.Observation.Start = util.ConvertTimeToMillis(apiStatusMetric.StartTime)
	apiStatusMetric.Observation.End = util.ConvertTimeToMillis(c.endTime)
	apiStatusMetricEventID, _ := uuid.NewRandom()
	apiStatusMetricEvent := V4Event{
		ID:        apiStatusMetricEventID.String(),
		Timestamp: apiStatusMetric.StartTime.UnixNano() / 1e6,
		Event:     metricEvent,
		App:       c.orgGUID,
		Version:   "4",
		Distribution: &V4EventDistribution{
			Environment: agent.GetCentralConfig().GetEnvironmentID(),
			Version:     "1",
		},
		Data: apiStatusMetric,
	}

	// Add all metrics to the batch
	AddCondorMetricEventToBatch(apiStatusMetricEvent, c.metricBatch, histogram)
}

func (c *collector) getOrRegisterCounter(name string) metrics.Counter {
	counter := c.registry.Get(name)
	if counter == nil {
		counter = metrics.NewCounter()
		c.registry.Register(name, counter)
	}
	return counter.(metrics.Counter)
}

func (c *collector) getOrRegisterHistogram(name string) metrics.Histogram {
	histogram := c.registry.Get(name)
	if histogram == nil {
		sampler := metrics.NewUniformSample(2048)
		histogram = metrics.NewHistogram(sampler)
		c.registry.Register(name, histogram)
	}
	return histogram.(metrics.Histogram)
}

func (c *collector) publishEvents() {
	if len(c.publishItemQueue) > 0 {
		defer c.storage.save()

		for _, eventQueueItem := range c.publishItemQueue {
			err := c.publisher.publishEvent(eventQueueItem.GetEvent())
			if err != nil {
				log.Errorf("Failed to publish usage event  [start timestamp: %d, end timestamp: %d]: %s - current usage report is kept and will be added to the next trigger interval.", util.ConvertTimeToMillis(c.startTime), util.ConvertTimeToMillis(c.endTime), err.Error())
			} else {
				log.Infof("Published usage report [start timestamp: %d, end timestamp: %d]", util.ConvertTimeToMillis(c.startTime), util.ConvertTimeToMillis(c.endTime))
				c.cleanupCounters(eventQueueItem)
			}
		}
	}
}

func (c *collector) cleanupCounters(eventQueueItem publishQueueItem) {
	usageEventItem, ok := eventQueueItem.(usageEventPublishItem)
	if ok {
		c.cleanupUsageCounter(usageEventItem)
	}
}

func (c *collector) cleanupUsageCounter(usageEventItem usageEventPublishItem) {
	itemMetric := usageEventItem.GetMetric()
	counter, ok := itemMetric.(metrics.Counter)
	if ok {
		// Clean up the usage counter and reset the start time to current endTime
		counter.Clear()
		c.startTime = c.endTime
		c.storage.updateUsage(0)
	}
}

func (c *collector) cleanupMetricCounter(histogram metrics.Histogram, event V4Event) {
	// Clean up entry in api status metric map and histogram counter
	apiID := event.Data.API.ID
	if apiStatusMap, ok := c.metricMap[apiID]; ok {
		c.storage.removeMetric(apiStatusMap[event.Data.StatusCode])
		if len(apiStatusMap) != 0 {
			c.metricMap[apiID] = apiStatusMap
		} else {
			delete(c.metricMap, apiID)
		}
		histogram.Clear()
	}
	log.Infof("Published metrics report for API %s [start timestamp: %d, end timestamp: %d]", event.Data.API.Name, util.ConvertTimeToMillis(c.startTime), util.ConvertTimeToMillis(c.endTime))
}
