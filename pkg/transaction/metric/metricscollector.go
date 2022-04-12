package metric

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/rcrowley/go-metrics"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
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
	usageStartTime   time.Time
	usageEndTime     time.Time
	metricStartTime  time.Time
	metricEndTime    time.Time
	orgGUID          string
	lock             *sync.Mutex
	batchLock        *sync.Mutex
	registry         metrics.Registry
	metricBatch      *EventBatch
	metricMap        map[string]map[string]map[string]map[string]*APIMetric
	publishItemQueue []publishQueueItem
	jobID            string
	publisher        publisher
	storage          storageCache
	reports          offlineReportCache
	usageConfig      config.UsageReportingConfig
	logger           log.FieldLogger
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
	logger := log.NewFieldLogger().
		WithPackage("sdk.transaction.metric").
		WithComponent("collector")
	metricCollector := &collector{
		// Set the initial start time to be minimum 1m behind, so that the job can generate valid event
		// if any usage event are to be generated on startup
		usageStartTime:   now().Add(-1 * time.Minute),
		metricStartTime:  now().Add(-1 * time.Minute),
		lock:             &sync.Mutex{},
		batchLock:        &sync.Mutex{},
		registry:         metrics.NewRegistry(),
		metricMap:        make(map[string]map[string]map[string]map[string]*APIMetric),
		publishItemQueue: make([]publishQueueItem, 0),
		usageConfig:      agent.GetCentralConfig().GetUsageReportingConfig(),
		logger:           logger,
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
	c.logger.
		WithField("startTimestamp", util.ConvertTimeToMillis(c.usageStartTime)).
		WithField("endTimestamp", util.ConvertTimeToMillis(c.usageEndTime)).
		WithField("eventType", "usage").
		Debug("generating usage event")

	c.logger.
		WithField("startTimestamp", util.ConvertTimeToMillis(c.metricStartTime)).
		WithField("endTimestamp", util.ConvertTimeToMillis(c.metricEndTime)).
		WithField("eventType", "metric").
		Debugf("generating metric event")
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
}

