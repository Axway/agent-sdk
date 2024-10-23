package metric

import (
	"encoding/json"
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
	"github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	startTimestampStr = "start-timestamp"
	endTimestampStr   = "end-timestamp"
	eventTypeStr      = "event-type"
	usageStr          = "usage"
	metricStr         = "metric"
	volumeStr         = "volume"
	countStr          = "count"
)

var exitMetricInit = false
var exitMutex = &sync.RWMutex{}

func ExitMetricInit() {
	exitMutex.Lock()
	defer exitMutex.Unlock()
	exitMetricInit = true
}

// Collector - interface for collecting metrics
type Collector interface {
	InitializeBatch()
	AddMetric(apiDetails models.APIDetails, statusCode string, duration, bytes int64, appName string)
	AddCustomMetricDetail(metric CustomMetricDetail)
	AddMetricDetail(metricDetail Detail)
	AddAPIMetricDetail(metric MetricDetail)
	AddAPIMetric(apiMetric *APIMetric)
	Publish()
	ShutdownPublish()
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
	metricMap        map[string]map[string]map[string]map[string]*centralMetricEvent
	metricMapLock    *sync.Mutex
	publishItemQueue []publishQueueItem
	jobID            string
	usagePublisher   *usagePublisher
	storage          storageCache
	reports          *usageReportCache
	metricConfig     config.MetricReportingConfig
	usageConfig      config.UsageReportingConfig
	logger           log.FieldLogger
	metricLogger     log.FieldLogger
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
	event        UsageEvent
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
			exitMutex.RLock()
			if exitMetricInit {
				exitMutex.RUnlock()
				return
			}
			exitMutex.RUnlock()

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
		metricMapLock:    &sync.Mutex{},
		registry:         metrics.NewRegistry(),
		metricMap:        make(map[string]map[string]map[string]map[string]*centralMetricEvent),
		publishItemQueue: make([]publishQueueItem, 0),
		metricConfig:     agent.GetCentralConfig().GetMetricReportingConfig(),
		usageConfig:      agent.GetCentralConfig().GetUsageReportingConfig(),
		logger:           logger,
		metricLogger:     log.NewMetricFieldLogger(),
	}

	// Create and initialize the storage cache for usage/metric and offline report cache by loading from disk
	metricCollector.storage = newStorageCache(metricCollector)
	metricCollector.storage.initialize()
	metricCollector.reports = newReportCache()
	metricCollector.usagePublisher = newUsagePublisher(metricCollector.storage, metricCollector.reports)

	if util.IsNotTest() {
		var err error
		if !metricCollector.usageConfig.IsOfflineMode() {
			metricCollector.jobID, err = jobs.RegisterScheduledJobWithName(metricCollector, metricCollector.metricConfig.GetSchedule(), "Metric Collector")
		} else {
			metricCollector.jobID, err = jobs.RegisterScheduledJobWithName(metricCollector, metricCollector.usageConfig.GetOfflineSchedule(), "Metric Collector")
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
	if c.usageConfig.IsOfflineMode() && !c.usagePublisher.isReady() {
		return false
	}
	return agent.GetCentralConfig().GetEnvironmentID() != ""
}

// Execute - process the metric collection and generation of usage/metric event
func (c *collector) Execute() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.usagePublisher.offline && healthcheck.GetStatus(traceability.HealthCheckEndpoint) != healthcheck.OK {
		c.logger.Warn("traceability is not connected, can not publish metrics at this time")
		return nil
	}

	c.usageEndTime = now()
	c.metricEndTime = now()
	c.orgGUID = c.getOrgGUID()

	usageMsg := "updating working usage report file"
	if !c.usageConfig.IsOfflineMode() {
		usageMsg = "caching usage event"
		c.logger.
			WithField(startTimestampStr, util.ConvertTimeToMillis(c.metricStartTime)).
			WithField(endTimestampStr, util.ConvertTimeToMillis(c.metricEndTime)).
			WithField(eventTypeStr, metricStr).
			Debug("generating metric events")
	}

	c.logger.
		WithField(startTimestampStr, util.ConvertTimeToMillis(c.usageStartTime)).
		WithField(endTimestampStr, util.ConvertTimeToMillis(c.usageEndTime)).
		WithField(eventTypeStr, usageStr).
		Debug(usageMsg)

	defer c.cleanup()
	c.generateEvents()
	c.publishEvents()
	return nil
}

