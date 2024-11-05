package customunit

import (
	"context"
	"io"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	cu "github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type customUnitClient struct {
	ctx                          context.Context
	quotaInfo                    *cu.QuotaInfo
	dialOpts                     []grpc.DialOption
	cOpts                        []grpc.CallOption
	url                          string
	conn                         *grpc.ClientConn
	quotaEnforcementClient       cu.QuotaEnforcementClient
	cancelCtx                    context.CancelFunc
	logger                       *logrus.Entry
	metricReportingClient        cu.MetricReportingServiceClient
	metricReportingServiceClient cu.MetricReportingService_MetricReportingClient
	isRunning                    bool
	cache                        agentcache.Manager
	metricCollector              metricCollector
	stopChan                     chan bool
}

type CustomUnitOption func(*customUnitClient)

type CustomUnitClientFactory func(context.Context, context.CancelFunc, ...CustomUnitOption) (customUnitClient, error)

func NewCustomUnitClientFactory(url string, agentCache cache.Manager, quotaInfo *cu.QuotaInfo) CustomUnitClientFactory {
	return func(ctx context.Context, ctxCancel context.CancelFunc, opts ...CustomUnitOption) (customUnitClient, error) {
		c := &customUnitClient{
			ctx:       ctx,
			quotaInfo: quotaInfo,
			url:       url,
			dialOpts: []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			},
			cancelCtx: ctxCancel,
			cache:     agentCache,
			stopChan:  make(chan bool, 1),
		}

		for _, o := range opts {
			o(c)
		}

		return *c, nil
	}
}

func WithMetricCollector(collector metricCollector) CustomUnitOption {
	return func(c *customUnitClient) {
		c.metricCollector = collector
	}
}

func WithGRPCDialOption(opt grpc.DialOption) CustomUnitOption {
	return func(c *customUnitClient) {
		c.dialOpts = append(c.dialOpts, opt)
	}
}

func (c *customUnitClient) createConnection() error {
	conn, err := grpc.DialContext(c.ctx, c.url, c.dialOpts...)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *customUnitClient) QuotaEnforcementInfo() (*cu.QuotaEnforcementResponse, error) {
	err := c.createConnection()
	if err != nil {
		// Handle the connection error
		return nil, err
	}
	c.quotaEnforcementClient = cu.NewQuotaEnforcementClient(c.conn)
	response, err := c.quotaEnforcementClient.QuotaEnforcementInfo(c.ctx, c.quotaInfo, c.cOpts...)

	return response, err
}

type metricCollector interface {
	AddCustomMetricDetail(models.CustomMetricDetail)
}

func (c *customUnitClient) MetricReporting() {
	if err := c.createConnection(); err != nil {
		//TODO:: Retry until the connection is stable
		return
	}
	c.metricReportingClient = cu.NewMetricReportingServiceClient(c.conn)
	metricServiceInit := &cu.MetricServiceInit{}

	client, err := c.metricReportingClient.MetricReporting(c.ctx, metricServiceInit, c.cOpts...)
	if err != nil {
		return
	}
	c.metricReportingServiceClient = client
	// process metrics
	c.processMetrics()
}

// processMetrics will stream custom metrics
func (c *customUnitClient) processMetrics() {
	for {
		select {
		case <-c.ctx.Done():
			if c.isRunning {
				c.isRunning = false
				c.cancelCtx()
			}
			c.MetricReporting()
		default:
			metricReport, err := c.recv()
			if err == io.EOF {
				c.logger.Debug("stream finished")
				break
			}
			if err != nil {
				break
			}
			c.reportMetrics(metricReport)
		}
	}
}
func (c *customUnitClient) recv() (*cu.MetricReport, error) {
	for {
		metricReport, err := c.metricReportingServiceClient.Recv()
		if err != nil {
			return nil, err
		}

		return metricReport, nil
	}
}

func (c *customUnitClient) reportMetrics(metricReport *cu.MetricReport) {
	// deprovision the metric report and send it to the metric collector
	customMetricDetail, err := c.buildCustomMetricDetail(metricReport)
	if err == nil {
		c.metricCollector.AddCustomMetricDetail(*customMetricDetail)
	}
}

