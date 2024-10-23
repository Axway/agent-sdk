package customunit

import (
	"context"
	"fmt"
	"io"

	"github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	cu "github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CustomUnitsQEClient struct {
	ctx       context.Context
	quotaInfo *customunits.QuotaInfo
	logger    *logrus.Entry
	dialOpts  []grpc.DialOption
	cOpts     []grpc.CallOption
	url       string
	conn      *grpc.ClientConn
}

func NewQuotaEnforcementClient(ctx context.Context, url string, quotaInfo *customunits.QuotaInfo) CustomUnitsQEClient {
	return CustomUnitsQEClient{
		ctx:       ctx,
		quotaInfo: quotaInfo,
		url:       url,
		dialOpts: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
	}
}

func QuotaEnforcementInfo(metricServicesConfig []config.MetricServiceConfiguration, ctx context.Context, quotaInfo *customunits.QuotaInfo) string {
	errMessage := ""
	// iterate over each metric service config
	for _, config := range metricServicesConfig {

		if config.MetricServiceEnabled() {
			// Initialize custom units client
			c := NewQuotaEnforcementClient(ctx, config.URL, quotaInfo)

			_, err := c.GetQuotaEnforcementInfo()
			// if error from QE and reject on fail, we return the error back to the central
			if err != nil && config.RejectOnFailEnabled() {
				errMessage = errMessage + fmt.Sprintf("TODO: message: %s", err.Error())
			}
		}
	}

	return errMessage
}

func (c *CustomUnitsQEClient) GetQuotaEnforcementInfo() (*customunits.QuotaEnforcementResponse, error) {
	conn, err := grpc.DialContext(c.ctx, c.url, c.dialOpts...)
	if err != nil {
		return nil, err
	}
	quotaEnforcementClient := customunits.NewQuotaEnforcementClient(conn)

	response, err := quotaEnforcementClient.QuotaEnforcementInfo(c.ctx, c.quotaInfo, c.cOpts...)

	return response, err
}

type metricCollector interface {
	AddCustomMetricDetail()
}

type customUnitMetricReportingClient struct {
	ctx                          context.Context
	logger                       *logrus.Entry
	mtricReportingClient         cu.MetricReportingServiceClient
	metricReport                 *cu.MetricReport
	metricReportingServiceClient cu.MetricReportingService_MetricReportingClient
	dialOpts                     []grpc.DialOption
	cOpts                        []grpc.CallOption
	url                          string
	conn                         *grpc.ClientConn
	metricCollector              metricCollector
}

type CustomUnitMetricReportingClient interface {
	MetricReporting() error
	Close() error
}

func NewCustomUnitMetricReportingClient(ctx context.Context, url string) CustomUnitMetricReportingClient {
	// Initialize custom units client
	return &customUnitMetricReportingClient{
		ctx: ctx,
		url: url,
		dialOpts: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
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
		return err
	}
	metricServiceInit := &cu.MetricServiceInit{}

	client, err := c.mtricReportingClient.MetricReporting(c.ctx, metricServiceInit, c.cOpts...)
	if err != nil {
		return err
	}
	c.metricReportingServiceClient = client
	// process metrics
	c.processMetrics()
	return nil
}

// processMetrics will stream custom metrics
func (c *customUnitMetricReportingClient) processMetrics() {
	for {
		metricReport, err := c.recv()
		if err == io.EOF {
			c.logger.Debug("stream finished")
			continue
		}
		if err != nil {
			// if the connection fails, re-establish the connection
			c.MetricReporting()
		}

		c.reportMetrics(metricReport)

	}

}

func (c *customUnitMetricReportingClient) reportMetrics(*cu.MetricReport) {
	// TODO::// deprovision the metric report and send it to the metric collector
	c.metricCollector.AddCustomMetricDetail()
}

func (c *customUnitMetricReportingClient) recv() (*cu.MetricReport, error) {
	metricReport, err := c.metricReportingServiceClient.Recv()
	if err != nil {
		return nil, err
	}
	return metricReport, nil
}

func (c *customUnitMetricReportingClient) Close() error {
	var err error
	defer c.conn.Close()
	if err != nil {
		return err
	}

	return nil
}
