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
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	startTimestamp = "start-timestamp"
	endTimestamp   = "end-timestamp"
	eventType      = "event-type"
	usage          = "usage"
	metric         = "metric"
)

var ExitMetricInit = false

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
	publisher        *metricPublisher
	storage          storageCache
	reports          *cacheReport
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
			time.Sleep(time.Millisecond * 50)
			if ExitMetricInit {
				return
			}

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
	// There are beat params on execution that doesn't require central config to be instantiated
	if agent.GetCentralConfig() == nil {
		// if this is the case, check central config and if not instantiated, return nil
		return nil
	}

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
	metricCollector.reports = newReportCache()
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
	if c.usageConfig.IsOfflineMode() && !c.publisher.isReady() {
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
		WithField(startTimestamp, util.ConvertTimeToMillis(c.usageStartTime)).
		WithField(endTimestamp, util.ConvertTimeToMillis(c.usageEndTime)).
		WithField(eventType, usage).
		Debug("generating usage event")

	c.logger.
		WithField(startTimestamp, util.ConvertTimeToMillis(c.metricStartTime)).
		WithField(endTimestamp, util.ConvertTimeToMillis(c.metricEndTime)).
		WithField(eventType, metric).
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

	// Lookup Access Request and Managed App
	apiID := detail.APIDetails.ID

	// Go get the access request and managed app
	accessRequest, managedApp := c.getAccessRequestAndManagedApp(cacheManager, detail)

	// Update consumer details
	subRef := v1.Reference{
		ID:   unknown,
		Name: unknown,
	}
	if accessRequest != nil {
		accessReqSub := accessRequest.GetReferenceByGVK(catalog.SubscriptionGVK())
		if accessReqSub.ID != "" {
			subRef = accessReqSub
		}
	}

	subscriptionID := subRef.ID
	appDetail := c.createAppDetail(managedApp)
	appID := appDetail.ID

	statusCode := detail.StatusCode

	hAPIID := strings.ReplaceAll(apiID, ".", "#")
	histogram := c.getOrRegisterHistogram("consumer." + subscriptionID + "." + appID + "." + hAPIID + "." + statusCode)

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
			Subscription:  c.createSubscriptionDetail(subRef),
			App:           appDetail,
			Product:       c.getProduct(accessRequest, c.logger),
			API:           c.createAPIDetail(detail.APIDetails, accessRequest),
			AssetResource: c.getAssetResource(accessRequest, c.logger),
			ProductPlan:   c.getProductPlan(accessRequest, c.logger),
			Quota:         c.getQuota(accessRequest, c.logger),
			StatusCode:    statusCode,
			Status:        c.getStatusText(statusCode),
			StartTime:     now(),
		}
	}
	histogram.Update(detail.Duration)
	c.storage.updateMetric(histogram, statusMap[statusCode])

	return statusMap[statusCode]
}

// getAccessRequest -
func (c *collector) getAccessRequestAndManagedApp(cacheManager cache.Manager, detail Detail) (*management.AccessRequest, *v1.ResourceInstance) {

	c.logger.
		WithField("apiID", detail.APIDetails.ID).
		WithField("stage", detail.APIDetails.Stage).
		Trace("metric collector information")

	// get the managed application
	managedApp := cacheManager.GetManagedApplicationByName(detail.AppDetails.Name)
	if managedApp == nil {
		c.logger.
			WithField("appName", detail.AppDetails.Name).
			Debug("could not get managed application by name, return empty API metrics")
		return nil, nil
	}
	c.logger.
		WithField("appName", detail.AppDetails.Name).
		WithField("managed-app-name", managedApp.Name).
		Trace("managed application info")

	// get the access request
	accessRequest := transutil.GetAccessRequest(cacheManager, managedApp, detail.APIDetails.ID, detail.APIDetails.Stage, detail.APIDetails.Version)
	if accessRequest == nil {
		c.logger.
			Debug("could not get access request, return empty API metrics")
		return nil, nil
	}
	c.logger.
		WithField("managed-app-name", managedApp.Name).
		WithField("apiID", detail.APIDetails.ID).
		WithField("stage", detail.APIDetails.Stage).
		WithField("access-request-name", accessRequest.Name).
		Trace("managed application info")

	return accessRequest, managedApp
}

