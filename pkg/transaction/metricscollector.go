package transaction

import (
	"strings"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	metrics "github.com/rcrowley/go-metrics"
)

// Metrics - struct to hold metrics for transaction
type Metrics struct {
	Count            int64   `json:"count"`
	MaxResponseTime  int64   `json:"maxResponseTime"`
	MinResponseTime  int64   `json:"minResponseTime"`
	MeanResponseTime float64 `json:"meanResponseTime"`
}

// StatusMetric - struct to hold metric specific for status code based transactions
type StatusMetric struct {
	StatusCode string `json:"statusCode"`
	Metrics
}

// APIMetric - struct to hold metric specific for api based transactions
type APIMetric struct {
	APIName       string `json:"apiName"`
	APIID         string `json:"apiID"`
	ObservedStart int64  `json:"observedStart,omitempty"`
	ObservedEnd   int64  `json:"observedEnd,omitempty"`
	Metrics
	StatusMetrics []*StatusMetric `json:"statusCodes,omitempty"`
}

// V4EventDistribution - represents V7 distribution
type V4EventDistribution struct {
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

// V4Event - represents V7 event
type V4Event struct {
	ID           string              `json:"id"`
	Timestamp    int64               `json:"timestamp"`
	Event        string              `json:"event"`
	App          string              `json:"app"` // ORG GUID
	Version      string              `json:"version"`
	Distribution V4EventDistribution `json:"distribution"`
	Data         interface{}         `json:"data"`
}

// collector - collects the metrics for transactions events
type collector struct {
	eventChannel       chan interface{}
	lock               *sync.Mutex
	registry           metrics.Registry
	apiMetricMap       map[string]*APIMetric
	apiStatusMetricMap map[string]map[string]*StatusMetric
}

var metricCollector *collector

// CreateMetricCollector - Create metric collector
func CreateMetricCollector(eventChannel chan interface{}) {
	metricCollector = &collector{
		lock:               &sync.Mutex{},
		registry:           metrics.NewRegistry(),
		apiMetricMap:       make(map[string]*APIMetric),
		apiStatusMetricMap: make(map[string]map[string]*StatusMetric),
		eventChannel:       eventChannel,
	}

	// go metrics.Log(metricCollector.registry, 5*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	go func() {
		freq := 30 * time.Second
		startTime := time.Now()
		for range time.Tick(freq) {
			metricCollector.lock.Lock()
			endTime := time.Now()
			metricCollector.generateAggregation(startTime, endTime)
			metricCollector.apiMetricMap = make(map[string]*APIMetric)
			metricCollector.apiStatusMetricMap = make(map[string]map[string]*StatusMetric)
			startTime = endTime
			metricCollector.lock.Unlock()
		}
	}()
}

