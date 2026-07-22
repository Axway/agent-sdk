package metric

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	startTimestampStr = "startTimestamp"
	endTimestampStr   = "endTimestamp"
	eventTypeStr      = "eventType"
	usageStr          = "usage"
	metricStr         = "metric"
	volumeStr         = "volume"
	countStr          = "count"
	defaultUnit       = "transactions"
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
	AddMetric(apiDetails models.APIDetails, statusCode string, duration, bytes int64, appName string)
	AddCustomMetricDetail(metric models.CustomMetricDetail)
	AddMetricDetail(metricDetail Detail)
	AddAPIMetricDetail(metric MetricDetail)
	AddAPIMetric(apiMetric *APIMetric)
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
	agentName        string
	lock             *sync.Mutex
	batchLock        *sync.Mutex
	registry         registry
	metricBatch      *EventBatch
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
	usageMetric  *counter
	volumeMetric *counter
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
		if agent.GetCustomUnitHandler() != nil {
			agent.GetCustomUnitHandler().HandleMetricReporting(globalMetricCollector)
		}
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
		usageStartTime:   now().Truncate(time.Minute), // round down to closest minute
		metricStartTime:  now().Truncate(time.Minute), // round down to closest minute
		lock:             &sync.Mutex{},
		batchLock:        &sync.Mutex{},
		registry:         newRegistry(),
		publishItemQueue: make([]publishQueueItem, 0),
		metricConfig:     agent.GetCentralConfig().GetMetricReportingConfig(),
		usageConfig:      agent.GetCentralConfig().GetUsageReportingConfig(),
		agentName:        agent.GetCentralConfig().GetAgentName(),
		logger:           logger,
		metricLogger:     log.NewMetricFieldLogger(),
	}

	// Create and initialize the storage cache for usage/metric and offline report cache by loading from disk
	metricCollector.storage = newStorageCache(metricCollector)
	metricCollector.storage.initialize()
	metricCollector.reports = newReportCache()
	metricCollector.usagePublisher = newUsagePublisher(metricCollector.storage, metricCollector.reports, metricCollector.updateUsageStartTime, metricCollector.metricCheck)

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

