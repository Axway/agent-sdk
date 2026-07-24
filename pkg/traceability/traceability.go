package traceability

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

const (
	countStr     = "count"
	eventTypeStr = "event-type"
)

// OutputEventProcessor - processes events before they are published
type OutputEventProcessor interface {
	Process(events []event.Event) []event.Event
}

var outputEventProcessor OutputEventProcessor
var pathDataMutex sync.Mutex = sync.Mutex{}
var dataDirPath string

const (
	minWindowSize             int = 1
	defaultStartMaxWindowSize int = 10
	defaultPort                   = "443"
	tcpPort                       = "5044"
	traceabilityStr               = "traceability"
	HealthCheckEndpoint           = traceabilityStr
)

var traceabilityClients []*Client
var clientMutex *sync.Mutex
var traceCfg *Config

// GetClient - returns a random client from the clients array
var GetClient = getClient

func addClient(c *Client) {
	clientMutex.Lock()
	defer clientMutex.Unlock()
	traceabilityClients = append(traceabilityClients, c)
}

func getClient() (*Client, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()
	switch clients := len(traceabilityClients); clients {
	case 0:
		return nil, fmt.Errorf("no traceability clients, can't publish metrics")
	case 1:
		return traceabilityClients[0], nil
	default:
		randomIndex := rand.Intn(len(traceabilityClients))
		return traceabilityClients[randomIndex], nil
	}
}

// NetworkClient replaces libbeat's outputs.Client/outputs.NetworkClient.
type NetworkClient interface {
	Connect() error
	Close() error
	Publish(ctx context.Context, batch event.Batch) error
	String() string
}

// Client - struct
type Client struct {
	sync.Mutex
	transportClient NetworkClient
	logger          log.FieldLogger
}

func init() {
	clientMutex = &sync.Mutex{}
}

// SetOutputEventProcessor -
func SetOutputEventProcessor(eventProcessor OutputEventProcessor) {
	outputEventProcessor = eventProcessor
}

// GetDataDirPath - Returns the path of the data directory
func GetDataDirPath() string {
	pathDataMutex.Lock()
	defer pathDataMutex.Unlock()
	return dataDirPath
}

// SetDataDirPath - Sets the path of the data directory
func SetDataDirPath(path string) {
	pathDataMutex.Lock()
	defer pathDataMutex.Unlock()
	dataDirPath = path
}

// checkCreateDir
func createDirIfNotExist(dirPath string) {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		// Create the directory with the same permissions as the data dir
		dataInfo, _ := os.Stat(GetDataDirPath())
		os.MkdirAll(dirPath, dataInfo.Mode().Perm())
	}
}

// GetCacheDirPath - Returns the path of the cache directory
func GetCacheDirPath() string {
	cacheDir := path.Join(GetDataDirPath(), "cache")
	createDirIfNotExist(cacheDir)
	return cacheDir
}

// GetReportsDirPath - Returns the path of the reports directory
func GetReportsDirPath() string {
	reportDir := path.Join(GetDataDirPath(), "reports")
	createDirIfNotExist(reportDir)
	return reportDir
}

// NewClient replaces the libbeat outputs.RegisterType("traceability", makeTraceabilityAgent)
// factory-registration mechanism.
func NewClient(cfg *Config) ([]*Client, error) {
	logger := log.NewFieldLogger().
		WithPackage("sdk.traceability").
		WithComponent("NewClient")

	traceCfg = cfg
	outputConfig = cfg

	if err := cfg.ValidateCfg(); err != nil {
		agent.UpdateStatusWithPrevious(agent.AgentFailed, agent.AgentRunning, err.Error())
		logger.WithError(err).Error("validating config")
		return nil, err
	}

	validateProtocolPort()
	logger = logger.WithField("config", cfg).WithField("hosts", cfg.Hosts)
	logger.Tracef("initializing traceability client")

	isSingleEntry := agent.GetCentralConfig().GetSingleURL() != ""

	networkClients, err := makeHTTPClient(cfg, cfg.Hosts, agent.GetUserAgent(), isSingleEntry)
	if err != nil {
		agent.UpdateStatusWithPrevious(agent.AgentFailed, agent.AgentRunning, err.Error())
		logger.WithError(err).Error("creating traceability client")
		return nil, err
	}

	clients := make([]*Client, 0, len(networkClients))
	for _, nc := range networkClients {
		outputClient := &Client{
			transportClient: nc,
			logger:          logger.WithComponent("traceabilityClient").WithPackage("sdk.traceability"),
		}
		clients = append(clients, outputClient)
		addClient(outputClient)
	}

	if !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() && util.IsNotTest() {
		if err := registerHealthCheckers(cfg); err != nil {
			logger.WithError(err).Error("could not register healthcheck")
		}
	}

	return clients, nil
}