func (c *collector) createSubscriptionDetail(subRef v1.Reference) SubscriptionDetails {
	detail := SubscriptionDetails{
		ID:   unknown,
		Name: unknown,
	}

	if subRef.ID != "" && subRef.Name != "" {
		detail.ID = subRef.ID
		detail.Name = subRef.Name
	}
	return detail
}

func (c *collector) createAppDetail(app *v1.ResourceInstance) AppDetails {
	detail := AppDetails{
		ID:   unknown,
		Name: unknown,
	}

	if app != nil {
		detail.ID, detail.Name = c.getConsumerApplication(app)
		detail.ConsumerOrgID = c.getConsumerOrgID(app)
	}
	return detail
}

func (c *collector) createAPIDetail(api APIDetails, accessReq *management.AccessRequest) APIDetails {
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

func (c *collector) getAssetResource(accessRequest *management.AccessRequest, log log.FieldLogger) models.AssetResource {
	// Set default to provider details in case access request or managed apps comes back nil
	assetResource := models.AssetResource{
		ID:   unknown,
		Name: unknown,
	}

	if accessRequest == nil {
		log.Trace("access request is nil. Setting default values to unknown")
		return assetResource
	}

	assetResourceRef := accessRequest.GetReferenceByGVK(catalog.AssetResourceGVK())
	if assetResourceRef.ID == "" || assetResourceRef.Name == "" {
		log.Trace("could not get asset resource, setting asset resource to unknown")
	} else {
		assetResource.ID = assetResourceRef.ID
		assetResource.Name = assetResourceRef.Name
	}
	log.WithField("asset-resource-id", assetResource.ID).
		WithField("asset-resource-name", assetResource.Name).
		Trace("asset resource information")

	return assetResource
}

func (c *collector) getProduct(accessRequest *management.AccessRequest, log log.FieldLogger) models.Product {
	product := models.Product{
		ID:          unknown,
		Name:        unknown,
		VersionID:   unknown,
		VersionName: unknown,
	}

	if accessRequest == nil {
		log.Trace("access request is nil. Setting default values to unknown")
		return product
	}

	productRef := accessRequest.GetReferenceByGVK(catalog.ProductGVK())
	if productRef.ID == "" || productRef.Name == "" {
		log.Trace("could not get product information, setting product to unknown")
	} else {
		product.ID = productRef.ID
		product.Name = productRef.Name
	}

	productReleaseRef := accessRequest.GetReferenceByGVK(catalog.ProductReleaseGVK())
	if productReleaseRef.ID == "" || productReleaseRef.Name == "" {
		log.Trace("could not get product release information, setting product release to unknown")
	} else {
		product.VersionID = productReleaseRef.ID
		product.VersionName = productReleaseRef.Name
	}
	log.WithField("product-id", product.ID).
		WithField("product-name", product.Name).
		WithField("product-version-id", product.VersionID).
		WithField("product-version-name", product.VersionName).
		Trace("product information")
	return product

}

func (c *collector) getProductPlan(accessRequest *management.AccessRequest, log log.FieldLogger) models.ProductPlan {
	productPlan := models.ProductPlan{
		ID: unknown,
	}

	if accessRequest == nil {
		log.Trace("access request is nil. Setting default values to unknown")
		return productPlan
	}

	productPlanRef := accessRequest.GetReferenceByGVK(catalog.ProductPlanGVK())
	if productPlanRef.ID == "" {
		log.Debug("could not get product plan ID, setting product plan to unknown")
	} else {
		productPlan.ID = productPlanRef.ID
	}
	log.WithField("product-plan-id", productPlan.ID).
		Trace("product plan ID information")

	return productPlan
}

func (c *collector) getQuota(accessRequest *management.AccessRequest, log log.FieldLogger) models.Quota {
	quota := models.Quota{
		ID: unknown,
	}
	if accessRequest == nil {
		log.Trace("access request or managed app is nil. Setting default values to unknown")
		return quota
	}
	quotaRef := accessRequest.GetReferenceByGVK(catalog.QuotaGVK())
	if quotaRef.ID == "" {
		log.Debug("could not get quota ID, setting quota to unknown")
	} else {
		quota.ID = quotaRef.ID
	}
	log.WithField("quota-id", quota.ID).
		Trace("quota ID information")

	return quota
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
			WithField(startTimestamp, util.ConvertTimeToMillis(c.usageStartTime)).
			WithField(endTimestamp, util.ConvertTimeToMillis(c.usageEndTime)).
			WithField(eventType, usage).
			Info("no usage event generated as no transactions recorded")

		c.logger.
			WithField(startTimestamp, util.ConvertTimeToMillis(c.metricStartTime)).
			WithField(endTimestamp, util.ConvertTimeToMillis(c.metricEndTime)).
			WithField(eventType, metric).
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
		WithField(startTimestamp, util.ConvertTimeToMillis(c.usageStartTime)).
		WithField(endTimestamp, util.ConvertTimeToMillis(c.usageEndTime)).
		WithField("count", c.getOrRegisterCounter(transactionCountMetric).Count()).
		WithField(eventType, usage).
		Info("creating usage event")

	if agent.GetCentralConfig().IsAxwayManaged() {
		usage[fmt.Sprintf("%s.%s", cmd.GetBuildDataPlaneType(), lighthouseVolume)] = c.getOrRegisterCounter(transactionVolumeMetric).Count()
		c.logger.
			WithField(eventType, "volume").
			WithField("total-bytes", c.getOrRegisterCounter(transactionVolumeMetric).Count()).
			WithField(startTimestamp, util.ConvertTimeToMillis(c.usageStartTime)).
			WithField(endTimestamp, util.ConvertTimeToMillis(c.usageEndTime)).
			Infof("creating volume event")
	}

	granularity := c.usageConfig.GetReportGranularity()
	reportTime := c.usageStartTime.Format(ISO8601)
	if c.usageConfig.IsOfflineMode() {
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
		apiID := strings.ReplaceAll(elements[3], "#", ".")
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
					WithField(startTimestamp, util.ConvertTimeToMillis(c.usageStartTime)).
					WithField(endTimestamp, util.ConvertTimeToMillis(c.usageEndTime)).
					WithField(eventType, usage).
					Error("failed to publish usage event. current usage report is kept and will be added to the next trigger interval")
			} else {
				c.logger.
					WithField(startTimestamp, util.ConvertTimeToMillis(c.usageStartTime)).
					WithField(endTimestamp, util.ConvertTimeToMillis(c.usageEndTime)).
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
			WithField(startTimestamp, util.ConvertTimeToMillis(c.usageStartTime)).
			WithField(endTimestamp, util.ConvertTimeToMillis(c.usageEndTime)).
			WithField("api-name", metric.API.Name).
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

func (c *collector) getConsumerOrgID(ri *v1.ResourceInstance) string {
	if ri == nil {
		return ""
	}

	// Lookup Subscription
	app := &management.ManagedApplication{}
	app.FromInstance(ri)

	if app.Marketplace.Resource.Owner != nil {
		return app.Marketplace.Resource.Owner.Organization.ID
	}
	return ""
}

func (c *collector) getConsumerApplication(ri *v1.ResourceInstance) (string, string) {
	if ri == nil {
		return "", ""
	}

	for _, ref := range ri.Metadata.References {
		// get the ID of the Catalog Application
		if ref.Kind == catalog.ApplicationGVK().Kind {
			return ref.ID, ref.Name
		}
	}

	return ri.Metadata.ID, ri.Name // default to the managed app id
}
