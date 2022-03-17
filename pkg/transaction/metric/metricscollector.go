package metric

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Collector - interface for collecting metrics
type Collector interface {
	AddMetric(apiDetails APIDetails, statusCode string, duration, bytes int64, appName string)
	AddMetricDetail(metricDetail Detail)
}

// collector - collects the metrics for transactions events
type collector struct {
	jobs.Job
	usageStartTime    time.Time
	usageEndTime      time.Time
	metricStartTime   time.Time
	metricEndTime     time.Time
	orgGUID           string
	lock              *sync.Mutex
	batchLock         *sync.Mutex
	registry          metrics.Registry
	metricBatch       *EventBatch
	metricMap         map[string]map[string]*APIMetric
	consumerMetricMap map[string]map[string]map[string]map[string]*SubscriptionMetric
	publishItemQueue  []publishQueueItem
	jobID             string
	publisher         publisher
	storage           storageCache
	reports           offlineReportCache
	usageConfig       config.UsageReportingConfig
}

type publishQueueItem interface {
	GetEvent() interface{}
	GetUsageMetric() interface{}
	GetVolumeMetric() interface{}
}

type usageEventPublishItem interface {
	publishQueueItem
}

type usageEventQueueItem struct {
	usageEventPublishItem
	event        LighthouseUsageEvent
	usageMetric  metrics.Counter
	volumeMetric metrics.Counter
}

func init() {
	go func() {
		// Wait for the datadir to be set and exist
		dataDir := ""
		_, err := os.Stat(dataDir)
		for dataDir == "" || os.IsNotExist(err) {
			dataDir = traceability.GetDataDirPath()
			_, err = os.Stat(dataDir)
		}
		GetMetricCollector()
	}()
}

func (qi *usageEventQueueItem) GetEvent() interface{} {
	return qi.event
}

func (qi *usageEventQueueItem) GetUsageMetric() interface{} {
	return qi.usageMetric
}

func (qi *usageEventQueueItem) GetVolumeMetric() interface{} {
	return qi.volumeMetric
}

var globalMetricCollector Collector

// GetMetricCollector - Create metric collector
func GetMetricCollector() Collector {
	if globalMetricCollector == nil && util.IsNotTest() {
		globalMetricCollector = createMetricCollector()
	}
	return globalMetricCollector
}

func createMetricCollector() Collector {
	metricCollector := &collector{
		// Set the initial start time to be minimum 1m behind, so that the job can generate valid event
		// if any usage event are to be generated on startup
		usageStartTime:    now().Add(-1 * time.Minute),
		metricStartTime:   now().Add(-1 * time.Minute),
		lock:              &sync.Mutex{},
		batchLock:         &sync.Mutex{},
		registry:          metrics.NewRegistry(),
		metricMap:         make(map[string]map[string]*APIMetric),
		consumerMetricMap: make(map[string]map[string]map[string]map[string]*SubscriptionMetric),
		publishItemQueue:  make([]publishQueueItem, 0),
		usageConfig:       agent.GetCentralConfig().GetUsageReportingConfig(),
	}

	// Create and initialize the storage cache for usage/metric and offline report cache by loading from disk
	metricCollector.storage = newStorageCache(metricCollector)
	metricCollector.storage.initialize()
	metricCollector.reports = newOfflineReportCache()
	metricCollector.publisher = newMetricPublisher(metricCollector.storage, metricCollector.reports)

	if util.IsNotTest() {
		var err error
		if !metricCollector.usageConfig.IsOfflineMode() {
			metricCollector.jobID, err = jobs.RegisterIntervalJobWithName(metricCollector, metricCollector.usageConfig.GetInterval(), "Metric Collector")
		} else {
			metricCollector.jobID, err = jobs.RegisterScheduledJobWithName(metricCollector, metricCollector.usageConfig.GetSchedule(), "Metric Collector")
		}
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
	// Wait until any existing offline reports are saved
	if c.usageConfig.IsOfflineMode() && !c.reports.isReady() {
		return false
	}
	return agent.GetCentralConfig().GetEnvironmentID() != ""
}

// Execute - process the metric collection and generation of usage/metric event
func (c *collector) Execute() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.usageEndTime = now()
	c.metricEndTime = now()
	c.orgGUID = c.getOrgGUID()
	log.Debugf("Generating usage event [start timestamp: %d, end timestamp: %d]", util.ConvertTimeToMillis(c.usageStartTime), util.ConvertTimeToMillis(c.usageEndTime))
	log.Debugf("Generating metric event [start timestamp: %d, end timestamp: %d]", util.ConvertTimeToMillis(c.metricStartTime), util.ConvertTimeToMillis(c.metricEndTime))
	defer func() {
		c.cleanup()
	}()

	c.generateEvents()
	c.publishEvents()
	return nil
}