// validateProtocolPort - sets the protocol for single entry point hosts
func validateProtocolPort() {
	isSingleEntry := agent.GetCentralConfig().GetSingleURL() != ""
	if isSingleEntry {
		// get the expected protocol for single entry host
		traceCfg.Protocol = agent.GetCentralConfig().GetTraceabilityProtocol()
	}
}

// makeHTTPClient replaces libbeat's outputs.SuccessNet/outputs.NewFailoverClient/
// outputs.WithBackoff wrapping.
func makeHTTPClient(cfg *Config, hosts []string, userAgent string, isSingleEntry bool) ([]NetworkClient, error) {
	tlsCfg := cfg.TLS.toTLSConfiguration()

	clients := make([]NetworkClient, len(hosts))
	for i, host := range hosts {
		hostURL, err := buildURL(cfg.Protocol, host)
		if err != nil {
			return nil, err
		}
		proxyURL, err := url.Parse(cfg.Proxy.URL)
		if err != nil {
			return nil, err
		}
		client, err := NewHTTPClient(HTTPClientSettings{
			URL:              hostURL,
			Proxy:            proxyURL,
			TLS:              tlsCfg,
			Timeout:          cfg.Timeout,
			CompressionLevel: cfg.CompressionLevel,
			UserAgent:        userAgent,
			IsSingleEntry:    isSingleEntry,
		})
		if err != nil {
			return nil, err
		}
		clients[i] = withBackoff(client, cfg.Backoff.Init, cfg.Backoff.Max)
	}

	if !cfg.LoadBalance {
		return []NetworkClient{newFailoverClient(clients)}, nil
	}
	return clients, nil
}

// SetTransportClient - set the transport client
func (client *Client) SetTransportClient(outputClient NetworkClient) {
	client.Lock()
	defer client.Unlock()
	client.transportClient = outputClient
}

// getTransportClient - get the transport client
func (client *Client) getTransportClient() NetworkClient {
	client.Lock()
	defer client.Unlock()
	return client.transportClient
}

// SetLogger - set the logger
func (client *Client) SetLogger(logger log.FieldLogger) {
	client.logger = logger
}

// Connect establishes a connection to the clients sink.
func (client *Client) Connect() error {
	// do not attempt to establish a connection in offline mode
	if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		return nil
	}

	return client.getTransportClient().Connect()
}

// Close publish a single event to output.
func (client *Client) Close() error {
	// do not attempt to close a connection in offline mode, it was never established
	if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		return nil
	}

	return client.getTransportClient().Close()
}

// Publish sends events to the clients sink.
func (client *Client) Publish(ctx context.Context, batch event.Batch) error {
	events := batch.Events()
	if len(events) == 0 {
		batch.ACK()
		return nil // nothing to do
	}
	_, isMetric := events[0].Meta["metric"]

	if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		if outputEventProcessor != nil && !isMetric {
			outputEventProcessor.Process(events)
		}
		batch.ACK()
		return nil
	}

	logger := client.logger.WithField(eventTypeStr, "metric")

	if !isMetric {
		logger = logger.WithField(eventTypeStr, "transaction")
		if outputEventProcessor != nil {
			updatedEvents := outputEventProcessor.Process(events)
			batch.SetEvents(updatedEvents)
		}

		sampledEvents, err := sampling.FilterEvents(batch.Events())
		if err != nil {
			logger.Error(err.Error())
		}
		batch.SetEvents(sampledEvents)
	}

	events = batch.Events()
	if len(events) == 0 {
		batch.ACK()
		return nil // nothing to do
	}

	logger = logger.WithField(countStr, len(events))
	logger.Info("publishing events")

	err := client.getTransportClient().Publish(ctx, batch)
	if err != nil {
		logger.WithError(err).Error("failed to publish events")
		return err
	}

	logger.Info("published events")

	return nil
}

func (client *Client) String() string {
	return traceabilityStr
}

func registerHealthCheckers(config *Config) error {
	hcJob := newTraceabilityHealthCheckJob()

	_, err := jobs.RegisterIntervalJobWithName(hcJob, config.Timeout, "Traceability Health Check")
	if err != nil {
		return err
	}

	_, err = hc.RegisterHealthcheck("Traceability Agent", HealthCheckEndpoint, hcJob.healthcheck)
	if err != nil {
		return err
	}
	return nil
}
