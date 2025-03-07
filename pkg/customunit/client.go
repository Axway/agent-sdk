package customunit

import (
	"context"
	"time"

	cu "github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type customUnitClient struct {
	logger    log.FieldLogger
	quotaInfo *cu.QuotaInfo
	dialOpts  []grpc.DialOption
	cOpts     []grpc.CallOption
	url       string
	conn      *grpc.ClientConn
	isRunning bool
	cache     agentCacheManager
	stopChan  chan struct{}
	delay     time.Duration
}

const maxRetryDelay = 5 * time.Minute
const initialRetryDelay = 30 * time.Second

type CustomUnitOption func(*customUnitClient)

type CustomUnitClientFactory func(agentCacheManager, ...CustomUnitOption) (*customUnitClient, error)

func NewCustomUnitClientFactory(url string, quotaInfo *cu.QuotaInfo) CustomUnitClientFactory {
	return func(cache agentCacheManager, opts ...CustomUnitOption) (*customUnitClient, error) {
		c := &customUnitClient{
			quotaInfo: quotaInfo,
			url:       url,
			dialOpts: []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			},
			cache:     cache,
			logger:    log.NewFieldLogger().WithPackage("customunit").WithComponent("client").WithField("metricServer", url),
			stopChan:  make(chan struct{}),
			isRunning: false,
			delay:     initialRetryDelay,
		}

		for _, o := range opts {
			o(c)
		}

		return c, nil
	}
}

func WithGRPCDialOption(opt grpc.DialOption) CustomUnitOption {
	return func(c *customUnitClient) {
		c.dialOpts = append(c.dialOpts, opt)
	}
}

func (c *customUnitClient) createConnection() error {
	conn, err := grpc.NewClient(c.url, c.dialOpts...)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *customUnitClient) QuotaEnforcementInfo() (*cu.QuotaEnforcementResponse, error) {
	err := c.createConnection()
	if err != nil {
		return nil, err
	}
	client := cu.NewQuotaEnforcementClient(c.conn)
	return client.QuotaEnforcementInfo(context.Background(), c.quotaInfo, c.cOpts...)
}

func (c *customUnitClient) StartMetricReporting(metricReportChan chan *cu.MetricReport) {
	c.isRunning = false
	for {
		err := c.createConnection()
		if err != nil {
			c.ExecuteBackoff()
			continue
		}

		client := cu.NewMetricReportingServiceClient(c.conn)

		stream, err := client.MetricReporting(context.Background(), &cu.MetricServiceInit{}, c.cOpts...)
		if err != nil {
			c.Close()
			c.ExecuteBackoff()
			continue
		}
		c.isRunning = true
		// reset the delay
		c.delay = initialRetryDelay
		// process metrics
		c.processMetrics(stream, metricReportChan)
		c.logger.Debug("connection lost, retrying to connect to metric server")
		c.Close()
	}
}

// processMetrics will stream custom metrics
func (c *customUnitClient) processMetrics(client cu.MetricReportingService_MetricReportingClient, metricReportChan chan *cu.MetricReport) {
	for {
		select {
		case <-c.stopChan:
			return
		default:
			metricReport, err := client.Recv()
			if err != nil {
				c.isRunning = false
				c.logger.WithError(err).Error(err.Error())
				return
			}
			metricReportChan <- metricReport
		}
	}

}

func (c *customUnitClient) ExecuteBackoff() {
	c.logger.WithField("retryDelay", c.delay).Debug("connection is still lost, trying again")
	time.Sleep(c.delay)
	c.delay = 2 * c.delay
	if c.delay >= maxRetryDelay {
		c.delay = maxRetryDelay
	}
}

func (c *customUnitClient) Close() {
	defer c.conn.Close()
}

func (c *customUnitClient) Stop() {
	if c.stopChan != nil {
		c.stopChan <- struct{}{}
	}
}
