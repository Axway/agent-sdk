package customunit

import (
	"encoding/json"
	"errors"
	"fmt"

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

	errMessage := ""
	for _, config := range h.servicesConfigs {
		logger := logger.WithField("url", config.URL)
		// if error from QE and reject on fail, we return the error back to the central
		msg := h.handleServiceQE(config, quotaInfo)
		if msg != "" {
			logger.WithField("err", msg).Error("error handling provisioning")
			msg = fmt.Sprintf("service (url: %s, message: %s)", config.URL, msg)
		}
		if !config.RejectOnFail || msg == "" {
			continue // if there was an error do not add it to the overall errMessage
		}

		// add it to the overall errMessage since reject on fail enabled
		if errMessage == "" {
			errMessage = msg
			continue
		}
		// append to existing errMessage if multiple services failed
		errMessage = fmt.Sprintf("%s; %s", errMessage, msg)
	}

	if errMessage != "" {
		err = errors.New(errMessage)
		logger.WithError(err).Error("received back from metric services for quota enforcement")
		return err
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
			if metricReport == nil {
				continue
			}
			logger := c.logger.WithField("api", metricReport.ApiService.GetValue()).WithField("app", metricReport.GetManagedApp().GetValue()).WithField("planUnit", metricReport.PlanUnit.GetUnitName())
			customMetricDetail, err := c.buildCustomMetricDetail(metricReport)
			if err != nil {
				logger.WithError(err).Error("could not build metric data")
				continue
			}
			// create the metric report and send it to the metric collector
			logger.Debug("collecting custom metric detail")
			metricCollector.AddCustomMetricDetail(*customMetricDetail)
		case <-c.stopChan:
			c.logger.Info("stopping to receive metric reports")
			for _, c := range c.clients {
				c.Stop()
			}
			return
		}
	}
}

func (c *CustomUnitHandler) buildCustomMetricDetail(metricReport *customunits.MetricReport) (*models.CustomMetricDetail, error) {
	apiServiceLookup := metricReport.GetApiService()
	managedAppLookup := metricReport.GetManagedApp()
	planUnitLookup := metricReport.GetPlanUnit()

	apiDetails, err := c.APIServiceLookup(apiServiceLookup)
	if err != nil {
		c.logger.Error(err)
		return nil, err
	}
	appDetails, err := c.ManagedApplicationLookup(managedAppLookup)
	if err != nil {
		c.logger.Error(err)
		return nil, err
	}

	planUnitDetails := c.PlanUnitLookup(planUnitLookup)

	if apiDetails == nil || appDetails == nil || planUnitDetails == nil {
		return nil, fmt.Errorf("unable to build custom metric detail")
	}

	return &models.CustomMetricDetail{
		APIDetails:  *apiDetails,
		AppDetails:  *appDetails,
		UnitDetails: *planUnitDetails,
		Count:       metricReport.Count,
	}, nil
}

func (c *CustomUnitHandler) APIServiceLookup(apiServiceLookup *customunits.APIServiceLookup) (*models.APIDetails, error) {
	apiSvcValue := apiServiceLookup.GetValue()
	apiLookupType := apiServiceLookup.GetType()
	apiCustomAttr := apiServiceLookup.GetCustomAttribute()
	var apiSvc *v1.ResourceInstance
	var err error

	if apiLookupType == customunits.APIServiceLookupType_CustomAPIServiceLookup && apiCustomAttr == "" {
		return nil, err
	}

	if apiSvcValue == "" {
		return nil, err
	}

	switch apiLookupType {
	case customunits.APIServiceLookupType_CustomAPIServiceLookup:
		for _, key := range c.cache.GetAPIServiceKeys() {
			apisvc := c.cache.GetAPIServiceWithAPIID(key)
			val, _ := util.GetAgentDetailsValue(apisvc, apiCustomAttr)
			if val == apiSvcValue {
				apiSvc = apisvc
				break
			}
		}
	case customunits.APIServiceLookupType_ExternalAPIID:
		apiSvc = c.cache.GetAPIServiceWithAPIID(apiSvcValue)
	case customunits.APIServiceLookupType_ServiceID:
		apiSvc = c.cache.GetAPIServiceWithAPIID(apiSvcValue)
	case customunits.APIServiceLookupType_ServiceName:
		apiSvc = c.cache.GetAPIServiceWithName(apiSvcValue)
	}
	if apiSvc == nil {
		return nil, nil
	}

	id, err := util.GetAgentDetailsValue(apiSvc, definitions.AttrExternalAPIID)
	if err != nil {
		return nil, err
	}

	return &models.APIDetails{
		ID:   transUtil.FormatProxyID(id),
		Name: apiSvc.Name,
	}, nil
}

func (c *CustomUnitHandler) ManagedApplicationLookup(appLookup *customunits.AppLookup) (*models.AppDetails, error) {
	appValue := appLookup.GetValue()
	appLookupType := appLookup.GetType()
	appCustomAttr := appLookup.GetCustomAttribute()
	var managedAppRI *v1.ResourceInstance
	var err error

	if appLookupType == customunits.AppLookupType_CustomAppLookup && appValue == "" {
		return nil, err
	}

	if appValue == "" {
		return nil, err
	}

	switch appLookupType {
	case customunits.AppLookupType_ExternalAppID:
		appCustomAttr = definitions.AttrExternalAppID
		fallthrough
	case customunits.AppLookupType_CustomAppLookup:
		for _, key := range c.cache.GetManagedApplicationCacheKeys() {
			app := c.cache.GetManagedApplication(key)
			val, _ := util.GetAgentDetailsValue(app, appCustomAttr)
			if val == appValue {
				managedAppRI = app
				break
			}
		}
	case customunits.AppLookupType_ManagedAppID:
		managedAppRI = c.cache.GetManagedApplication(appValue)
	case customunits.AppLookupType_ManagedAppName:
		managedAppRI = c.cache.GetManagedApplicationByName(appValue)
	}
	if managedAppRI == nil {
		return nil, nil
	}
	managedApp := &management.ManagedApplication{}
	err = managedApp.FromInstance(managedAppRI)
	if err != nil {
		return nil, err
	}
	consumerOrgID := ""
	if managedApp.Marketplace.Resource.Owner != nil {
		consumerOrgID = managedApp.Marketplace.Resource.Owner.ID
	}

	return &models.AppDetails{
		ID:            managedApp.Metadata.ID,
		Name:          managedApp.Name,
		ConsumerOrgID: consumerOrgID,
	}, nil
}

func (c *CustomUnitHandler) PlanUnitLookup(planUnitLookup *customunits.UnitLookup) *models.Unit {
	return &models.Unit{
		Name: planUnitLookup.GetUnitName(),
	}
}

func (c *CustomUnitHandler) Stop() {
	c.stopChan <- struct{}{}
}
