package customunit

import (
	"context"
	"io"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	cu "github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type customUnitsQEClient struct {
	ctx       context.Context
	quotaInfo *cu.QuotaInfo
	dialOpts  []grpc.DialOption
	cOpts     []grpc.CallOption
	url       string
	conn      *grpc.ClientConn
	client    cu.QuotaEnforcementClient
}

type QEOption func(*customUnitsQEClient)

type QuotaEnforcementClientFactory func(context.Context, ...QEOption) (customUnitsQEClient, error)

func NewQuotaEnforcementClientFactory(url string, quotaInfo *cu.QuotaInfo) QuotaEnforcementClientFactory {
	return func(ctx context.Context, opts ...QEOption) (customUnitsQEClient, error) {
		c := &customUnitsQEClient{
			ctx:       ctx,
			quotaInfo: quotaInfo,
			url:       url,
			dialOpts: []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			},
		}

		for _, o := range opts {
			o(c)
		}

		return *c, nil
	}
}

func WithGRPCDialOption(opt grpc.DialOption) QEOption {
	return func(c *customUnitsQEClient) {
		c.dialOpts = append(c.dialOpts, opt)
	}
}

func (c *customUnitsQEClient) createConnection() error {
	conn, err := grpc.DialContext(c.ctx, c.url, c.dialOpts...)
	if err != nil {
		return err
	}
	c.conn = conn
	c.client = cu.NewQuotaEnforcementClient(conn)
	return nil
}

func (c *customUnitsQEClient) QuotaEnforcementInfo() (*cu.QuotaEnforcementResponse, error) {
	err := c.createConnection()
	if err != nil {
		// Handle the connection error
		return nil, err
	}
	response, err := c.client.QuotaEnforcementInfo(c.ctx, c.quotaInfo, c.cOpts...)

	return response, err
}

type metricCollector interface {
	AddCustomMetricDetail(models.CustomMetricDetail)
}

type customUnitMetricReportingClient struct {
	ctx                          context.Context
	cancelCtx                    context.CancelFunc
	logger                       *logrus.Entry
	mtricReportingClient         cu.MetricReportingServiceClient
	metricReportingServiceClient cu.MetricReportingService_MetricReportingClient
	dialOpts                     []grpc.DialOption
	cOpts                        []grpc.CallOption
	url                          string
	timer                        *time.Timer
	isRunning                    bool
	conn                         *grpc.ClientConn
	cache                        agentcache.Manager
	metricCollector              metricCollector
}

type MROption func(*customUnitMetricReportingClient)

type MetricReportingClientFactory func(context.Context, context.CancelFunc, ...MROption) (customUnitMetricReportingClient, error)

func NewCustomMetricReportingClientFactory(url string, agentCache cache.Manager) MetricReportingClientFactory {
	return func(ctx context.Context, ctxCancel context.CancelFunc, opts ...MROption) (customUnitMetricReportingClient, error) {
		c := &customUnitMetricReportingClient{
			ctx:       ctx,
			cancelCtx: ctxCancel,
			url:       url,
			cache:     agentCache,
			dialOpts: []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			},
			timer: time.NewTimer(time.Hour),
		}

		for _, o := range opts {
			o(c)
		}

		return *c, nil
	}
}

func WithGRPCDialOptionForMR(opt grpc.DialOption) MROption {
	return func(c *customUnitMetricReportingClient) {
		c.dialOpts = append(c.dialOpts, opt)
	}
}

func (c *customUnitMetricReportingClient) createConnection() error {
	// create the metric reporting connection
	conn, err := grpc.DialContext(c.ctx, c.url, c.dialOpts...)
	if err != nil {
		c.logger.WithError(err).Errorf("failed to connect to metric server")
		return err
	}
	c.conn = conn
	c.mtricReportingClient = cu.NewMetricReportingServiceClient(c.conn)
	return nil
}