// AddMetric - add metric for API transaction to collection
func (c *collector) AddMetric(apiDetails APIDetails, statusCode string, duration, bytes int64, appName string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.batchLock.Lock()
	defer c.batchLock.Unlock()
	c.updateUsage(1)
	c.updateVolume(bytes)
	c.updateMetric(apiDetails, statusCode, duration)
}

// AddMetricDetail - add metric for API transaction and consumer subscription to collection
func (c *collector) AddMetricDetail(metricDetail Detail) {
	c.AddMetric(metricDetail.APIDetails, metricDetail.StatusCode, metricDetail.Duration, metricDetail.Bytes, metricDetail.APIDetails.Name)
	c.updateConsumerMetric(metricDetail)
}

func (c *collector) updateVolume(bytes int64) {
	if !agent.GetCentralConfig().IsAxwayManaged() {
		return // no need to update volume for customer managed
	}
	transactionVolume := c.getOrRegisterCounter(transactionVolumeMetric)
	transactionVolume.Inc(bytes)
	c.storage.updateVolume(transactionVolume.Count())
}

func (c *collector) updateUsage(count int64) {
	transactionCount := c.getOrRegisterCounter(transactionCountMetric)
	transactionCount.Inc(count)
	c.storage.updateUsage(int(transactionCount.Count()))
}

func (c *collector) updateConsumerMetric(metricAppDetail Detail) {
	if !c.usageConfig.CanPublishSubscriptionMetric() {
		return // no need to update metrics with publish off
	}

	if metricAppDetail.AppDetails.Name == "" {
		return
	}

	cacheManager := agent.GetCacheManager()

	// Lookup Managed App
	apiID := metricAppDetail.APIDetails.ID
	stage := metricAppDetail.APIDetails.Stage

	managedApp := cacheManager.GetManagedApplicationByName(metricAppDetail.AppDetails.Name)
	consumerOrgGUID := c.getConsumerOrgGUID(managedApp)
	accessRequest := c.getAccessRequest(cacheManager, managedApp, apiID, stage)
	subscription := c.getSubscription(cacheManager, accessRequest)

	appID := "unkown"
	appName := "unknown"
	if managedApp != nil {
		appID = managedApp.Metadata.ID
		appName = managedApp.Name
	}

	subscriptionID := "unknown"
	subscriptionName := "unknown"
	if subscription != nil {
		subscriptionID = subscription.Metadata.ID
		subscriptionName = subscription.Name
	}

	apiServiceInstanceName := "unknown"
	if accessRequest != nil {
		apiServiceInstanceName = accessRequest.Spec.ApiServiceInstance
	}

	statusCode := metricAppDetail.StatusCode
	consumerApp := c.getOrRegisterHistogram("subscription." + subscriptionID + "." + appID + "." + apiID + "." + statusCode)

	consumerAppMap, ok := c.consumerMetricMap[subscriptionID]
	if !ok {
		consumerAppMap = make(map[string]map[string]map[string]*SubscriptionMetric)
		c.consumerMetricMap[subscriptionID] = consumerAppMap
	}

	consumerAPIMap, ok := consumerAppMap[appID]
	if !ok {
		consumerAPIMap = make(map[string]map[string]*SubscriptionMetric)
		consumerAppMap[appID] = consumerAPIMap
	}

	consumerAPIStatusMap, ok := consumerAPIMap[apiID]
	if !ok {
		consumerAPIStatusMap = make(map[string]*SubscriptionMetric)
		consumerAPIMap[apiID] = consumerAPIStatusMap
	}

	if _, ok := consumerAPIStatusMap[statusCode]; !ok {
		// First api metric for api+statuscode,
		// setup the start time to be used for reporting metric event
		consumerAPIStatusMap[statusCode] = &SubscriptionMetric{
			Subscription: SubscriptionDetails{
				ID:                 subscriptionID,
				Name:               subscriptionName,
				AppID:              appID,
				AppName:            appName,
				APIID:              apiID,
				APIName:            metricAppDetail.APIDetails.Name,
				APIServiceInstance: apiServiceInstanceName,
				ConsumerOrgGUID:    consumerOrgGUID,
			},
			StatusCode: statusCode,
			Status:     c.getStatusText(statusCode),
			StartTime:  now(),
		}
	}

	consumerApp.Update(metricAppDetail.Duration)
}