// AddMetricDetail - add metric for API transaction and consumer subscription to collection
func (c *collector) AddMetricDetail(metricDetail Detail) {
	c.AddMetric(metricDetail.APIDetails, metricDetail.StatusCode, metricDetail.Duration, metricDetail.Bytes, metricDetail.APIDetails.Name)
	c.updateMetric(metricDetail)
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

func (c *collector) updateMetric(detail Detail) *APIMetric {
	if !c.usageConfig.CanPublishMetric() {
		return nil // no need to update metrics with publish off
	}

	cacheManager := agent.GetCacheManager()

	// Lookup Managed App
	apiID := detail.APIDetails.ID
	stage := detail.APIDetails.Stage

	managedApp := cacheManager.GetManagedApplicationByName(detail.AppDetails.Name)
	accessRequest := c.getAccessRequest(cacheManager, managedApp, apiID, stage)
	subscription := c.getSubscription(cacheManager, accessRequest)

	subscriptionID := c.getSubscriptionID(subscription)
	appID := c.getApplicationID(managedApp)
	statusCode := detail.StatusCode

	histogram := c.getOrRegisterHistogram("consumer." + subscriptionID + "." + appID + "." + apiID + "." + statusCode)

	appMap, ok := c.metricMap[subscriptionID]
	if !ok {
		appMap = make(map[string]map[string]map[string]*APIMetric)
		c.metricMap[subscriptionID] = appMap
	}

	apiMap, ok := appMap[appID]
	if !ok {
		apiMap = make(map[string]map[string]*APIMetric)
		appMap[appID] = apiMap
	}

	statusMap, ok := apiMap[apiID]
	if !ok {
		statusMap = make(map[string]*APIMetric)
		apiMap[apiID] = statusMap
	}

	if _, ok := statusMap[statusCode]; !ok {
		// First api metric for sub+app+api+statuscode,
		// setup the start time to be used for reporting metric event
		statusMap[statusCode] = &APIMetric{
			Subscription: c.createSubscriptionDetail(subscription),
			App:          c.createAppDetail(managedApp),
			API:          c.createAPIDetail(detail.APIDetails, accessRequest),
			StatusCode:   statusCode,
			Status:       c.getStatusText(statusCode),
			StartTime:    now(),
		}
	}
	histogram.Update(detail.Duration)
	c.storage.updateMetric(histogram, statusMap[statusCode])

	return statusMap[statusCode]
}

func (c *collector) getSubscriptionID(subscription *v1.ResourceInstance) string {
	if subscription == nil {
		return unknown
	}
	return subscription.Metadata.ID
}

func (c *collector) createSubscriptionDetail(subscription *v1.ResourceInstance) SubscriptionDetails {
	detail := SubscriptionDetails{
		ID:   unknown,
		Name: unknown,
	}

	if subscription != nil {
		detail.ID = subscription.Metadata.ID
		detail.Name = subscription.Name
	}
	return detail
}

func (c *collector) getApplicationID(app *v1.ResourceInstance) string {
	if app == nil {
		return unknown
	}
	return app.Metadata.ID
}

func (c *collector) createAppDetail(app *v1.ResourceInstance) AppDetails {
	detail := AppDetails{
		ID:   unknown,
		Name: unknown,
	}

	if app != nil {
		detail.ID = app.Metadata.ID
		detail.Name = app.Name
		detail.ConsumerOrgID = c.getConsumerOrgID(app)
	}
	return detail
}

func (c *collector) createAPIDetail(api APIDetails, accessReq *v1alpha1.AccessRequest) APIDetails {
	detail := APIDetails{
		ID:                 api.ID,
		Name:               api.Name,
		Revision:           api.Revision,
		TeamID:             api.TeamID,
		APIServiceInstance: unknown,
	}

	if accessReq != nil {
		detail.APIServiceInstance = accessReq.Spec.ApiServiceInstance
	}
	return detail
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
	if agent.GetCentralConfig().GetEnvironmentID() == "" || cmd.GetBuildDataPlaneType() == "" {
		c.logger.Warn("Unable to process usage and metric event generation. Please verify the agent config")
		return
	}

	if len(c.publishItemQueue) == 0 {
		c.logger.
			WithField("startTimestamp", util.ConvertTimeToMillis(c.usageStartTime)).
			WithField("endTimestamp", util.ConvertTimeToMillis(c.usageEndTime)).
			WithField("eventType", "usage").
			Info("no usage event generated as no transactions recorded")

		c.logger.
			WithField("startTimestamp", util.ConvertTimeToMillis(c.metricStartTime)).
			WithField("endTimestamp", util.ConvertTimeToMillis(c.metricEndTime)).
			WithField("eventType", "metric").
			Info("no metric event generated as no transactions recorded")
	}

	c.metricBatch = NewEventBatch(c)
	c.registry.Each(c.processUsageFromRegistry)
	if c.usageConfig.CanPublishMetric() {
		err := c.metricBatch.Publish()
		if err != nil {
			c.logger.
				WithError(err).
				Errorf("could not send metric event. Current metric data is kept and will be added to the next trigger interval")
		}
	}
}

func (c *collector) processUsageFromRegistry(name string, metric interface{}) {
	switch {
	case name == transactionCountMetric:
		if c.usageConfig.CanPublishUsage() {
			c.generateUsageEvent(c.orgGUID)
		} else {
			c.logger.Info("Publishing the usage event is turned off")
		}

	// case transactionVolumeMetric:
	case name == transactionVolumeMetric:
		return // skip volume metric as it is handled with Count metric
	default:
		c.processMetric(name, metric)
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
	c.logger.
		WithField("startTimestamp", util.ConvertTimeToMillis(c.usageStartTime)).
		WithField("endTimestamp", util.ConvertTimeToMillis(c.usageEndTime)).
		WithField("count", c.getOrRegisterCounter(transactionCountMetric).Count()).
		WithField("eventType", "usage").
		Info("creating usage event")

	if agent.GetCentralConfig().IsAxwayManaged() {
		usage[fmt.Sprintf("%s.%s", cmd.GetBuildDataPlaneType(), lighthouseVolume)] = c.getOrRegisterCounter(transactionVolumeMetric).Count()
		c.logger.
			WithField("eventType", "volume").
			WithField("totalBytes", c.getOrRegisterCounter(transactionVolumeMetric).Count()).
			WithField("startTimestamp", util.ConvertTimeToMillis(c.usageStartTime)).
			WithField("endTimestamp", util.ConvertTimeToMillis(c.usageEndTime)).
			Infof("creating volume event")
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

func (c *collector) processMetric(metricName string, metric interface{}) {
	elements := strings.Split(metricName, ".")
	if len(elements) == 5 {
		subscriptionID := elements[1]
		appID := elements[2]
		apiID := elements[3]
		statusCode := elements[4]
		if appMap, ok := c.metricMap[subscriptionID]; ok {
			if apiMap, ok := appMap[appID]; ok {
				if statusMap, ok := apiMap[apiID]; ok {
					if statusDetail, ok := statusMap[statusCode]; ok {
						statusMetric := (metric.(metrics.Histogram))
						c.settMetricsFromHistogram(statusDetail, statusMetric)
						c.generateMetricEvent(statusMetric, statusDetail, appID)
					}
				}
			}
		}
	}
}

func (c *collector) settMetricsFromHistogram(metrics *APIMetric, histogram metrics.Histogram) {
	metrics.Count = histogram.Count()
	metrics.Response.Max = histogram.Max()
	metrics.Response.Min = histogram.Min()
	metrics.Response.Avg = histogram.Mean()
}

func (c *collector) generateMetricEvent(histogram metrics.Histogram, metric *APIMetric, apiID string) {
	if metric.Count == 0 {
		return
	}

	metric.Observation.Start = util.ConvertTimeToMillis(c.metricStartTime)
	metric.Observation.End = util.ConvertTimeToMillis(c.metricEndTime)
	// Generate app subscription metric
	c.generateV4Event(histogram, metric, c.orgGUID)

}

func (c *collector) generateV4Event(histogram metrics.Histogram, v4data V4Data, orgGUID string) {
	eventID, _ := uuid.NewRandom()
	event := V4Event{
		ID:        eventID.String(),
		Timestamp: c.metricStartTime.UnixNano() / 1e6,
		Event:     metricEvent,
		App:       orgGUID,
		Version:   "4",
		Distribution: &V4EventDistribution{
			Environment: agent.GetCentralConfig().GetEnvironmentID(),
			Version:     "1",
		},
		Data: v4data,
	}
	AddCondorMetricEventToBatch(event, c.metricBatch, histogram)
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
				c.logger.
					WithError(err).
					WithField("startTimestamp", util.ConvertTimeToMillis(c.usageStartTime)).
					WithField("endTimestamp", util.ConvertTimeToMillis(c.usageEndTime)).
					WithField("eventType", "usage").
					Error("failed to publish usage event. current usage report is kept and will be added to the next trigger interval")
			} else {
				c.logger.
					WithField("startTimestamp", util.ConvertTimeToMillis(c.usageStartTime)).
					WithField("endTimestamp", util.ConvertTimeToMillis(c.usageEndTime)).
					Info("published usage report")
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
	metric, ok := v4Data.(*APIMetric)
	if ok {
		subID := metric.Subscription.ID
		appID := metric.App.ID
		apiID := metric.API.ID
		statusCode := metric.StatusCode
		if consumerAppMap, ok := c.metricMap[subID]; ok {
			if apiMap, ok := consumerAppMap[appID]; ok {
				if apiStatusMap, ok := apiMap[apiID]; ok {
					c.storage.removeMetric(apiStatusMap[statusCode])
					delete(c.metricMap[subID][appID][apiID], statusCode)
					histogram.Clear()
				}
				if len(c.metricMap[subID][appID][apiID]) == 0 {
					delete(c.metricMap[subID][appID], apiID)
				}
			}
			if len(c.metricMap[subID][appID]) == 0 {
				delete(c.metricMap[subID], appID)
			}
		}
		if len(c.metricMap[subID]) == 0 {
			delete(c.metricMap, subID)
		}
		c.logger.
			WithField("startTimestamp", util.ConvertTimeToMillis(c.usageStartTime)).
			WithField("endTimestamp", util.ConvertTimeToMillis(c.usageEndTime)).
			WithField("apiName", metric.API.Name).
			Info("Published metrics report for API")
	}
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

func (c *collector) getConsumerOrgID(managedApp *v1.ResourceInstance) string {
	if managedApp == nil {
		return ""
	}

	// Lookup Subscription
	app := &v1alpha1.ManagedApplication{}
	app.FromInstance(managedApp)

	return app.Marketplace.Resource.Owner.Organization.Id
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

func (c *collector) getSubscription(cacheManager cache.Manager, accessRequest *v1alpha1.AccessRequest) *v1.ResourceInstance {
	subscriptionName := definitions.GetSubscriptionNameFromAccessRequest(accessRequest)
	if subscriptionName == "" {
		return nil
	}

	subscription := cacheManager.GetSubscriptionByName(subscriptionName)
	if subscription == nil {
		return nil
	}
	return subscription
}