func (c *collector) InitializeBatch() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.metricBatch = NewEventBatch(c)
}

// AddMetric - add metric for API transaction to collection
func (c *collector) AddMetric(apiDetails models.APIDetails, statusCode string, duration, bytes int64, appName string) {
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
	c.createOrUpdateMetric(metricDetail)
}

// AddAPIMetricDetail - add metric details for several response codes and transactions
func (c *collector) AddAPIMetricDetail(detail MetricDetail) {
	if !c.metricConfig.CanPublish() || c.usageConfig.IsOfflineMode() {
		return
	}

	transactionCtx := transactionContext{
		APIDetails: detail.APIDetails,
		AppDetails: detail.AppDetails,
		Status:     detail.StatusCode,
	}
	newMetric := c.createMetric(transactionCtx)
	// update the new metric with all the necessary details
	newMetric.Count = detail.Count
	newMetric.Response = &detail.Response
	newMetric.StartTime = time.UnixMilli(detail.Observation.Start)
	newMetric.Observation = &detail.Observation
	newMetric.StatusCode = detail.StatusCode
	newMetric.Status = c.getStatusText(detail.StatusCode)

	c.addMetric(newMetric)
}

// AddCustomMetricDetail - add custom unit metric details for an api/app combo
func (c *collector) AddCustomMetricDetail(detail CustomMetricDetail) {
	if !c.metricConfig.CanPublish() || c.usageConfig.IsOfflineMode() {
		return
	}

	logger := c.logger.WithField("handler", "customMetric").
		WithField("apiID", detail.APIDetails.ID).
		WithField("appID", detail.AppDetails.ID).
		WithField("unitID", detail.UnitDetails.ID).
		WithField("unitName", detail.UnitDetails.Name)

	if detail.APIDetails.ID == "" {
		logger.Error("custom units require API information")
		return
	}

	if detail.AppDetails.ID == "" {
		logger.Error("custom units require App information")
		return
	}

	if detail.UnitDetails.ID == "" || detail.UnitDetails.Name == "" {
		logger.Error("custom units require Unit information")
		return
	}
	logger.WithField("count", detail.Count).Debug("received custom unit report")

	transactionCtx := transactionContext{
		APIDetails: detail.APIDetails,
		AppDetails: detail.AppDetails,
		UnitName:   detail.UnitDetails.ID,
	}

	metric := c.createMetric(transactionCtx)
	metric.StartTime = time.UnixMilli(detail.Observation.Start)
	metric.Observation = &detail.Observation

	if m := c.getExistingMetric(metric); m != nil {
		// use the cached metric
		metric = m
	}

	// add the count
	metric.Count += detail.Count

	counter := c.getOrRegisterCounter(metric.getKey())
	counter.Inc(detail.Count)

	c.updateMetricWithCachedMetric(metric, newCustomCounter(counter))
}

// AddAPIMetric - add api metric for API transaction
func (c *collector) AddAPIMetric(metric *APIMetric) {
	c.addMetric(centralMetricFromAPIMetric(metric))
}

// addMetric - add central metric event
func (c *collector) addMetric(metric *centralMetricEvent) {
	if metric.EventID == "" {
		metric.EventID = uuid.NewString()
	}
	metric.Status = c.getStatusText(metric.StatusCode)

	v4Event := c.createV4Event(metric.Observation.Start, metric)
	metricData, _ := json.Marshal(v4Event)
	pubEvent, err := (&CondorMetricEvent{
		Message:   string(metricData),
		Fields:    make(map[string]interface{}),
		Timestamp: v4Event.Data.GetStartTime(),
		ID:        v4Event.ID,
	}).CreateEvent()
	if err != nil {
		return
	}
	c.updateUsage(metric.Count)
	c.metricBatch.AddEventWithoutHistogram(pubEvent)
}

func (c *collector) Publish() {
	c.metricBatch.Publish()
}

