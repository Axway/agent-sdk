package customunit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transUtil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type CustomUnitMetricServerManager struct {
	configs          []config.MetricServiceConfiguration
	cache            cache.Manager
	agentType        config.AgentType
	logger           log.FieldLogger
	clients          []*customUnitClient
	metricReportChan chan *customunits.MetricReport
	stopChan         chan struct{}
}

type metricCollector interface {
	AddCustomMetricDetail(models.CustomMetricDetail)
}

func NewCustomUnitMetricServerManager(configs []config.MetricServiceConfiguration, cache cache.Manager, agentType config.AgentType) *CustomUnitMetricServerManager {
	return &CustomUnitMetricServerManager{
		configs:          configs,
		cache:            cache,
		agentType:        agentType,
		metricReportChan: make(chan *customunits.MetricReport, 100),
		stopChan:         make(chan struct{}),
		clients:          []*customUnitClient{},
		logger:           log.NewFieldLogger().WithPackage("customunit").WithComponent("manager"),
	}
}

func (h *CustomUnitMetricServerManager) HandleQuotaEnforcement(ctx context.Context, cancelCtx context.CancelFunc, ar *management.AccessRequest, app *management.ManagedApplication) error {
	// Build quota info
	quotaInfo, err := h.buildQuotaInfo(ctx, ar, app)
	if err != nil {
		return fmt.Errorf("could not build quota info from access request")
	}
	errMessage := ""
	for _, config := range h.configs {
		if config.MetricServiceEnabled() {
			factory := NewCustomUnitClientFactory(config.URL, h.cache, quotaInfo)
			client, _ := factory()
			response, err := client.QuotaEnforcementInfo()
			if err != nil {
				// if error from QE and reject on fail, we return the error back to the central
				if response != nil && response.Error != "" && config.RejectOnFailEnabled() {
					errMessage = errMessage + fmt.Sprintf("TODO: message: %s", err.Error())
				}
			}
		}
	}

	if errMessage != "" {
		return errors.New(errMessage)
	}
	return nil
}

func (h *CustomUnitMetricServerManager) buildQuotaInfo(ctx context.Context, ar *management.AccessRequest, app *management.ManagedApplication) (*customunits.QuotaInfo, error) {
	unitRef, count := h.getQuotaInfo(ar)
	if unitRef == "" {
		return nil, nil
	}

	instance, err := h.getServiceInstance(ctx, ar)
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
		return nil, err
	}

	q := &customunits.QuotaInfo{
		ApiInfo: &customunits.APIInfo{
			ServiceDetails: util.GetAgentDetailStrings(service),
			ServiceName:    service.Name,
			ServiceID:      service.Metadata.ID,
			ExternalAPIID:  extAPIID,
		},
		AppInfo: &customunits.AppInfo{
			AppDetails: util.GetAgentDetailStrings(app),
			AppName:    app.Name,
			AppID:      app.Metadata.ID,
		},
		Quota: &customunits.Quota{
			Count: int64(count),
			Unit:  unitRef,
		},
	}

	return q, nil
}

type reference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

func (h *CustomUnitMetricServerManager) getQuotaInfo(ar *management.AccessRequest) (string, int) {
	index := 0
	if len(ar.Spec.AdditionalQuotas) < index+1 {
		return "", 0
	}

	q := ar.Spec.AdditionalQuotas[index]
	for _, r := range ar.References {
		d, _ := json.Marshal(r)
		ref := &reference{}
		json.Unmarshal(d, ref)
		if ref.Kind == catalog.QuotaGVK().Kind && ref.Name == q.Name {
			return ref.Unit, int(q.Limit)
		}
	}
	return "", 0
}

func (h *CustomUnitMetricServerManager) getServiceInstance(_ context.Context, ar *management.AccessRequest) (*v1.ResourceInstance, error) {
	instRef := ar.GetReferenceByGVK(management.APIServiceInstanceGVK())
	instID := instRef.ID
	instance, err := h.cache.GetAPIServiceInstanceByID(instID)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (m *CustomUnitMetricServerManager) HandleMetricReporting(ctx context.Context, cancelCtx context.CancelFunc, metricCollector metricCollector) {
	if m.agentType != config.TraceabilityAgent {
		return
	}
	go m.receiveMetrics(metricCollector)
	// iterate over each metric service config
	for _, config := range m.configs {
		// Initialize custom units client
		factory := NewCustomUnitClientFactory(config.URL, m.cache, &customunits.QuotaInfo{})
		client, _ := factory()
		go client.StartMetricReporting(m.metricReportChan)
		m.clients = append(m.clients, client)
	}
}

func (c *CustomUnitMetricServerManager) receiveMetrics(metricCollector metricCollector) {
	for {
		select {
		case metricReport := <-c.metricReportChan:
			if metricReport == nil {
				continue
			}
			logger := c.logger.WithField("api", metricReport.ApiService.GetValue())
			customMetricDetail, err := c.buildCustomMetricDetail(metricReport)
			if err != nil {
				logger.Error(err)
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

func (c *CustomUnitMetricServerManager) buildCustomMetricDetail(metricReport *customunits.MetricReport) (*models.CustomMetricDetail, error) {
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

func (c *CustomUnitMetricServerManager) APIServiceLookup(apiServiceLookup *customunits.APIServiceLookup) (*models.APIDetails, error) {
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

func (c *CustomUnitMetricServerManager) ManagedApplicationLookup(appLookup *customunits.AppLookup) (*models.AppDetails, error) {
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
		appCustomAttr = definitions.AttrExternalAPIID
		fallthrough
	case customunits.AppLookupType_CustomAppLookup:
		for _, key := range c.cache.GetAPIServiceKeys() {
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

func (c *CustomUnitMetricServerManager) PlanUnitLookup(planUnitLookup *customunits.UnitLookup) *models.Unit {
	return &models.Unit{
		Name: planUnitLookup.GetUnitName(),
	}
}

func (c *CustomUnitMetricServerManager) Stop() {
	c.stopChan <- struct{}{}
}
