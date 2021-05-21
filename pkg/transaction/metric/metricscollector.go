package metric

import (
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/log"
	jwt "github.com/dgrijalva/jwt-go"
	metrics "github.com/rcrowley/go-metrics"
)

// Collector - interface for collecting metrics
type Collector interface {
	AddMetric(apiID, apiName, statusCode string, duration int64, appName, teamName string)
}

// collector - collects the metrics for transactions events
type collector struct {
	jobs.Job
	startTime    time.Time
	endTime      time.Time
	orgGUID      string
	eventChannel chan interface{}
	lock         *sync.Mutex
	registry     metrics.Registry
	metricMap    map[string]map[string]*APIMetric
	jobID        string
}

// NewMetricCollector - Create metric collector
func NewMetricCollector(eventChannel chan interface{}) Collector {
	metricCollector := &collector{
		// Set the initial start time to be minimum 1m behind, so that the job can generate valid event
		// if any usage event are to be generated on startup
		startTime:    time.Now().Add(-1 * time.Minute),
		lock:         &sync.Mutex{},
		registry:     metrics.NewRegistry(),
		metricMap:    make(map[string]map[string]*APIMetric),
		eventChannel: eventChannel,
	}

	var err error
	metricCollector.jobID, err = jobs.RegisterIntervalJob(metricCollector, agent.GetCentralConfig().GetEventAggregationInterval())
	if err != nil {
		panic(err)
	}

	return metricCollector
}

// Status - returns the status of the metric collector
func (c *collector) Status() error {
	return nil
}

// Ready - indicates that the collector job is ready to process
func (c *collector) Ready() bool {
	return true
}

// Execute - process the metric collection and generation of usage/metric event
func (c *collector) Execute() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.endTime = time.Now()
	c.orgGUID = c.getOrgGUID()
	log.Debugf("Generating usage/metric event [start timestamp: %d, end timestamp: %d]", convertTimeToMillis(c.startTime), convertTimeToMillis(c.endTime))
	defer func() {
		c.startTime = c.endTime
	}()

	c.generateEvents()
	return nil
}

func convertTimeToMillis(tm time.Time) int64 {
	return tm.UnixNano() / 1e6
}

// AddMetric - add metric for API transaction to collection
func (c *collector) AddMetric(apiID, apiName, statusCode string, duration int64, appName, teamName string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	transactionCount := c.getOrRegisterCounter("transaction.count")
	apiStatusDuration := c.getOrRegisterHistogram("transaction.status." + apiID + "." + statusCode)

	apiStatusMap, ok := c.metricMap[apiID]
	if !ok {
		apiStatusMap = make(map[string]*APIMetric)
		c.metricMap[apiID] = apiStatusMap
	}

	_, ok = apiStatusMap[statusCode]
	if !ok {
		apiStatusMap[statusCode] = &APIMetric{
			API: APIDetails{
				Name: apiName,
				ID:   apiID,
			},
			StatusCode: statusCode,
		}
	}

	transactionCount.Inc(1)
	apiStatusDuration.Update(duration)
}

func (c *collector) cleanup() {
	c.metricMap = make(map[string]map[string]*APIMetric)
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
	defer c.cleanup()
	if agent.GetCentralConfig().GetEnvironmentID() == "" ||
		cmd.GetBuildDataPlaneType() == "" {
		log.Warn("Unable to process usage and metric event generation. Please verify the agent config")
		return
	}
	if len(c.metricMap) != 0 {
		c.registry.Each(c.processUsageFromRegistry)
		// if agent.GetCentralConfig().CanPublishMetricEvent() {
		// 	counter := 0
		// 	for _, apiStatusMetricMap := range c.metricMap {
		// 		for _, apiStatusMetric := range apiStatusMetricMap {
		// 			c.generateAPIStatusMetricEvent(apiStatusMetric)
		// 			counter++
		// 		}
		// 	}
		// 	log.Infof("Generated %d metric events [start timestamp: %d, end timestamp: %d]", counter, convertTimeToMillis(c.startTime), convertTimeToMillis(c.endTime))
		// } else {
		// 	log.Info("Publishing the metric event is turned off")
		// }
	} else {
		log.Infof("No usage/metric event generated as no transactions recorded [start timestamp: %d, end timestamp: %d]", convertTimeToMillis(c.startTime), convertTimeToMillis(c.endTime))
	}

}