func (c *collector) ShutdownPublish() {
	c.Execute()
	c.usagePublisher.Execute()
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

func (c *collector) createMetric(detail transactionContext) *centralMetricEvent {
	// Go get the access request and managed app
	accessRequest, managedApp := c.getAccessRequestAndManagedApp(agent.GetCacheManager(), detail)

	cme := &centralMetricEvent{
		metricInfo: metricInfo{
			Subscription:  c.createSubscriptionDetail(accessRequest),
			App:           c.createAppDetail(managedApp),
			Product:       c.getProduct(accessRequest),
			API:           c.createAPIDetail(detail.APIDetails),
			AssetResource: c.getAssetResource(accessRequest),
			ProductPlan:   c.getProductPlan(accessRequest),
			Quota:         c.getQuota(accessRequest, detail.UnitName),
			Unit:          c.getProductPlanUnit(accessRequest, detail.UnitName),
		},
		StartTime: now(),
		EventID:   uuid.NewString(),
	}

	if detail.Status != "" {
		cme.StatusCode = detail.Status
		cme.Status = c.getStatusText(detail.Status)
	}

	return cme
}

func (c *collector) createOrUpdateMetric(detail Detail) *centralMetricEvent {
	if !c.metricConfig.CanPublish() || c.usageConfig.IsOfflineMode() {
		return nil // no need to update metrics with publish off
	}

	transactionCtx := transactionContext{
		APIDetails: detail.APIDetails,
		AppDetails: detail.AppDetails,
		Status:     detail.StatusCode,
		UnitName:   detail.UnitName,
	}

	metric := c.createMetric(transactionCtx)

	histogram := c.getOrRegisterHistogram(metric.getKey())
	histogram.Update(detail.Duration)

	return c.updateMetricWithCachedMetric(metric, newCustomHistogram(histogram))
}

func (c *collector) getExistingMetric(metric *centralMetricEvent) *centralMetricEvent {
	keyParts := strings.Split(metric.getKey(), ".")

	c.metricMapLock.Lock()
	defer c.metricMapLock.Unlock()

	if _, ok := c.metricMap[keyParts[1]]; !ok {
		return nil
	}
	if _, ok := c.metricMap[keyParts[1]][keyParts[2]]; !ok {
		return nil
	}
	if _, ok := c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]]; !ok {
		return nil
	}
	if _, ok := c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]][keyParts[4]]; !ok {
		return nil
	}
	return c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]][keyParts[4]]
}

func (c *collector) updateMetricWithCachedMetric(metric *centralMetricEvent, cached cachedMetricInterface) *centralMetricEvent {
	keyParts := strings.Split(metric.getKey(), ".")

	c.metricMapLock.Lock()
	defer c.metricMapLock.Unlock()

	if _, ok := c.metricMap[keyParts[1]]; !ok {
		c.metricMap[keyParts[1]] = make(map[string]map[string]map[string]*centralMetricEvent)
	}
	if _, ok := c.metricMap[keyParts[1]][keyParts[2]]; !ok {
		c.metricMap[keyParts[1]][keyParts[2]] = make(map[string]map[string]*centralMetricEvent)
	}
	if _, ok := c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]]; !ok {
		c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]] = make(map[string]*centralMetricEvent)
	}
	if _, ok := c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]][keyParts[4]]; !ok {
		// First api metric for sub+app+api+statuscode,
		// setup the start time to be used for reporting metric event
		c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]][keyParts[4]] = metric
	}

	c.storage.updateMetric(cached, c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]][keyParts[4]])
	return c.metricMap[keyParts[1]][keyParts[2]][keyParts[3]][keyParts[4]]
}