func (c *collector) generateAggregation(startTime, endTime time.Time) {
	var transactionCount int64
	authToken, _ := agent.GetCentralAuthToken()
	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = true

	claims := jwt.MapClaims{}
	_, _, _ = parser.ParseUnverified(authToken, claims)
	orgGUID := ""
	claim, ok := claims["org_guid"]
	if ok {
		orgGUID = claim.(string)
	}

	if len(c.apiMetricMap) != 0 {
		metricCollector.registry.Each(func(name string, metric interface{}) {
			if name == "transaction.count" {
				counterMetric := metric.(metrics.Counter)
				transactionCount = counterMetric.Count()
				counterMetric.Clear()
			}
			elements := strings.Split(name, ".")
			if len(elements) > 2 {
				apiID := elements[2]
				apiDetail, ok := c.apiMetricMap[apiID]
				if ok {
					if strings.HasPrefix(name, "transaction.api") {
						apiMetric := (metric.(metrics.Histogram))
						c.setEventMetricsFromHistogram(&apiDetail.Metrics, apiMetric)
						apiMetric.Clear()
					}
					if strings.HasPrefix(name, "transaction.status") {
						statusCode := elements[3]
						apiStatusMap := c.apiStatusMetricMap[apiID]
						statusCodeDetail, ok := apiStatusMap[statusCode]
						if ok {
							statusMetric := (metric.(metrics.Histogram))
							c.setEventMetricsFromHistogram(&statusCodeDetail.Metrics, statusMetric)
							statusMetric.Clear()
						}
					}
				}
			}
		})

		for _, apiDetail := range c.apiMetricMap {
			// aggergationEvent.APIDetails = append(aggergationEvent.APIDetails, apiDetail)
			apiID := apiDetail.APIID
			apiStatusMap := c.apiStatusMetricMap[apiID]
			for _, statusDetail := range apiStatusMap {
				apiDetail.StatusMetrics = append(apiDetail.StatusMetrics, statusDetail)
			}
			detail := apiDetail
			detail.ObservedStart = startTime.UnixNano() / 1e6
			detail.ObservedEnd = endTime.UnixNano() / 1e6
			apiMetricEventID, _ := uuid.NewV4()
			apiMetricEvent := V4Event{
				ID:        apiMetricEventID.String(),
				Timestamp: startTime.UnixNano() / 1e6,
				Event:     "api.transaction.metric",
				App:       orgGUID,
				Version:   "4",
				Distribution: V4EventDistribution{
					Environment: agent.GetCentralConfig().GetEnvironmentName(),
					Version:     "1",
				},
				Data: detail,
			}
			c.eventChannel <- apiMetricEvent
		}
		// if aggergationEvent.TransactionCount != 0 {
		// 	c.eventChannel <- aggergationEvent
		// }
		if transactionCount != 0 {
			usageEventID, _ := uuid.NewV4()
			usageEvent := V4Event{
				ID:        usageEventID.String(),
				Timestamp: startTime.UnixNano() / 1e6,
				Event:     "usage." + agent.GetCentralConfig().GetEnvironmentName() + ".transactions",
				App:       orgGUID,
				Version:   "4",
				Distribution: V4EventDistribution{
					Environment: agent.GetCentralConfig().GetEnvironmentName(),
					Version:     "1",
				},
				Data: map[string]interface{}{
					"value":         transactionCount,
					"observedStart": startTime.UnixNano() / 1e6,
					"observedEnd":   endTime.UnixNano() / 1e6,
				},
			}
			c.eventChannel <- usageEvent
		}
	}

}

func (c *collector) setEventMetricsFromHistogram(eventmetric *Metrics, histogram metrics.Histogram) {
	eventmetric.Count = histogram.Count()
	eventmetric.MaxResponseTime = histogram.Max()
	eventmetric.MinResponseTime = histogram.Min()
	eventmetric.MeanResponseTime = histogram.Mean()
}

func (c *collector) collectMetric(apiID, apiName, statusCode string, duration int64, appName, teamName string) {
	// c.lock.Lock()
	// defer c.lock.Unlock()

	transactionCount := c.getOrRegisterCounter("transaction.count")

	apiDuration := c.getOrRegisterHistogram("transaction.api." + apiID)
	apiStatusDuration := c.getOrRegisterHistogram("transaction.status." + apiID + "." + statusCode)

	_, ok := c.apiMetricMap[apiID]
	if !ok {
		c.apiMetricMap[apiID] = &APIMetric{APIName: apiName, APIID: apiID}
	}

	apiStatusMap, ok := c.apiStatusMetricMap[apiID]
	if !ok {
		apiStatusMap = make(map[string]*StatusMetric)
		c.apiStatusMetricMap[apiID] = apiStatusMap
	}

	_, ok = apiStatusMap[statusCode]
	if !ok {
		apiStatusMap[statusCode] = &StatusMetric{StatusCode: statusCode}
	}

	transactionCount.Inc(1)
	apiDuration.Update(duration)
	apiStatusDuration.Update(duration)
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