func (c *collector) generateUsageEvent(transactionCount int64, orgGUID string) {
	if transactionCount != 0 {
		c.generateLighthouseUsageEvent(transactionCount, orgGUID)
	}
}

func (c *collector) generateLighthouseUsageEvent(transactionCount int64, orgGUID string) {
	lightHouseUsageEvent := LighthouseUsageEvent{
		OrgGUID:     orgGUID,
		EnvID:       agent.GetCentralConfig().GetEnvironmentID(),
		Timestamp:   ISO8601Time(c.endTime),
		SchemaID:    agent.GetCentralConfig().GetLighthouseURL() + "/api/v1/report.schema.json",
		Granularity: int(c.endTime.Sub(c.startTime).Milliseconds()),
		Report: map[string]LighthouseUsageReport{
			c.endTime.Format("2006-01-02T15:04:05.000Z"): {
				Product: cmd.GetBuildDataPlaneType(),
				Usage: map[string]int64{
					cmd.GetBuildDataPlaneType() + ".Transactions": transactionCount,
				},
				Meta: make(map[string]interface{}),
			},
		},
		Meta: make(map[string]interface{}),
	}
	log.Infof("Creating usage event with %d transactions", transactionCount)
	c.eventChannel <- lightHouseUsageEvent
}

// func (c *collector) generateAPIStatusMetricEvent(apiStatusMetric *APIMetric) {
// 	apiStatusMetric.Observation.Start = convertTimeToMillis(c.startTime)
// 	apiStatusMetric.Observation.End = convertTimeToMillis(c.endTime)
// 	apiStatusMetricEventID, _ := uuid.NewV4()
// 	apiStatusMetricEvent := V4Event{
// 		ID:        apiStatusMetricEventID.String(),
// 		Timestamp: c.startTime.UnixNano() / 1e6,
// 		Event:     "api.transaction.status.metric",
// 		App:       c.orgGUID,
// 		Version:   "4",
// 		Distribution: V4EventDistribution{
// 			Environment: agent.GetCentralConfig().GetEnvironmentID(),
// 			Version:     "1",
// 		},
// 		Data: apiStatusMetric,
// 	}
// disabling sending metrics for now
// c.eventChannel <- apiStatusMetricEvent
// 	_ = apiStatusMetricEvent
// }

func (c *collector) processUsageFromRegistry(name string, metric interface{}) {
	if name == "transaction.count" {
		counterMetric := metric.(metrics.Counter)
		transactionCount := counterMetric.Count()
		counterMetric.Clear()
		if agent.GetCentralConfig().CanPublishUsageEvent() {
			c.generateUsageEvent(transactionCount, c.orgGUID)
			log.Infof("Generated usage events [start timestamp: %d, end timestamp: %d]", convertTimeToMillis(c.startTime), convertTimeToMillis(c.endTime))
		} else {
			log.Info("Publishing the usage event is turned off")
		}
	}

	// c.processTransactionMetric(name, metric)
}

// func (c *collector) processTransactionMetric(metricName string, metric interface{}) {
// 	elements := strings.Split(metricName, ".")
// 	if len(elements) > 2 {
// 		apiID := elements[2]
// 		apiStatusMap, ok := c.metricMap[apiID]
// 		if ok {
// 			if strings.HasPrefix(metricName, "transaction.status") {
// 				statusCode := elements[3]
// 				statusCodeDetail, ok := apiStatusMap[statusCode]
// 				if ok {
// 					statusMetric := (metric.(metrics.Histogram))
// 					c.setEventMetricsFromHistogram(statusCodeDetail, statusMetric)
// 					statusMetric.Clear()
// 				}
// 			}
// 		}
// 	}
// }

// func (c *collector) setEventMetricsFromHistogram(apiStatusDetails *APIMetric, histogram metrics.Histogram) {
// 	apiStatusDetails.Count = histogram.Count()
// 	apiStatusDetails.Response.Max = histogram.Max()
// 	apiStatusDetails.Response.Min = histogram.Min()
// 	apiStatusDetails.Response.Avg = histogram.Mean()
// }

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