// getAccessRequest -
func (c *collector) getAccessRequestAndManagedApp(cacheManager cache.Manager, detail transactionContext) (*management.AccessRequest, *v1.ResourceInstance) {
	if detail.AppDetails.Name == "" {
		return nil, nil
	}

	c.logger.
		WithField("apiID", detail.APIDetails.ID).
		WithField("stage", detail.APIDetails.Stage).
		Trace("metric collector information")

	// get the managed application
	managedApp := cacheManager.GetManagedApplicationByName(detail.AppDetails.Name)
	if managedApp == nil {
		c.logger.
			WithField("appName", detail.AppDetails.Name).
			Trace("could not get managed application by name, return empty API metrics")
		return nil, nil
	}
	c.logger.
		WithField("appName", detail.AppDetails.Name).
		WithField("managedAppName", managedApp.Name).
		Trace("managed application info")

	// get the access request
	accessRequest := transutil.GetAccessRequest(cacheManager, managedApp, detail.APIDetails.ID, detail.APIDetails.Stage, detail.APIDetails.Version)
	if accessRequest == nil {
		c.logger.
			Debug("could not get access request, return empty API metrics")
		return nil, nil
	}
	c.logger.
		WithField("managedAppName", managedApp.Name).
		WithField("apiID", detail.APIDetails.ID).
		WithField("stage", detail.APIDetails.Stage).
		WithField("accessRequestName", accessRequest.Name).
		Trace("managed application info")

	return accessRequest, managedApp
}

func (c *collector) createSubscriptionDetail(accessRequest *management.AccessRequest) *models.Subscription {
	if accessRequest == nil {
		return nil
	}

	subRef := accessRequest.GetReferenceByGVK(catalog.SubscriptionGVK())
	if subRef.ID != "" {
		return nil
	}

	return &models.Subscription{
		ID:   subRef.ID,
		Name: subRef.Name,
	}
}

func (c *collector) createAppDetail(appRI *v1.ResourceInstance) *models.AppDetails {
	if appRI == nil {

		// TODO remove the following when product plan unit ready
		// return &models.AppDetails{
		// 	ID:   "app-id",
		// 	Name: "app-name",
		// }
		return nil
	}

	app := &management.ManagedApplication{}
	app.FromInstance(appRI)

	orgID := ""
	if app.Marketplace.Resource.Owner != nil {
		orgID = app.Marketplace.Resource.Owner.Organization.ID
	}

	appRef := app.GetReferenceByGVK(catalog.ApplicationGVK())
	if appRef.ID == "" {
		return nil
	}

	return &models.AppDetails{
		ID:            appRef.ID,
		Name:          appRef.Name,
		ConsumerOrgID: orgID,
	}
}

func (c *collector) createAPIDetail(api models.APIDetails) *models.APIDetails {
	return &models.APIDetails{
		ID:       api.ID,
		Name:     api.Name,
		Revision: api.Revision,
		TeamID:   api.TeamID,
	}
}

func (c *collector) getAssetResource(accessRequest *management.AccessRequest) *models.AssetResource {
	if accessRequest == nil {
		return nil
	}

	assetResourceRef := accessRequest.GetReferenceByGVK(catalog.AssetResourceGVK())
	if assetResourceRef.ID == "" {
		return nil
	}
	return &models.AssetResource{
		ID:   assetResourceRef.ID,
		Name: assetResourceRef.Name,
	}
}

func (c *collector) getProduct(accessRequest *management.AccessRequest) *models.Product {
	if accessRequest == nil {
		return nil
	}

	productRef := accessRequest.GetReferenceByGVK(catalog.ProductGVK())
	releaseRef := accessRequest.GetReferenceByGVK(catalog.ProductReleaseGVK())

	if productRef.ID == "" || releaseRef.ID == "" {
		return nil
	}

	return &models.Product{
		ID:          productRef.ID,
		Name:        productRef.Name,
		VersionID:   releaseRef.ID,
		VersionName: releaseRef.Name,
	}
}

func (c *collector) getProductPlan(accessRequest *management.AccessRequest) *models.ProductPlan {
	if accessRequest == nil {
		return nil
	}

	productPlanRef := accessRequest.GetReferenceByGVK(catalog.ProductPlanGVK())
	if productPlanRef.ID == "" {
		return nil
	}

	return &models.ProductPlan{
		ID: productPlanRef.ID,
	}
}

func (c *collector) getQuota(accessRequest *management.AccessRequest, id string) *models.Quota {
	if accessRequest == nil {
		return nil
	}

	quotaRef := accessRequest.GetReferenceByIDAndGVK(id, catalog.QuotaGVK())
	if quotaRef.ID == "" {
		return nil
	}

	return &models.Quota{
		ID: quotaRef.ID,
	}
}