func (c *collector) updateMetric(apiDetails APIDetails, statusCode string, duration int64) *APIMetric {
	if !c.usageConfig.CanPublishMetric() {
		return nil // no need to update metrics with publish off
	}
	apiStatusDuration := c.getOrRegisterHistogram("transaction.status." + apiDetails.ID + "." + statusCode)

	apiStatusMap, ok := c.metricMap[apiDetails.ID]
	if !ok {
		apiStatusMap = make(map[string]*APIMetric)
		c.metricMap[apiDetails.ID] = apiStatusMap
	}

	if _, ok := apiStatusMap[statusCode]; !ok {
		// First api metric for api+statuscode,
		// setup the start time to be used for reporting metric event
		apiStatusMap[statusCode] = &APIMetric{
			API:        apiDetails,
			StatusCode: statusCode,
			Status:     c.getStatusText(statusCode),
			StartTime:  now(),
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
		log.Infof("No usage event generated as no transactions recorded [start timestamp: %d, end timestamp: %d]", util.ConvertTimeToMillis(c.usageStartTime), util.ConvertTimeToMillis(c.usageEndTime))
		log.Infof("No metric event generated as no transactions recorded [start timestamp: %d, end timestamp: %d]", util.ConvertTimeToMillis(c.metricStartTime), util.ConvertTimeToMillis(c.metricEndTime))
	}

	c.metricBatch = NewEventBatch(c)
	c.registry.Each(c.processUsageFromRegistry)
	if c.usageConfig.CanPublishMetric() {
		err := c.metricBatch.Publish()
		if err != nil {
			log.Errorf("Could not send metric event: %s, current metric data is kept and will be added to the next trigger interval.", err.Error())
		}
	}
}

func (c *collector) processUsageFromRegistry(name string, metric interface{}) {
	switch {
	case name == transactionCountMetric:
		if c.usageConfig.CanPublishUsage() {
			c.generateUsageEvent(c.orgGUID)
		} else {
			log.Info("Publishing the usage event is turned off")
		}

	// case transactionVolumeMetric:
	case name == transactionVolumeMetric:
		return // skip volume metric as it is handled with Count metric
	// case transactionVolumeMetric:
	case strings.HasPrefix(name, "subscription."):
		c.processConsumerMetric(name, metric)
	default:
		c.processTransactionMetric(name, metric)
	}
}

func (c *collector) generateUsageEvent(orgGUID string) {
	if c.getOrRegisterCounter(transactionCountMetric).Count() != 0 || c.usageConfig.IsOfflineMode() {
		c.generateLighthouseUsageEvent(orgGUID)
	}
}

func (c *collector) generateLighthouseUsageEvent(orgGUID string) {
	usage := map[string]int64{
		fmt.Sprintf("%s.%s", cmd.GetBuildDataPlaneType(), lighthouseTransactions): c.getOrRegisterCounter(transactionCountMetric).Count(),
	}
	log.Infof("Creating usage event with %d transactions [start timestamp: %d, end timestamp: %d]", c.getOrRegisterCounter(transactionCountMetric).Count(), util.ConvertTimeToMillis(c.usageStartTime), util.ConvertTimeToMillis(c.usageEndTime))

	if agent.GetCentralConfig().IsAxwayManaged() {
		usage[fmt.Sprintf("%s.%s", cmd.GetBuildDataPlaneType(), lighthouseVolume)] = c.getOrRegisterCounter(transactionVolumeMetric).Count()
		log.Infof("Creating volume event with %d bytes [start timestamp: %d, end timestamp: %d]", c.getOrRegisterCounter(transactionVolumeMetric).Count(), util.ConvertTimeToMillis(c.usageStartTime), util.ConvertTimeToMillis(c.usageEndTime))
	}

	granularity := int(c.usageEndTime.Sub(c.usageStartTime).Milliseconds())
	reportTime := c.usageStartTime.Format(ISO8601)
	if c.usageConfig.IsOfflineMode() {
		granularity = c.usageConfig.GetReportGranularity()
		reportTime = c.usageEndTime.Add(time.Duration(-1*granularity) * time.Millisecond).Format(ISO8601)
	}

	lightHouseUsageEvent := LighthouseUsageEvent{
		OrgGUID:     orgGUID,
		EnvID:       agent.GetCentralConfig().GetEnvironmentID(),
		Timestamp:   ISO8601Time(c.usageEndTime),
		SchemaID:    c.usageConfig.GetURL() + schemaPath,
		Granularity: granularity,
		Report: map[string]LighthouseUsageReport{
			reportTime: {
				Product: cmd.GetBuildDataPlaneType(),
				Usage:   usage,
				Meta:    make(map[string]interface{}),
			},
		},
		Meta: map[string]interface{}{
			"AgentName":    agent.GetCentralConfig().GetAgentName(),
			"AgentVersion": cmd.BuildVersion,
		},
	}

	queueItem := &usageEventQueueItem{
		event:        lightHouseUsageEvent,
		usageMetric:  c.getOrRegisterCounter(transactionCountMetric),
		volumeMetric: c.getOrRegisterCounter(transactionVolumeMetric),
	}
	c.publishItemQueue = append(c.publishItemQueue, queueItem)
}

func (c *collector) processTransactionMetric(metricName string, metric interface{}) {
	elements := strings.Split(metricName, ".")
	if len(elements) > 2 {
		apiID := elements[2]
		statusCode := elements[3]
		if apiStatusMap, ok := c.metricMap[apiID]; ok {
			if statusCodeDetail, ok := apiStatusMap[statusCode]; ok && strings.HasPrefix(metricName, "transaction.status") {
				statusMetric := (metric.(metrics.Histogram))
				c.setEventMetricsFromHistogram(statusCodeDetail, statusMetric)
				c.generateAPIStatusMetricEvent(statusMetric, statusCodeDetail, apiID)
			}
		}
	}
}

func (c *collector) processConsumerMetric(metricName string, metric interface{}) {
	elements := strings.Split(metricName, ".")
	if len(elements) == 5 {
		subscriptionID := elements[1]
		appID := elements[2]
		apiID := elements[3]
		statusCode := elements[4]
		if consumerSubscriptionMap, ok := c.consumerMetricMap[subscriptionID]; ok {
			if consumerAppMap, ok := consumerSubscriptionMap[appID]; ok {
				if consumerAPIMap, ok := consumerAppMap[apiID]; ok {
					if consumerAPIStatusDetail, ok := consumerAPIMap[statusCode]; ok {
						statusMetric := (metric.(metrics.Histogram))
						c.setConsumerEventMetricsFromHistogram(consumerAPIStatusDetail, statusMetric)
						c.generateAppMetricEvent(statusMetric, consumerAPIStatusDetail, appID)
					}
				}
			}
		}
	}
}

func (c *collector) setConsumerEventMetricsFromHistogram(apiStatusDetails *SubscriptionMetric, histogram metrics.Histogram) {
	apiStatusDetails.Count = histogram.Count()
	apiStatusDetails.Response.Max = histogram.Max()
	apiStatusDetails.Response.Min = histogram.Min()
	apiStatusDetails.Response.Avg = histogram.Mean()
}

func (c *collector) generateAppMetricEvent(histogram metrics.Histogram, subscriptionStatusMetric *SubscriptionMetric, apiID string) {
	if subscriptionStatusMetric.Count == 0 {
		return
	}

	subscriptionStatusMetric.Observation.Start = util.ConvertTimeToMillis(c.metricStartTime)
	subscriptionStatusMetric.Observation.End = util.ConvertTimeToMillis(c.metricEndTime)
	// Generate app subscription metric for provider
	c.generateAppV4Event(histogram, subscriptionStatusMetric, c.orgGUID)

	// Generate app subscription metric for consumer
	if subscriptionStatusMetric.Subscription.ConsumerOrgGUID != "" && subscriptionStatusMetric.Subscription.ConsumerOrgGUID != c.orgGUID {
		c.generateAppV4Event(histogram, subscriptionStatusMetric, subscriptionStatusMetric.Subscription.ConsumerOrgGUID)
	}
}

func (c *collector) generateAppV4Event(histogram metrics.Histogram, v4data V4Data, orgGUID string) {
	subscriptionStatusMetricEventID, _ := uuid.NewRandom()
	subscriptionStatusMetricEvent := V4Event{
		ID:        subscriptionStatusMetricEventID.String(),
		Timestamp: c.metricStartTime.UnixNano() / 1e6,
		Event:     subscriptionMetricEvent,
		App:       orgGUID,
		Version:   "4",
		Distribution: &V4EventDistribution{
			Environment: agent.GetCentralConfig().GetEnvironmentID(),
			Version:     "1",
		},
		Data: v4data,
	}
	AddCondorMetricEventToBatch(subscriptionStatusMetricEvent, c.metricBatch, histogram)
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

	apiStatusMetric.Observation.Start = util.ConvertTimeToMillis(c.metricStartTime)
	apiStatusMetric.Observation.End = util.ConvertTimeToMillis(c.metricEndTime)
	apiStatusMetricEventID, _ := uuid.NewRandom()
	apiStatusMetricEvent := V4Event{
		ID:        apiStatusMetricEventID.String(),
		Timestamp: c.metricStartTime.UnixNano() / 1e6,
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
				log.Errorf("Failed to publish usage event  [start timestamp: %d, end timestamp: %d]: %s - current usage report is kept and will be added to the next trigger interval.", util.ConvertTimeToMillis(c.usageStartTime), util.ConvertTimeToMillis(c.usageEndTime), err.Error())
			} else {
				log.Infof("Published usage report [start timestamp: %d, end timestamp: %d]", util.ConvertTimeToMillis(c.usageStartTime), util.ConvertTimeToMillis(c.usageEndTime))
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
	itemUsageMetric := usageEventItem.GetUsageMetric()
	if usage, ok := itemUsageMetric.(metrics.Counter); ok {
		// Clean up the usage counter and reset the start time to current endTime
		usage.Clear()
		itemVolumeMetric := usageEventItem.GetVolumeMetric()
		if volume, ok := itemVolumeMetric.(metrics.Counter); ok {
			volume.Clear()
		}
		c.usageStartTime = c.usageEndTime
		c.storage.updateUsage(0)
		c.storage.updateVolume(0)
	}
}

func (c *collector) cleanupMetricCounter(histogram metrics.Histogram, v4Data V4Data) {
	switch v4Data.GetType() {
	case "APIMetric":
		apiMetric := v4Data.(*APIMetric)
		apiID := apiMetric.API.ID
		if apiStatusMap, ok := c.metricMap[apiID]; ok {
			c.storage.removeMetric(apiStatusMap[apiMetric.StatusCode])
			if len(apiStatusMap) != 0 {
				c.metricMap[apiID] = apiStatusMap
			} else {
				delete(c.metricMap, apiID)
			}
		}
		log.Infof("Published metrics report for API %s [start timestamp: %d, end timestamp: %d]", apiMetric.API.Name, util.ConvertTimeToMillis(c.usageStartTime), util.ConvertTimeToMillis(c.usageEndTime))
	case "SubscriptionMetric":
		subscriptionMetric := v4Data.(*SubscriptionMetric)
		subID := subscriptionMetric.Subscription.ID
		if consumerAppMap, ok := c.consumerMetricMap[subID]; ok {
			if len(consumerAppMap) != 0 {
				c.consumerMetricMap[subID] = consumerAppMap
			} else {
				delete(c.consumerMetricMap, subID)
			}
		}
		log.Infof("Published metrics report for subscription %s [start timestamp: %d, end timestamp: %d]", subscriptionMetric.Subscription.Name, util.ConvertTimeToMillis(c.usageStartTime), util.ConvertTimeToMillis(c.usageEndTime))
	}
	histogram.Clear()
}

func (c *collector) getStatusText(statusCode string) string {
	httpStatusCode, _ := strconv.Atoi(statusCode)
	statusText := "Unknown"
	switch {
	case httpStatusCode >= 200 && httpStatusCode < 400:
		statusText = "Success"
	case httpStatusCode >= 400 && httpStatusCode < 500:
		statusText = "Failure"
	case httpStatusCode >= 500 && httpStatusCode < 511:
		statusText = "Exception"
	}
	return statusText
}

func (c *collector) getConsumerOrgGUID(managedApp *v1.ResourceInstance) string {
	if managedApp == nil {
		return ""
	}

	// Lookup Subscription
	// TODO - Use subject subresource on managed application once model includes it.
	consumerOrgGUID, _ := util.GetSubResourcePropertyValue(managedApp,
		defs.XMarketplaceSubject, defs.AttrSubjectOrgGUID)
	return consumerOrgGUID
}

func (c *collector) getAccessRequest(cacheManager cache.Manager, managedApp *v1.ResourceInstance, apiID, stage string) *v1alpha1.AccessRequest {
	if managedApp == nil {
		return nil
	}

	// Lookup Access Request
	apiID = strings.TrimPrefix(apiID, "remoteApiId_")
	accessReq := cacheManager.GetAccessRequestByAppAndAPI(managedApp.Name, apiID, stage)
	return accessReq
}

func (c *collector) getSubscriptionFromAccessReq(accessRequest *v1alpha1.AccessRequest) string {
	// Lookup Access Request
	if accessRequest == nil {
		return ""
	}
	// Lookup Subscription
	// Temporary using custom subresource, use subscription reference in AccessRequest
	// TODO - Use subscription reference subresource on AccessRequest once controller starts to populate it.
	subscriptionName, _ := util.GetSubResourcePropertyValue(accessRequest,
		defs.XMarketplaceSubscription, defs.AttrSubscriptionName)
	return subscriptionName
}

func (c *collector) getSubscription(cacheManager cache.Manager, accessRequest *v1alpha1.AccessRequest) *v1.ResourceInstance {
	if accessRequest == nil {
		return nil
	}

	subscriptionName := c.getSubscriptionFromAccessReq(accessRequest)
	subscription := cacheManager.GetSubscriptionByName(subscriptionName)
	if subscription == nil {
		return nil
	}
	return subscription
}