func (c *customUnitClient) buildCustomMetricDetail(metricReport *cu.MetricReport) (*models.CustomMetricDetail, error) {
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

	return &models.CustomMetricDetail{
		APIDetails:  *apiDetails,
		AppDetails:  *appDetails,
		UnitDetails: *planUnitDetails,
		Count:       metricReport.Count,
	}, nil
}

func (c *customUnitClient) Close() error {
	var err error
	defer c.conn.Close()
	if err != nil {
		return err
	}

	return nil
}

func (c *customUnitClient) APIServiceLookup(apiServiceLookup *cu.APIServiceLookup) (*models.APIDetails, error) {
	apiSvcValue := apiServiceLookup.GetValue()
	apiLookupType := apiServiceLookup.GetType()
	apiCustomAttr := apiServiceLookup.GetCustomAttribute()
	var apiSvc *v1.ResourceInstance
	var err error

	if apiLookupType == cu.APIServiceLookupType_CustomAPIServiceLookup && apiCustomAttr == "" {
		return nil, err
	}

	if apiSvcValue == "" {
		return nil, err
	}

	switch apiLookupType {
	case cu.APIServiceLookupType_CustomAPIServiceLookup:
		for _, key := range c.cache.GetAPIServiceKeys() {
			apisvc := c.cache.GetAPIServiceWithAPIID(key)
			val, _ := util.GetAgentDetailsValue(apisvc, apiCustomAttr)
			if val == apiSvcValue {
				apiSvc = apisvc
				break
			}
		}
	case cu.APIServiceLookupType_ExternalAPIID:
		apiSvc = c.cache.GetAPIServiceWithAPIID(apiSvcValue)
	case cu.APIServiceLookupType_ServiceID:
		apiSvc = c.cache.GetAPIServiceWithAPIID(apiSvcValue)
	case cu.APIServiceLookupType_ServiceName:
		apiSvc = c.cache.GetAPIServiceWithName(apiSvcValue)
	}
	if apiSvc == nil {
		return nil, nil
	}

	id, _ := util.GetAgentDetailsValue(apiSvc, definitions.AttrExternalAPIID) //TODO err handle

	return &models.APIDetails{
		ID:   id,
		Name: apiSvc.Name,
	}, nil
}

func (c *customUnitClient) ManagedApplicationLookup(appLookup *cu.AppLookup) (*models.AppDetails, error) {
	appValue := appLookup.GetValue()
	appLookupType := appLookup.GetType()
	appCustomAttr := appLookup.GetCustomAttribute()
	var managedAppRI *v1.ResourceInstance
	var err error

	if appLookupType == cu.AppLookupType_CustomAppLookup && appValue == "" {
		return nil, err
	}

	if appValue == "" {
		return nil, err
	}

	switch appLookupType {
	case cu.AppLookupType_ExternalAppID:
		appCustomAttr = definitions.AttrExternalAPIID
		fallthrough
	case cu.AppLookupType_CustomAppLookup:
		for _, key := range c.cache.GetAPIServiceKeys() {
			app := c.cache.GetManagedApplication(key)
			val, _ := util.GetAgentDetailsValue(app, appCustomAttr)
			if val == appValue {
				managedAppRI = app
				break
			}
		}
	case cu.AppLookupType_ManagedAppID:
		managedAppRI = c.cache.GetManagedApplication(appValue)
	case cu.AppLookupType_ManagedAppName:
		managedAppRI = c.cache.GetManagedApplicationByName(appValue)
	}
	if managedAppRI == nil {
		return nil, nil
	}
	managedApp := &management.ManagedApplication{}
	managedApp.FromInstance(managedAppRI) //TODO err handle

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

func (c *customUnitClient) PlanUnitLookup(planUnitLookup *cu.UnitLookup) *models.Unit {
	return &models.Unit{
		Name: planUnitLookup.GetUnitName(),
	}
}