func (c *collector) getProductPlanUnit(accessRequest *management.AccessRequest, name string) *models.Unit {
	if accessRequest == nil {
		return nil
	}

	unitRef := accessRequest.GetReferenceByNameAndGVK(name, catalog.ProductPlanUnitGVK())
	if unitRef.ID == "" {
		return nil
	}

	return &models.Unit{
		ID:   unitRef.ID,
		Name: unitRef.Name,
	}
}

func (c *collector) cleanup() {
	c.publishItemQueue = make([]publishQueueItem, 0)
}

func (c *collector) getOrgGUID() string {
	authToken, _ := agent.GetCentralAuthToken()
	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = true

	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(authToken, claims)
	if err != nil {
		return ""
	}

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

	c.metricBatch = NewEventBatch(c)
	c.registry.Each(c.processRegistry)

	if len(c.metricBatch.events) == 0 && !c.usageConfig.IsOfflineMode() {
		c.logger.
			WithField(startTimestampStr, util.ConvertTimeToMillis(c.metricStartTime)).
			WithField(endTimestampStr, util.ConvertTimeToMillis(c.metricEndTime)).
			WithField(eventTypeStr, metricStr).
			Info("no metric events generated as no transactions recorded")
	}

	if c.metricConfig.CanPublish() {
		err := c.metricBatch.Publish()
		if err != nil {
			c.logger.WithError(err).Errorf("could not send metric event, data is kept and will be added to the next trigger interval")
		}
	}
}

func (c *collector) processRegistry(name string, metric interface{}) {
	switch {
	case name == transactionCountMetric:
		if c.usageConfig.CanPublish() {
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
	// skip generating a report if no usage when online
	if c.getOrRegisterCounter(transactionCountMetric).Count() == 0 && !c.usageConfig.IsOfflineMode() {
		return
	}

	usageMap := map[string]int64{
		fmt.Sprintf("%s.%s", cmd.GetBuildDataPlaneType(), lighthouseTransactions): c.getOrRegisterCounter(transactionCountMetric).Count(),
	}
	c.logger.
		WithField(startTimestampStr, util.ConvertTimeToMillis(c.usageStartTime)).
		WithField(endTimestampStr, util.ConvertTimeToMillis(c.usageEndTime)).
		WithField(countStr, c.getOrRegisterCounter(transactionCountMetric).Count()).
		WithField(eventTypeStr, usageStr).
		Info("creating usage event for cache")

	if agent.GetCentralConfig().IsAxwayManaged() {
		usageMap[fmt.Sprintf("%s.%s", cmd.GetBuildDataPlaneType(), lighthouseVolume)] = c.getOrRegisterCounter(transactionVolumeMetric).Count()
		c.logger.
			WithField(eventTypeStr, volumeStr).
			WithField("total-bytes", c.getOrRegisterCounter(transactionVolumeMetric).Count()).
			WithField(startTimestampStr, util.ConvertTimeToMillis(c.usageStartTime)).
			WithField(endTimestampStr, util.ConvertTimeToMillis(c.usageEndTime)).
			Infof("creating volume event for cache")
	}

	granularity := c.usageConfig.GetReportGranularity()
	// for offline usage reporting granularity computed with offline schedule
	if granularity == 0 {
		granularity = c.metricConfig.GetReportGranularity()
	}

	reportTime := c.usageStartTime.Format(ISO8601)
	if c.usageConfig.IsOfflineMode() {
		reportTime = c.usageEndTime.Add(time.Duration(-1*granularity) * time.Millisecond).Format(ISO8601)
	}

	usageEvent := UsageEvent{
		OrgGUID:     orgGUID,
		EnvID:       agent.GetCentralConfig().GetEnvironmentID(),
		Timestamp:   ISO8601Time(c.usageEndTime),
		SchemaID:    c.usageConfig.GetURL() + schemaPath,
		Granularity: granularity,
		Report: map[string]UsageReport{
			reportTime: {
				Product: cmd.GetBuildDataPlaneType(),
				Usage:   usageMap,
				Meta:    make(map[string]interface{}),
			},
		},
		Meta: map[string]interface{}{
			"AgentName":       agent.GetCentralConfig().GetAgentName(),
			"AgentVersion":    cmd.BuildVersion,
			"AgentType":       cmd.BuildAgentName,
			"AgentSDKVersion": cmd.SDKBuildVersion,
		},
	}

	queueItem := &usageEventQueueItem{
		event:        usageEvent,
		usageMetric:  c.getOrRegisterCounter(transactionCountMetric),
		volumeMetric: c.getOrRegisterCounter(transactionVolumeMetric),
	}
	c.publishItemQueue = append(c.publishItemQueue, queueItem)
}

func (c *collector) processMetric(metricName string, metric interface{}) {
	c.metricMapLock.Lock()
	defer c.metricMapLock.Unlock()
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
						c.setMetricsFromHistogram(statusDetail, statusMetric)
						c.generateMetricEvent(statusMetric, statusDetail)
					}
				}
			}
		}
	}
}

