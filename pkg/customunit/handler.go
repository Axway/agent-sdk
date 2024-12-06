package customunit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transUtil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type CustomUnitHandler struct {
	servicesConfigs  []config.MetricServiceConfiguration
	cache            agentCacheManager
	agentType        config.AgentType
	logger           log.FieldLogger
	clients          []*customUnitClient
	metricReportChan chan *customunits.MetricReport
	stopChan         chan struct{}
	apiCustomLookups map[string]string
	appCustomLookups map[string]string
}

type metricCollector interface {
	AddCustomMetricDetail(models.CustomMetricDetail)
}

type agentCacheManager interface {
	GetAPIServiceWithName(string) *v1.ResourceInstance
	GetAPIServiceInstanceByID(string) (*v1.ResourceInstance, error)
	GetAPIServiceKeys() []string
	GetAPIServiceWithAPIID(string) *v1.ResourceInstance
	GetManagedApplication(string) *v1.ResourceInstance
	GetManagedApplicationByName(string) *v1.ResourceInstance
	GetManagedApplicationCacheKeys() []string
}

func NewCustomUnitHandler(servicesConfigs []config.MetricServiceConfiguration, cache agentCacheManager, agentType config.AgentType) *CustomUnitHandler {
	return &CustomUnitHandler{
		servicesConfigs:  servicesConfigs,
		cache:            cache,
		agentType:        agentType,
		metricReportChan: make(chan *customunits.MetricReport, 100),
		stopChan:         make(chan struct{}),
		clients:          []*customUnitClient{},
		logger:           log.NewFieldLogger().WithPackage("customunit").WithComponent("manager"),
		apiCustomLookups: map[string]string{},
		appCustomLookups: map[string]string{},
	}
}

func (h *CustomUnitHandler) HandleQuotaEnforcement(ar *management.AccessRequest, app *management.ManagedApplication) error {
	if len(h.servicesConfigs) == 0 {
		return nil
	}

	// Build quota info
	logger := h.logger.WithField("applicationName", app.Name).WithField("apiInstance", ar.Spec.ApiServiceInstance)
	quotaInfo, err := h.buildQuotaInfo(logger, ar, app)
	if err != nil {
		logger.WithError(err).Error("could not build quota info from access request")
		return err
	}

	var errMsgs []string
	mutex := sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(h.servicesConfigs))

	rejectOnFail := false
	for i := range h.servicesConfigs {
		go func(c config.MetricServiceConfiguration) {
			defer wg.Done()

			msg := h.handleServiceQE(c, quotaInfo)
			if msg == "" {
				return
			}
			msg = fmt.Sprintf("service (url: %s, rejectOnFail: %v, message: %s)", c.URL, c.RejectOnFail, msg)

			mutex.Lock()
			defer mutex.Unlock()
			errMsgs = append(errMsgs, msg)
			rejectOnFail = rejectOnFail || c.RejectOnFail
		}(h.servicesConfigs[i])
	}
	wg.Wait()

	var qeErr error
	if len(errMsgs) > 0 {
		qeErr = errors.New(strings.Join(errMsgs, "; "))
		logger.WithError(qeErr).Error("errors from metric services for quota enforcement")
	}

	if rejectOnFail {
		return qeErr
	}
	return nil
}

func (h *CustomUnitHandler) handleServiceQE(config config.MetricServiceConfiguration, quotaInfo *customunits.QuotaInfo) string {
	if !config.Enable {
		return ""
	}
	factory := NewCustomUnitClientFactory(config.URL, quotaInfo)
	client, err := factory(h.cache)
	// err creating the client
	if err != nil {
		return err.Error()
	}

	response, err := client.QuotaEnforcementInfo()
	// err or errored response from the quota enforcement call
	if err != nil {
		return err.Error()
	} else if response != nil && response.Error != "" {
		return response.Error
	}

	return ""
}

func (h *CustomUnitHandler) buildQuotaInfo(logger log.FieldLogger, ar *management.AccessRequest, app *management.ManagedApplication) (*customunits.QuotaInfo, error) {
	unitRef, interval, count := h.getQuotaInfo(ar)
	quota := &customunits.Quota{}
	if unitRef != "" {
		quota = &customunits.Quota{
			Count:    int64(count),
			Unit:     unitRef,
			Interval: intervalToProtoInterval(interval),
		}
	}

	instance, err := h.getServiceInstance(ar)
	if err != nil {
		return nil, err
	}

	// Get service instance from access request to fetch the api service
	serviceRef := instance.GetReferenceByGVK(management.APIServiceGVK())
	service := h.cache.GetAPIServiceWithName(serviceRef.Name)
	if service == nil {
		return nil, fmt.Errorf("could not find service connected to quota")
	}
	extAPIID, err := util.GetAgentDetailsValue(instance, definitions.AttrExternalAPIID)
	if err != nil {
		logger.WithError(err).Error("external api id not found on api service, still sending to metric services")
	}
	extAppID, err := util.GetAgentDetailsValue(app, definitions.AttrExternalAppID)
	if err != nil {
		logger.WithError(err).Error("external app id not found on api service, still sending to metric services")
	}

	q := &customunits.QuotaInfo{
		ApiInfo: &customunits.APIInfo{
			ServiceDetails: util.GetAgentDetailStrings(service),
			ServiceName:    service.Name,
			ServiceID:      service.Metadata.ID,
			ExternalAPIID:  extAPIID,
		},
		AppInfo: &customunits.AppInfo{
			AppDetails:    util.GetAgentDetailStrings(app),
			AppName:       app.Name,
			AppID:         app.Metadata.ID,
			ExternalAppID: extAppID,
		},
		Quota: quota,
	}

	return q, nil
}