func (c *customUnitMetricReportingClient) MetricReporting() error {
	if err := c.createConnection(); err != nil {
		//TODO:: Retry until the connection is stable
		return err
	}
	metricServiceInit := &cu.MetricServiceInit{}

	client, err := c.mtricReportingClient.MetricReporting(c.ctx, metricServiceInit, c.cOpts...)
	if err != nil {
		return err
	}
	c.metricReportingServiceClient = client
	// process metrics
	go c.processMetrics()
	return nil
}

// processMetrics will stream custom metrics
func (c *customUnitMetricReportingClient) processMetrics() {
	for {
		select {
		case <-c.ctx.Done():
			if c.isRunning {
				c.isRunning = false
				c.timer.Stop()
				c.cancelCtx()
			}
			c.MetricReporting()
		case <-c.metricReportingServiceClient.Context().Done():
			if c.isRunning {
				c.isRunning = false
				c.timer.Stop()
				c.cancelCtx()
			}
			c.MetricReporting()
		case <-c.timer.C:
			metricReport, err := c.metricReportingServiceClient.Recv()
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

func (c *customUnitMetricReportingClient) reportMetrics(metricReport *cu.MetricReport) {
	// deprovision the metric report and send it to the metric collector
	customMetricDetail, err := c.buildCustomMetricDetail(metricReport)
	if err == nil {
		c.metricCollector.AddCustomMetricDetail(*customMetricDetail)
	}
}

func (c *customUnitMetricReportingClient) buildCustomMetricDetail(metricReport *cu.MetricReport) (*models.CustomMetricDetail, error) {
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
	}, nil
}

func (c *customUnitMetricReportingClient) Close() error {
	var err error
	defer c.conn.Close()
	if err != nil {
		return err
	}

	return nil
}

func (c *customUnitMetricReportingClient) APIServiceLookup(apiServiceLookup *cu.APIServiceLookup) (*models.APIDetails, error) {
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
		apiSvc = c.cache.GetAPIServiceWithPrimaryKey(apiSvcValue)
	case cu.APIServiceLookupType_ServiceName:
		apiSvc = c.cache.GetAPIServiceWithName(apiSvcValue)
	}
	if apiSvc == nil {
		return nil, nil
	}

	return &models.APIDetails{
		ID:   apiSvc.Metadata.ID,
		Name: apiSvc.Name,
	}, nil
}

func (c *customUnitMetricReportingClient) ManagedApplicationLookup(appLookup *cu.AppLookup) (*models.AppDetails, error) {
	appValue := appLookup.GetValue()
	appLookupType := appLookup.GetType()
	appCustomAttr := appLookup.GetCustomAttribute()
	var managedApp *v1.ResourceInstance
	var err error

	if appLookupType == cu.AppLookupType_CustomAppLookup && appValue == "" {
		return nil, err
	}

	if appValue == "" {
		return nil, err
	}

	switch appLookupType {
	case cu.AppLookupType_CustomAppLookup:
		for _, key := range c.cache.GetAPIServiceKeys() {
			app := c.cache.GetManagedApplication(key)
			val, _ := util.GetAgentDetailsValue(app, appCustomAttr)
			if val == appValue {
				managedApp = app
				break
			}
		}
	case cu.AppLookupType_ExternalAppID:
		managedApp = c.cache.GetManagedApplication(appValue)
	case cu.AppLookupType_ManagedAppID:
		managedApp = c.cache.GetManagedApplication(appValue)
	case cu.AppLookupType_ManagedAppName:
		managedApp = c.cache.GetManagedApplicationByName(appValue)
	}
	if managedApp == nil {
		return nil, nil
	}
	return &models.AppDetails{
		ID:            managedApp.Metadata.ID,
		Name:          managedApp.Name,
		ConsumerOrgID: managedApp.Owner.ID,
	}, nil
}

func (c *customUnitMetricReportingClient) PlanUnitLookup(planUnitLookup *cu.UnitLookup) *models.Unit {

	return &models.Unit{
		Name: planUnitLookup.GetUnitName(),
	}
}