func (c *collector) setMetricsFromHistogram(metrics *centralMetricEvent, histogram metrics.Histogram) {
	metrics.Count = histogram.Count()
	metrics.Response = &ResponseMetrics{
		Max: histogram.Max(),
		Min: histogram.Min(),
		Avg: histogram.Mean(),
	}
}

func (c *collector) generateMetricEvent(histogram metrics.Histogram, metric *centralMetricEvent) {
	if metric.Count == 0 {
		c.logger.Trace("skipping registry entry with no reported quantity")
		return
	}
	metric.Observation = &ObservationDetails{
		Start: util.ConvertTimeToMillis(c.metricStartTime),
		End:   util.ConvertTimeToMillis(c.metricEndTime),
	}

	// Generate app subscription metric
	c.generateV4Event(histogram, metric)
}

func (c *collector) createV4Event(startTime int64, v4data V4Data) V4Event {
	return V4Event{
		ID:        v4data.GetEventID(),
		Timestamp: startTime,
		Event:     metricEvent,
		App:       c.orgGUID,
		Version:   "4",
		Distribution: &V4EventDistribution{
			Environment: agent.GetCentralConfig().GetEnvironmentID(),
			Version:     "1",
		},
		Data: v4data,
	}
}

func (c *collector) generateV4Event(histogram metrics.Histogram, v4data V4Data) {
	generatedEvent := c.createV4Event(c.metricStartTime.UnixMilli(), v4data)
	c.metricLogger.WithFields(generatedEvent.getLogFields()).Info("generated")
	AddCondorMetricEventToBatch(generatedEvent, c.metricBatch, histogram)
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
			err := c.usagePublisher.publishEvent(eventQueueItem.GetEvent())
			if err != nil {
				c.logger.
					WithError(err).
					WithField(startTimestampStr, util.ConvertTimeToMillis(c.usageStartTime)).
					WithField(endTimestampStr, util.ConvertTimeToMillis(c.usageEndTime)).
					WithField(eventTypeStr, usageStr).
					Error("failed to add usage report to cache. Current usage report is kept and will be added to the next interval")
			} else {
				c.logger.
					WithField(startTimestampStr, util.ConvertTimeToMillis(c.usageStartTime)).
					WithField(endTimestampStr, util.ConvertTimeToMillis(c.usageEndTime)).
					Info("added usage report to cache")
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

func (c *collector) logMetric(msg string, metric *APIMetric) {
	c.metricLogger.WithField("id", metric.EventID).Info(msg)
}

func (c *collector) cleanupMetricCounter(histogram metrics.Histogram, metric *APIMetric) {
	c.metricMapLock.Lock()
	defer c.metricMapLock.Unlock()
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
		WithField(startTimestampStr, util.ConvertTimeToMillis(c.usageStartTime)).
		WithField(endTimestampStr, util.ConvertTimeToMillis(c.usageEndTime)).
		WithField("api-name", metric.API.Name).
		Info("Published metrics report for API")
}

func (c *collector) getStatusText(statusCode string) string {
	httpStatusCode, _ := strconv.Atoi(statusCode)
	switch {
	case httpStatusCode >= 100 && httpStatusCode < 400:
		return "Success"
	case httpStatusCode >= 400 && httpStatusCode < 500:
		return "Failure"
	default:
		return "Exception"
	}
}