// metricCheck - lock to wait for any inflight metric processing to complete
func (c *collector) metricCheck() {
	c.lock.Lock()
	defer c.lock.Unlock()
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

func (c *collector) updateStartTime() {
	if c.metricStartTime.IsZero() {
		c.metricStartTime = now().Truncate(time.Minute)
	}
}

// AddMetric - add metric for API transaction to collection
func (c *collector) AddMetric(apiDetails models.APIDetails, statusCode string, duration, bytes int64, appName string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.batchLock.Lock()
	defer c.batchLock.Unlock()
	c.updateStartTime()
	c.updateUsage(1)
	c.updateVolume(bytes)
}

// AddMetricDetail - add metric for API transaction and consumer subscription to collection
func (c *collector) AddMetricDetail(metricDetail Detail) {
	c.AddMetric(metricDetail.APIDetails, metricDetail.StatusCode, metricDetail.Duration, metricDetail.Bytes, metricDetail.APIDetails.Name)
	c.createOrUpdateAPICounter(metricDetail)
}

// AddAPIMetricDetail - add metric details for several response codes and transactions
func (c *collector) AddAPIMetricDetail(detail MetricDetail) {
	if !c.metricConfig.CanPublish() || c.usageConfig.IsOfflineMode() {
		return
	}

	c.lock.Lock()
	c.batchLock.Lock()
	c.updateStartTime()
	c.updateUsage(detail.Count)
	c.batchLock.Unlock()
	c.lock.Unlock()

	c.createOrUpdateAPICounterStats(Detail{
		APIDetails: detail.APIDetails,
		AppDetails: detail.AppDetails,
		StatusCode: detail.StatusCode,
	}, detail.Count, detail.Response.Min, detail.Response.Max, detail.Response.Avg)
}

// AddCustomMetricDetail - add custom unit metric details for an api/app combo
func (c *collector) AddCustomMetricDetail(detail models.CustomMetricDetail) {
	if !c.metricConfig.CanPublish() || c.usageConfig.IsOfflineMode() {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.batchLock.Lock()
	defer c.batchLock.Unlock()

	logger := c.logger.WithField("handler", "customMetric").
		WithField("apiID", detail.APIDetails.ID).
		WithField("appID", detail.AppDetails.ID).
		WithField("unitName", detail.UnitDetails.Name)

	if detail.APIDetails.ID == "" {
		logger.Error("custom units require API information")
		return
	}

	if detail.AppDetails.ID == "" {
		logger.Error("custom units require App information")
		return
	}

	if detail.UnitDetails.Name == "" {
		logger.Error("custom units require Unit information")
		return
	}
	logger.WithField("count", detail.Count).Debug("received custom unit report")

	transactionCtx := transactionContext{
		APIDetails: detail.APIDetails,
		AppDetails: detail.AppDetails,
		UnitName:   detail.UnitDetails.Name,
	}

	metric := c.createMetric(transactionCtx)

	if m := c.getExistingMetric(metric); m != nil {
		// use the cached metric
		metric = m
	}

	// add the count
	metric.Units.CustomUnits[detail.UnitDetails.Name].Count += detail.Count

	c.updateStartTime()
	counter := c.getOrRegisterGroupedCounter(metric.getKey())
	counter.Inc(detail.Count)

	c.updateMetricWithCachedMetric(metric, newCustomCounter(counter))
}

// AddAPIMetric - add api metric for API transaction, merging its counts and response stats into
// any metric already cached for the same subscription/application/api/status
func (c *collector) AddAPIMetric(apiMetric *APIMetric) {
	if !c.metricConfig.CanPublish() || c.usageConfig.IsOfflineMode() {
		return
	}

	metric := centralMetricFromAPIMetric(apiMetric)
	if metric == nil {
		return
	}
	if metric.EventID == "" {
		metric.EventID = uuid.NewString()
	}

	// the incoming metric already carries fully resolved subscription/app/product context,
	// so mark it resolved to keep resolveMetricContext from overwriting it from the cache later
	metric.ctx = transactionContext{AppDetails: apiMetric.App}
	metric.resolved = true

	c.lock.Lock()
	defer c.lock.Unlock()
	c.batchLock.Lock()
	defer c.batchLock.Unlock()

	c.updateStartTime()

	if apiMetric.Unit != nil {
		c.updateCachedCustomUnitMetric(apiMetric, metric)
		return
	}

	apiCtr := c.getOrRegisterGroupedAPICounter(metric.getKey())
	apiCtr.UpdateWithStats(apiMetric.Count, apiMetric.Response.Min, apiMetric.Response.Max, apiMetric.Response.Avg)
	c.updateMetricWithCachedMetric(metric, apiCtr)
}

// updateCachedCustomUnitMetric merges a custom-unit APIMetric into any cached metric already tracked for its key
func (c *collector) updateCachedCustomUnitMetric(apiMetric *APIMetric, metric *centralMetric) {
	if m := c.getExistingMetric(metric); m != nil {
		metric = m
	}
	metric.Units.CustomUnits[apiMetric.Unit.Name].Count += apiMetric.Count

	counter := c.getOrRegisterGroupedCounter(metric.getKey())
	counter.Inc(apiMetric.Count)

	c.updateMetricWithCachedMetric(metric, newCustomCounter(counter))
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

// creates a centralMetric with detail available now, resolution of full central context will happen later
func (c *collector) createMetric(detail transactionContext) *centralMetric {
	apicDeployment, _, runtimeType := centralConfigFields()

	me := &centralMetric{
		Version:        metricDataVersion,
		APICDeployment: apicDeployment,
		Environment:    &EnvironmentInfo{RuntimeType: runtimeType},
		API:            c.createAPIDetail(detail.APIDetails),
		Observation: &models.ObservationDetails{
			Start: now().Unix(),
		},
		EventID: uuid.NewString(),
		ctx:     detail,
	}

	// transactions
	if detail.Status != "" {
		me.Units = &Units{
			Transactions: &Transactions{
				Status: GetStatusText(detail.Status),
			},
		}
	} else if detail.UnitName != "" {
		me.Units = &Units{
			CustomUnits: map[string]*UnitCount{
				detail.UnitName: {},
			},
		}
	}

	return me
}

// resolveMetricContext resolves the access request/managed application for metric
func (c *collector) resolveMetricContext(metric *centralMetric) {
	if metric.resolved {
		return
	}

	accessRequest, managedApp := c.getAccessRequestAndManagedApp(agent.GetCacheManager(), metric.ctx)

	metric.Marketplace = transutil.GetMarketplaceDetails(managedApp)
	metric.Subscription = c.createSubscriptionDetail(accessRequest)
	metric.App = c.createAppDetail(managedApp)
	metric.Product = c.getProduct(accessRequest)
	metric.AssetResource = c.getAssetResource(accessRequest)
	metric.APIServiceRevision = c.getAPIServiceRevision(accessRequest)
	metric.ProductPlan = c.getProductPlan(accessRequest)

	if metric.Units != nil {
		if metric.Units.Transactions != nil {
			metric.Units.Transactions.Quota = c.getQuota(accessRequest, defaultUnit)
		}
		for name, unitCount := range metric.Units.CustomUnits {
			unitCount.Quota = c.getQuota(accessRequest, name)
		}
	}

	// one-shot: whether or not managedApp resolved, don't keep retrying on subsequent cycles
	metric.resolved = true
}

// getResolvedMetric fetches the metric template cached under key in group, resolving its access
// request/managed application context (if not already resolved) before returning it.
func (c *collector) getResolvedMetric(group groupedMetrics, key string) (*centralMetric, bool) {
	metric, ok := group.getMetric(key)
	if !ok {
		return nil, false
	}
	c.resolveMetricContext(metric)
	return metric, true
}

func (c *collector) createOrUpdateAPICounter(detail Detail) *centralMetric {
	metric, apiCounter := c.setupAPICounter(detail)
	if metric == nil {
		return nil
	}

	apiCounter.Update(detail.Duration)

	return c.updateMetricWithCachedMetric(metric, apiCounter)
}

// createOrUpdateAPICounterStats - add a batch of transactions known by count, min, max, and average response time
func (c *collector) createOrUpdateAPICounterStats(detail Detail, count, min, max int64, avg float64) *centralMetric {
	metric, apiCounter := c.setupAPICounter(detail)
	if metric == nil {
		return nil
	}

	apiCounter.UpdateWithStats(count, min, max, avg)

	return c.updateMetricWithCachedMetric(metric, apiCounter)
}

func (c *collector) setupAPICounter(detail Detail) (*centralMetric, *apiCounter) {
	if !c.metricConfig.CanPublish() || c.usageConfig.IsOfflineMode() {
		return nil, nil // no need to update metrics with publish off
	}

	// Update the detail with the resolved API ID
	detail.APIDetails.ID = transutil.ResolveIDWithPrefix(detail.APIDetails.ID, detail.APIDetails.Name)

	transactionCtx := transactionContext{
		APIDetails: detail.APIDetails,
		AppDetails: detail.AppDetails,
		Status:     detail.StatusCode,
		UnitName:   detail.UnitName,
	}

	metric := c.createMetric(transactionCtx)

	apiCounter := c.getOrRegisterGroupedAPICounter(metric.getKey())

	return metric, apiCounter
}

func (c *collector) getExistingMetric(metric *centralMetric) *centralMetric {
	groupKey, uniqueKey := splitMetricKey(metric.getKey())
	groupedMetric := c.getOrRegisterGroupedMetrics(c.groupKeyWithStartTime(groupKey))

	existing, _ := groupedMetric.getMetric(uniqueKey)
	return existing
}

func (c *collector) updateMetricWithCachedMetric(metric *centralMetric, cached cachedMetricInterface) *centralMetric {
	groupKey, uniqueKey := splitMetricKey(metric.getKey())
	groupedMetric := c.getOrRegisterGroupedMetrics(c.groupKeyWithStartTime(groupKey))

	// first api metric for sub+app+api+statuscode wins and becomes the template used for reporting
	metric = groupedMetric.getOrSetMetric(uniqueKey, metric)

	c.storage.updateMetric(cached, metric)
	return metric
}

// getAccessRequest -
func (c *collector) getAccessRequestAndManagedApp(cacheManager cache.Manager, detail transactionContext) (*management.AccessRequest, *v1.ResourceInstance) {
	if detail.AppDetails.Name == "" && detail.AppDetails.ID == "" {
		return nil, nil
	}

	c.logger.
		WithField("apiID", detail.APIDetails.ID).
		WithField("stage", detail.APIDetails.Stage).
		Trace("metric collector information")

	// get the managed application
	// cached metrics will only have the catalog api id
	managedApp := cacheManager.GetManagedApplicationByApplicationID(detail.AppDetails.ID)
	if managedApp == nil {
		managedApp = cacheManager.GetManagedApplication(detail.AppDetails.ID)
	}
	if managedApp == nil {
		managedApp = cacheManager.GetManagedApplicationByName(detail.AppDetails.Name)
	}
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
			Debug("could not get access request, return managed application only")
		return nil, managedApp
	}
	c.logger.
		WithField("managedAppName", managedApp.Name).
		WithField("apiID", detail.APIDetails.ID).
		WithField("stage", detail.APIDetails.Stage).
		WithField("accessRequestName", accessRequest.Name).
		Trace("managed application info")

	return accessRequest, managedApp
}

func (c *collector) createSubscriptionDetail(accessRequest *management.AccessRequest) *models.ResourceReference {
	if accessRequest == nil {
		return &models.ResourceReference{ID: unknown}
	}

	subRef := accessRequest.GetReferenceByGVK(catalog.SubscriptionGVK())
	if subRef.ID == "" {
		return &models.ResourceReference{ID: unknown}
	}

	return &models.ResourceReference{
		ID: subRef.ID,
	}
}

func (c *collector) createAppDetail(appRI *v1.ResourceInstance) *models.ApplicationResourceReference {
	if appRI == nil {
		return &models.ApplicationResourceReference{
			ResourceReference: models.ResourceReference{ID: unknown},
			ConsumerOrgID:     none,
			Owner:             &models.Owner{Type: none},
		}
	}

	app := &management.ManagedApplication{}
	app.FromInstance(appRI)

	orgID := none
	if app.Marketplace.Resource.Owner != nil && app.Marketplace.Resource.Owner.Organization.ID != "" {
		orgID = app.Marketplace.Resource.Owner.Organization.ID
	}

	appID := unknown
	if appRef := app.GetReferenceByGVK(catalog.ApplicationGVK()); appRef.ID != "" {
		appID = appRef.ID
	}

	return &models.ApplicationResourceReference{
		ResourceReference: models.ResourceReference{
			ID: appID,
		},
		ConsumerOrgID: orgID,
		Owner:         transutil.ResolveAppOwnerFromManagedApp(appRI),
	}
}

func (c *collector) createAPIDetail(api models.APIDetails) *models.APIResourceReference {
	ref := &models.APIResourceReference{
		ResourceReference: models.ResourceReference{
			ID: api.ID,
		},
		Name: api.Name,
	}
	cacheManager := agent.GetCacheManager()
	svc := cacheManager.GetAPIServiceWithAPIID(strings.TrimPrefix(api.ID, transutil.SummaryEventProxyIDPrefix))
	ref.APIServiceID = unknown
	if svc != nil {
		ref.APIServiceID = svc.Metadata.ID
	}
	ref.Owner = transutil.ResolveAPIOwnerFromInstance(svc)
	return ref
}

// getAPIServiceRevision uses the APIServiceInstance reference on the AccessRequest as the
// revision identifier. AccessRequests do not carry a direct APIServiceRevision reference.
func (c *collector) getAPIServiceRevision(accessRequest *management.AccessRequest) *models.ResourceReference {
	if accessRequest == nil {
		return &models.ResourceReference{ID: unknown}
	}

	ref := accessRequest.GetReferenceByGVK(management.APIServiceInstanceGVK())
	if ref.ID == "" {
		return &models.ResourceReference{ID: unknown}
	}

	return &models.ResourceReference{ID: ref.ID}
}

func (c *collector) getAssetResource(accessRequest *management.AccessRequest) *models.ResourceReference {
	if accessRequest == nil {
		return &models.ResourceReference{ID: unknown}
	}

	assetResourceRef := accessRequest.GetReferenceByGVK(catalog.AssetResourceGVK())
	if assetResourceRef.ID == "" {
		return &models.ResourceReference{ID: unknown}
	}

	return &models.ResourceReference{
		ID: assetResourceRef.ID,
	}
}

func (c *collector) getProduct(accessRequest *management.AccessRequest) *models.ProductResourceReference {
	if accessRequest == nil {
		return &models.ProductResourceReference{
			ResourceReference: models.ResourceReference{ID: unknown},
			VersionID:         unknown,
		}
	}

	productRef := accessRequest.GetReferenceByGVK(catalog.ProductGVK())
	releaseRef := accessRequest.GetReferenceByGVK(catalog.ProductReleaseGVK())

	ref := &models.ProductResourceReference{
		ResourceReference: models.ResourceReference{ID: unknown},
		VersionID:         unknown,
	}
	if productRef.ID != "" {
		ref.ID = productRef.ID
		// owner only applies once the product itself is resolved
		ref.Owner = transutil.ResolveProductOwner(accessRequest.GetEmbeddedReferenceByGVK(catalog.PublishedProductGVK()))
	}
	if releaseRef.ID != "" {
		ref.VersionID = releaseRef.ID
	}

	return ref
}

func (c *collector) getProductPlan(accessRequest *management.AccessRequest) *models.ResourceReference {
	if accessRequest == nil {
		return &models.ResourceReference{ID: unknown}
	}

	productPlanRef := accessRequest.GetReferenceByGVK(catalog.ProductPlanGVK())
	if productPlanRef.ID == "" {
		return &models.ResourceReference{ID: unknown}
	}

	return &models.ResourceReference{
		ID: productPlanRef.ID,
	}
}

func (c *collector) getQuota(accessRequest *management.AccessRequest, unitName string) *models.ResourceReference {
	if accessRequest == nil {
		return &models.ResourceReference{ID: unknown}
	}
	if unitName == "" {
		unitName = defaultUnit
	}

	quotaName := ""

	// get quota for unit
	for _, r := range accessRequest.References {
		rMap := r.(map[string]interface{})
		if rMap["kind"].(string) != catalog.QuotaGVK().Kind {
			continue
		}
		if unit, ok := rMap["unit"]; ok && unit.(string) == unitName {
			// no unit is transactions
			quotaName = strings.Split(rMap["name"].(string), "/")[2]
			break
		}
	}

	if quotaName == "" {
		return &models.ResourceReference{ID: unknown}
	}

	quotaRef := accessRequest.GetReferenceByNameAndGVK(quotaName, catalog.QuotaGVK())
	if quotaRef.ID == "" {
		return &models.ResourceReference{ID: unknown}
	}

	return &models.ResourceReference{
		ID: quotaRef.ID,
	}
}

func (c *collector) cleanup() {
	c.publishItemQueue = make([]publishQueueItem, 0)
}

func (c *collector) getOrgGUID() string {
	return GetOrgGUID()
}

// GetOrgGUID parses the provider org GUID from the central auth token JWT.
// Returns empty string if the token is unavailable or the claim is absent.
func GetOrgGUID() string {
	authToken, _ := agent.GetCentralAuthToken()
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(authToken, claims)
	if err != nil {
		return ""
	}
	if claim, ok := claims["org_guid"]; ok {
		return claim.(string)
	}
	return ""
}

func (c *collector) generateEvents() {
	if agent.GetCentralConfig().GetEnvironmentID() == "" || cmd.GetBuildDataPlaneType() == "" {
		c.logger.Warn("Unable to process usage and metric event generation. Please verify the agent config")
		return
	}

	// snapshot the start time in effect for this publish cycle, then reset the attribute so that any
	// metrics recorded from here on start a new generation instead of being folded into this batch
	publishStartTime := c.metricStartTime
	c.metricStartTime = time.Time{}

	c.metricBatch = NewEventBatch(c)
	c.registry.Each(func(name string, metric interface{}) {
		c.processRegistry(name, metric, publishStartTime)
	})

	if len(c.metricBatch.events) == 0 && !c.usageConfig.IsOfflineMode() {
		c.logger.
			WithField(startTimestampStr, util.ConvertTimeToMillis(publishStartTime)).
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

func (c *collector) processRegistry(name string, metric interface{}, publishStartTime time.Time) {
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
		c.processMetric(name, metric, publishStartTime)
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

func (c *collector) processMetric(metricName string, groupedMetricInterface interface{}, publishStartTime time.Time) {
	elements := strings.Split(metricName, ".")
	if len(elements) != 4 {
		return
	}

	groupStartTime, err := strconv.ParseInt(elements[3], 10, 64)
	if err != nil || groupStartTime > util.ConvertTimeToMillis(publishStartTime) {
		// this generation of metrics started after this publish cycle began, handle it on a later cycle
		return
	}

	groupedMetric, ok := groupedMetricInterface.(groupedMetrics)
	if !ok {
		c.logger.Error("metric data to process was not the expected type")
		return
	}

	logger := c.logger.
		WithField("applicationID", desanitizeKeySegment(elements[1])).
		WithField("apiID", desanitizeKeySegment(elements[2]))
	c.handleGroupedMetric(logger, groupedMetric, publishStartTime, metricName)
}

func (c *collector) handleGroupedMetric(logger log.FieldLogger, groupedMetric groupedMetrics, publishStartTime time.Time, registryKey string) {
	countersAdded := false
	// handle each api counter, on the first one add the counter information
	for k, apiCtr := range groupedMetric.apiCounters {
		logger := logger.WithField("status", k)
		metric, ok := c.getResolvedMetric(groupedMetric, k)
		if !ok {
			logger.Debug("no metrics in map for status")
			continue
		}
		c.setMetricsFromAPICounter(metric, apiCtr)
		var counters map[string]*counter
		if !countersAdded {
			c.setMetricCounters(logger, metric, groupedMetric)
			counters = groupedMetric.counters
			countersAdded = true
		}
		c.generateMetricEvent(counters, metric, publishStartTime, registryKey, groupedMetric)
	}

	// create metric with just custom units
	if !countersAdded && len(groupedMetric.counters) > 0 {
		key := ""
		for k := range groupedMetric.counters {
			key = k
			break
		}
		metric, ok := c.getResolvedMetric(groupedMetric, key)
		if !ok {
			logger.WithField("counterKey", key).Error("could not get metric for counter")
			return
		}
		c.setMetricCounters(logger, metric, groupedMetric)
		c.generateMetricEvent(groupedMetric.counters, metric, publishStartTime, registryKey, groupedMetric)
	}
}

func (c *collector) setMetricCounters(logger log.FieldLogger, metricData *centralMetric, groupedMetric groupedMetrics) {
	if metricData.Units.CustomUnits == nil {
		metricData.Units.CustomUnits = map[string]*UnitCount{}
	}

	for k, cnt := range groupedMetric.counters {
		logger := logger.WithField("unit", k)
		metric, ok := c.getResolvedMetric(groupedMetric, k)
		if !ok {
			logger.Error("no counter in map for unit")
			continue
		}

		// create a new quota pointer
		var quota *models.ResourceReference
		if metric.Units.CustomUnits[k].Quota != nil {
			quota = &models.ResourceReference{
				ID: metric.Units.CustomUnits[k].Quota.ID,
			}
		}

		metricData.Units.CustomUnits[k] = &UnitCount{
			Count: cnt.Count(),
			Quota: quota,
		}
	}
}

func (c *collector) setMetricsFromAPICounter(m *centralMetric, apiCtr *apiCounter) {
	m.Units.Transactions.Count = apiCtr.Count()
	m.Units.Transactions.Duration = int64(apiCtr.Mean() * float64(apiCtr.Count()))
	m.Units.Transactions.Response = &ResponseMetrics{
		Max: apiCtr.Max(),
		Min: apiCtr.Min(),
		Avg: apiCtr.Mean(),
	}
}

func (c *collector) generateMetricEvent(counters map[string]*counter, metric *centralMetric, publishStartTime time.Time, registryKey string, group groupedMetrics) {
	if metric.Units != nil && metric.Units.Transactions != nil && metric.Units.Transactions.Count == 0 {
		c.logger.Trace("skipping registry entry with no reported quantity")
		return
	}
	metric.Observation = &models.ObservationDetails{
		Start: util.ConvertTimeToMillis(publishStartTime),
		End:   util.ConvertTimeToMillis(c.metricEndTime),
	}
	metric.Reporter = &Reporter{
		AgentVersion:     cmd.BuildVersion,
		AgentType:        cmd.BuildAgentName,
		AgentSDKVersion:  cmd.SDKBuildVersion,
		AgentName:        c.agentName,
		ObservationDelta: metric.Observation.End - metric.Observation.Start,
	}

	// Generate app subscription metric
	c.generateV4Event(counters, metric, publishStartTime, registryKey, group)
}

func (c *collector) createV4Event(startTime int64, v4data V4Data) V4Event {
	return V4Event{
		ID:        v4data.GetEventID(),
		Timestamp: startTime,
		Event:     metricEvent,
		Org:       c.orgGUID,
		Version:   "4",
		Distribution: &V4EventDistribution{
			Environment: agent.GetCentralConfig().GetEnvironmentID(),
		},
		Data: v4data,
	}
}

func (c *collector) generateV4Event(counters map[string]*counter, v4data V4Data, publishStartTime time.Time, registryKey string, group groupedMetrics) {
	generatedEvent := c.createV4Event(publishStartTime.UnixMilli(), v4data)
	c.metricLogger.WithFields(generatedEvent.getLogFields()).Info("generated")
	AddCondorMetricEventToBatch(generatedEvent, c.metricBatch, registryKey, counters, group)
}

func (c *collector) getOrRegisterCounter(name string) *counter {
	cnt := c.registry.Get(name)
	if cnt == nil {
		cnt = newCounter()
		c.registry.Register(name, cnt)
	}
	return cnt.(*counter)
}

func (c *collector) getOrRegisterGroupedMetrics(name string) groupedMetrics {
	group := c.registry.Get(name)
	if group == nil {
		group = newGroupedMetric()
		c.registry.Register(name, group)
	}
	return group.(groupedMetrics)
}

func (c *collector) getOrRegisterGroupedCounter(name string) *counter {
	groupKey, countKey := splitMetricKey(name)
	groupedMetric := c.getOrRegisterGroupedMetrics(c.groupKeyWithStartTime(groupKey))

	return groupedMetric.getOrCreateCounter(countKey)
}

func (c *collector) getOrRegisterGroupedAPICounter(name string) *apiCounter {
	groupKey, counterKey := splitMetricKey(name)
	groupedMetric := c.getOrRegisterGroupedMetrics(c.groupKeyWithStartTime(groupKey))

	return groupedMetric.getOrCreateAPICounter(counterKey)
}

// groupKeyWithStartTime - appends the current metric generation's start time to the group key so that
// metrics accumulated under different start times are kept in separate registry entries
func (c *collector) groupKeyWithStartTime(groupKey string) string {
	return fmt.Sprintf("%s.%d", groupKey, c.metricStartTime.UnixMilli())
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
	if usage, ok := itemUsageMetric.(*counter); ok {
		// Clean up the usage counter and reset the start time to current endTime
		usage.Clear()
		itemVolumeMetric := usageEventItem.GetVolumeMetric()
		if volume, ok := itemVolumeMetric.(*counter); ok {
			volume.Clear()
		}
		c.storage.updateUsage(0)
		c.storage.updateVolume(0)
	}
}

func (c *collector) updateUsageStartTime() {
	// called after usage report publishing job is executed
	c.usageStartTime = now().Truncate(time.Minute)
	c.storage.updateUsage(0)
}

func (c *collector) logMetric(msg string, metric *centralMetric) {
	c.metricLogger.WithField("id", metric.EventID).Info(msg)
}

// cleanupMetricCounters - called once a metric event has been acked, to remove the persisted cache
// entry for the published metric (and any custom unit metrics acked alongside it), and to remove that
// status/unit's entry from the group. A status/unit whose event was never acked (publish failed, was
// retried, or cancelled) is left in the group so it is picked up again on the next publish cycle instead
// of being lost. Once every entry in the group has been acked, the group itself is removed from the
// registry.
func (c *collector) cleanupMetricCounters(registryKey string, counters map[string]*counter, group groupedMetrics, metric *centralMetric) {
	c.storage.removeMetric(metric)

	_, statusKey := splitMetricKey(metric.getKey())
	empty := group.removeAndCheckEmpty(statusKey)

	for k := range counters {
		if m, ok := group.getMetric(k); ok {
			c.storage.removeMetric(m)
		}
		empty = group.removeAndCheckEmpty(k)
	}

	if empty {
		c.registry.Deregister(registryKey)
	}

	c.logger.
		WithField(startTimestampStr, util.ConvertTimeToMillis(c.usageStartTime)).
		WithField(endTimestampStr, util.ConvertTimeToMillis(c.usageEndTime)).
		WithField("apiName", metric.API.Name).
		Info("Published metrics report for API")
}

func GetStatusText(statusCode string) string {
	return sampling.GetStatusFromCodeString(statusCode).String()
}
