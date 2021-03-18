package metric

import (
	"strings"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/log"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
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
}

// NewMetricCollector - Create metric collector
func NewMetricCollector(eventChannel chan interface{}) Collector {
	metricCollector := &collector{
		startTime:    time.Now(),
		lock:         &sync.Mutex{},
		registry:     metrics.NewRegistry(),
		metricMap:    make(map[string]map[string]*APIMetric),
		eventChannel: eventChannel,
	}

	// go metrics.Log(metricCollector.registry, 5*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	_, err := jobs.RegisterIntervalJob(metricCollector, 30*time.Second)
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
	defer func() {
		c.startTime = c.endTime
	}()

	c.generateEvents()

	return nil
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
	if agent.GetCentralConfig().GetPlatformEnvironmentID() == "" ||
		agent.GetDataplaneType() == "" {
		log.Warn("Unable to process usage and metric event generation.")
		return
	}
	if len(c.metricMap) != 0 {
		c.registry.Each(c.processMetricFromRegistry)
		if agent.GetCentralConfig().CanPublishMetricEvent() {
			for _, apiStatusMetricMap := range c.metricMap {
				for _, apiStatusMetric := range apiStatusMetricMap {
					c.generateAPIStatusMetricEvent(apiStatusMetric)
				}
			}
		} else {
			log.Debug("Publishing the metric event is turned off")
		}
	}
}

func (c *collector) generateUsageEvent(transactionCount int64, orgGUID string) {
	if transactionCount != 0 {
		usageEventID, _ := uuid.NewV4()
		usageEvent := V4Event{
			ID:        usageEventID.String(),
			Timestamp: c.startTime.UnixNano() / 1e6,
			Event:     "usage." + agent.GetDataplaneType() + ".Transactions",
			App:       orgGUID,
			Version:   "4",
			Distribution: V4EventDistribution{
				Environment: agent.GetCentralConfig().GetPlatformEnvironmentID(),
				Version:     "1",
			},
			Data: map[string]interface{}{
				"value":         transactionCount,
				"observedStart": c.startTime.UnixNano() / 1e6,
				"observedEnd":   c.endTime.UnixNano() / 1e6,
				"governance":    "Customer Managed",
			},
		}
		c.eventChannel <- usageEvent
	}
}

func (c *collector) generateAPIStatusMetricEvent(apiStatusMetric *APIMetric) {
	apiStatusMetric.Observation.Start = c.startTime.UnixNano() / 1e6
	apiStatusMetric.Observation.End = c.endTime.UnixNano() / 1e6
	apiStatusMetricEventID, _ := uuid.NewV4()
	apiStatusMetricEvent := V4Event{
		ID:        apiStatusMetricEventID.String(),
		Timestamp: c.startTime.UnixNano() / 1e6,
		Event:     "api.transaction.status.metric",
		App:       c.orgGUID,
		Version:   "4",
		Distribution: V4EventDistribution{
			Environment: agent.GetCentralConfig().GetPlatformEnvironmentID(),
			Version:     "1",
		},
		Data: apiStatusMetric,
	}
	c.eventChannel <- apiStatusMetricEvent
}

func (c *collector) processMetricFromRegistry(name string, metric interface{}) {
	if name == "transaction.count" {
		counterMetric := metric.(metrics.Counter)
		transactionCount := counterMetric.Count()
		counterMetric.Clear()
		if agent.GetCentralConfig().CanPublishUsageEvent() {
			c.generateUsageEvent(transactionCount, c.orgGUID)
		} else {
			log.Debug("Publishing the usage event is turned off")
		}
	}

	c.processTransactionMetric(name, metric)
}

func (c *collector) processTransactionMetric(metricName string, metric interface{}) {
	elements := strings.Split(metricName, ".")
	if len(elements) > 2 {
		apiID := elements[2]
		apiStatusMap, ok := c.metricMap[apiID]
		if ok {
			if strings.HasPrefix(metricName, "transaction.status") {
				statusCode := elements[3]
				statusCodeDetail, ok := apiStatusMap[statusCode]
				if ok {
					statusMetric := (metric.(metrics.Histogram))
					c.setEventMetricsFromHistogram(statusCodeDetail, statusMetric)
					statusMetric.Clear()
				}
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
