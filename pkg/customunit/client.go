package customunit

import (
	"context"
	"io"
	"sync"

	cu "github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

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
}

type CustomUnitMetricReportingClient interface {
	MetricReporting() error
	Close() error
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

	response, err := c.mtricReportingClient.MetricReporting(c.ctx, metricServiceInit, c.cOpts...)
	if err != nil {
		return err
	}
	c.metricReportingServiceClient = response
	// process metrics
	c.processMetrics()
	return nil
}

// processMetrics will stream custom metrics
func (c *customUnitMetricReportingClient) processMetrics() error {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	for {
		metricReport, err := c.recv()
		if err == io.EOF {
			c.logger.Debug("stream finished")
			return nil
		}
		if err != nil {
			// if the connection fails, re-establish the connection
			c.MetricReporting()
		}
		wg.Add(1)
		go func() {
			c.reportMetrics(metricReport)
			wg.Done()
		}()
	}
}

func (c *customUnitMetricReportingClient) reportMetrics(*cu.MetricReport) {
	// TODO::// deprovision the metric report and send it to the metric collector
	metric.GetMetricCollector().AddCustomMetricDetail()
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
