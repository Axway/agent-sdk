package customunit

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	cu "github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type customUnitClient struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	logger    *logrus.Entry
	quotaInfo *cu.QuotaInfo
	dialOpts  []grpc.DialOption
	cOpts     []grpc.CallOption
	url       string
	conn      *grpc.ClientConn
	isRunning bool
	cache     cache.Manager
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
			logger:    logrus.NewEntry(log.Get()),
		}

		for _, o := range opts {
			o(c)
		}

		return *c, nil
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
	client := cu.NewQuotaEnforcementClient(c.conn)
	return client.QuotaEnforcementInfo(c.ctx, c.quotaInfo, c.cOpts...)
}

func (c *customUnitClient) MetricReporting(metricReportChan chan *cu.MetricReport) {
	if err := c.createConnection(); err != nil {
		//TODO:: Retry until the connection is stable
		return
	}
	client := cu.NewMetricReportingServiceClient(c.conn)

	stream, err := client.MetricReporting(c.ctx, &cu.MetricServiceInit{}, c.cOpts...)
	if err != nil {
		return
	}
	// process metrics
	c.processMetrics(stream, metricReportChan)
}

// processMetrics will stream custom metrics
func (c *customUnitClient) processMetrics(client cu.MetricReportingService_MetricReportingClient, metricReportChan chan *cu.MetricReport) {
	for {
		select {
		case <-c.ctx.Done():
			if c.isRunning {
				c.isRunning = false
				c.cancelCtx()
			}
			go c.MetricReporting(metricReportChan)
			return
		default:
			metricReport, err := client.Recv()
			if err != nil {
				c.logger.Debug("stream finished")
				c.Close()
				break
			}
			metricReportChan <- metricReport
		}
	}
}

func (c *customUnitClient) Close() error {
	var err error
	defer c.conn.Close()
	if err != nil {
		return err
	}

	return nil
}