func intervalToProtoInterval(interval string) customunits.QuotaIntervalType {
	return map[string]customunits.QuotaIntervalType{
		provisioning.Daily.String():    customunits.QuotaIntervalType_IntervalDaily,
		provisioning.Weekly.String():   customunits.QuotaIntervalType_IntervalWeekly,
		provisioning.Monthly.String():  customunits.QuotaIntervalType_IntervalMonthly,
		provisioning.Annually.String(): customunits.QuotaIntervalType_IntervalAnnually,
	}[interval]
}

type reference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

func (h *CustomUnitHandler) getQuotaInfo(ar *management.AccessRequest) (string, string, int) {
	index := 0
	if len(ar.Spec.AdditionalQuotas) < index+1 {
		return "", "", 0
	}

	q := ar.Spec.AdditionalQuotas[index]
	for _, r := range ar.References {
		d, _ := json.Marshal(r)
		ref := &reference{}
		json.Unmarshal(d, ref)
		if ref.Kind == catalog.QuotaGVK().Kind && ref.Name == q.Name {
			return ref.Unit, q.Interval, int(q.Limit)
		}
	}
	return "", "", 0
}

func (h *CustomUnitHandler) getServiceInstance(ar *management.AccessRequest) (*v1.ResourceInstance, error) {
	instRef := ar.GetReferenceByGVK(management.APIServiceInstanceGVK())
	instID := instRef.ID
	instance, err := h.cache.GetAPIServiceInstanceByID(instID)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (m *CustomUnitHandler) HandleMetricReporting(metricCollector metricCollector) {
	if m.agentType != config.TraceabilityAgent {
		return
	}
	if len(m.servicesConfigs) > 0 {
		go m.receiveMetrics(metricCollector)
	}
	// iterate over each metric service config
	for _, config := range m.servicesConfigs {
		// Initialize custom units client
		factory := NewCustomUnitClientFactory(config.URL, &customunits.QuotaInfo{})
		client, _ := factory(m.cache)
		go client.StartMetricReporting(m.metricReportChan)
		m.clients = append(m.clients, client)
	}
}

func (c *CustomUnitHandler) receiveMetrics(metricCollector metricCollector) {
	for {
		select {
		case metricReport := <-c.metricReportChan:
			// create the metric report and send it to the metric collector
			c.handleMetricReport(metricReport, metricCollector)
		case <-c.stopChan:
			c.logger.Info("stopping to receive metric reports")
			for _, c := range c.clients {
				c.Stop()
			}
			return
		}
	}
}

func (c *CustomUnitHandler) handleMetricReport(metricReport *customunits.MetricReport, metricCollector metricCollector) {
	if metricReport == nil {
		return
	}

	logger := c.logger.WithField("api", metricReport.ApiService.GetValue()).WithField("app", metricReport.GetManagedApp().GetValue()).WithField("planUnit", metricReport.PlanUnit.GetUnitName())
	customMetricDetail := c.buildCustomMetricDetail(logger, metricReport)
	if customMetricDetail == nil {
		return
	}

	logger.Debug("collecting custom metric detail")
	metricCollector.AddCustomMetricDetail(*customMetricDetail)
}

func (c *CustomUnitHandler) buildCustomMetricDetail(logger log.FieldLogger, metricReport *customunits.MetricReport) *models.CustomMetricDetail {
	apiServiceLookup := metricReport.GetApiService()
	managedAppLookup := metricReport.GetManagedApp()
	planUnitLookup := metricReport.GetPlanUnit()

	planUnitDetails := c.PlanUnitLookup(logger, planUnitLookup)
	apiDetails := c.APIServiceLookup(logger, apiServiceLookup)
	appDetails := c.ManagedApplicationLookup(logger, managedAppLookup)

	if apiDetails == nil || appDetails == nil || planUnitDetails == nil {
		logger.Error("unable to build custom metric detail")
		return nil
	}

	return &models.CustomMetricDetail{
		APIDetails:  *apiDetails,
		AppDetails:  *appDetails,
		UnitDetails: *planUnitDetails,
		Count:       metricReport.Count,
	}
}

func (c *CustomUnitHandler) APIServiceLookup(logger log.FieldLogger, apiServiceLookup *customunits.APIServiceLookup) *models.APIDetails {
	apiSvcValue := apiServiceLookup.GetValue()
	apiLookupType := apiServiceLookup.GetType()

	if apiSvcValue == "" {
		logger.Error("not able to find api service lookup value")
		return nil
	}

	var apiSvc *v1.ResourceInstance
	switch apiLookupType {
	case customunits.APIServiceLookupType_CustomAPIServiceLookup:
		apiSvc = c.customAPILookup(apiSvcValue, apiServiceLookup.GetCustomAttribute())
	case customunits.APIServiceLookupType_ExternalAPIID:
		fallthrough
	case customunits.APIServiceLookupType_ServiceID:
		apiSvc = c.cache.GetAPIServiceWithAPIID(apiSvcValue)
	case customunits.APIServiceLookupType_ServiceName:
		apiSvc = c.cache.GetAPIServiceWithName(apiSvcValue)
	}

	if apiSvc == nil {
		return nil
	}

	id, err := util.GetAgentDetailsValue(apiSvc, definitions.AttrExternalAPIID)
	if err != nil {
		logger.WithError(err).Error("could not find external api id")
		return nil
	}

	return &models.APIDetails{
		ID:   transUtil.FormatProxyID(id),
		Name: apiSvc.Name,
	}
}

func (c *CustomUnitHandler) customAPILookup(apiSvcValue, apiCustomAttr string) *v1.ResourceInstance {
	if apiCustomAttr == "" {
		c.logger.Error("not able to lookup api service by custom attribute without the attribute name set")
		return nil
	}

	customKey := fmt.Sprintf("%s_%s", apiCustomAttr, apiSvcValue)
	if apiID, ok := c.apiCustomLookups[customKey]; ok {
		return c.cache.GetAPIServiceWithAPIID(apiID)
	}

	for _, key := range c.cache.GetAPIServiceKeys() {
		apisvc := c.cache.GetAPIServiceWithAPIID(key)
		val, _ := util.GetAgentDetailsValue(apisvc, apiCustomAttr)
		if val == apiSvcValue {
			c.apiCustomLookups[customKey] = apisvc.Metadata.ID
			return apisvc
		}
	}

	return nil
}

func (c *CustomUnitHandler) ManagedApplicationLookup(logger log.FieldLogger, appLookup *customunits.AppLookup) *models.AppDetails {
	appValue := appLookup.GetValue()
	appLookupType := appLookup.GetType()

	if appValue == "" {
		logger.Error("not able to find the app lookup value")
		return nil
	}

	var managedAppRI *v1.ResourceInstance
	switch appLookupType {
	case customunits.AppLookupType_ExternalAppID:
		managedAppRI = c.customAppLookup(appValue, definitions.AttrExternalAppID)
	case customunits.AppLookupType_CustomAppLookup:
		managedAppRI = c.customAppLookup(appValue, appLookup.GetCustomAttribute())
	case customunits.AppLookupType_ManagedAppID:
		managedAppRI = c.cache.GetManagedApplication(appValue)
	case customunits.AppLookupType_ManagedAppName:
		managedAppRI = c.cache.GetManagedApplicationByName(appValue)
	}

	if managedAppRI == nil {
		return nil
	}
	managedApp := &management.ManagedApplication{}
	err := managedApp.FromInstance(managedAppRI)
	if err != nil {
		log.Error("could not read managed application from cache")
		return nil
	}

	consumerOrgID := ""
	if managedApp.Marketplace.Resource.Owner != nil {
		consumerOrgID = managedApp.Marketplace.Resource.Owner.ID
	}

	return &models.AppDetails{
		ID:            managedApp.Metadata.ID,
		Name:          managedApp.Name,
		ConsumerOrgID: consumerOrgID,
	}
}

func (c *CustomUnitHandler) customAppLookup(appValue, appCustomAttr string) *v1.ResourceInstance {
	if appCustomAttr == "" {
		c.logger.Error("not able to lookup application by custom attribute without the attribute name set")
		return nil
	}

	customKey := fmt.Sprintf("%s_%s", appCustomAttr, appValue)
	if appID, ok := c.appCustomLookups[customKey]; ok {
		return c.cache.GetManagedApplication(appID)
	}

	for _, key := range c.cache.GetManagedApplicationCacheKeys() {
		app := c.cache.GetManagedApplication(key)
		val, _ := util.GetAgentDetailsValue(app, appCustomAttr)
		if val == appValue {
			c.appCustomLookups[customKey] = app.Metadata.ID
			return app
		}
	}

	return nil
}

func (c *CustomUnitHandler) PlanUnitLookup(logger log.FieldLogger, planUnitLookup *customunits.UnitLookup) *models.Unit {
	if planUnitLookup.GetUnitName() == "" {
		logger.Error("plan unit name required for lookups")
		return nil
	}

	return &models.Unit{
		Name: planUnitLookup.GetUnitName(),
	}
}

func (c *CustomUnitHandler) Stop() {
	c.stopChan <- struct{}{}
}
