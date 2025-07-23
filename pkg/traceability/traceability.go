package traceability

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"os"
	"path"
	"reflect"
	"sync"
	"unsafe"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"golang.org/x/net/proxy"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

const (
	countStr     = "count"
	eventTypeStr = "event-type"
)

// OutputEventProcessor - P
type OutputEventProcessor interface {
	Process(events []publisher.Event) []publisher.Event
}

var outputEventProcessor OutputEventProcessor
var pathDataMutex sync.Mutex = sync.Mutex{}

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

// Client - struct
type Client struct {
	sync.Mutex
	transportClient outputs.Client
	logger          log.FieldLogger
}

func init() {
	clientMutex = &sync.Mutex{}
	outputs.RegisterType(traceabilityStr, makeTraceabilityAgent)
}

// SetOutputEventProcessor -
func SetOutputEventProcessor(eventProcessor OutputEventProcessor) {
	outputEventProcessor = eventProcessor
}

// GetDataDirPath - Returns the path of the data directory
func GetDataDirPath() string {
	pathDataMutex.Lock()
	defer pathDataMutex.Unlock()
	return paths.Paths.Data
}

// SetDataDirPath - Sets the path of the data directory
func SetDataDirPath(path string) {
	pathDataMutex.Lock()
	defer pathDataMutex.Unlock()
	paths.Paths.Data = path
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

func makeTraceabilityAgent(
	indexManager outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	libbeatCfg *common.Config,
) (outputs.Group, error) {
	logger := log.NewFieldLogger().
		WithPackage("sdk.traceability").
		WithComponent("makeTraceabilityAgent")

	var err error

	logger.Trace("reading config")
	traceCfg, err = readConfig(libbeatCfg, beat)
	if err != nil {
		agent.UpdateStatusWithPrevious(agent.AgentFailed, agent.AgentRunning, err.Error())
		logger.WithError(err).Error("reading config")
		return outputs.Fail(err)
	}

	defer func() {
		if err != nil {
			// skip hc register if err hit making agent
			return
		}

		if !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() && util.IsNotTest() {
			err := registerHealthCheckers(traceCfg)
			if err != nil {
				logger.WithError(err).Error("could not register healthcheck")
			}
		}
	}()

	validateProtocolPort()
	logger = logger.WithField("config", traceCfg)

	if err := libbeatCfg.Merge(HostConfig{Hosts: traceCfg.Hosts, Protocol: traceCfg.Protocol}); err != nil {
		agent.UpdateStatusWithPrevious(agent.AgentFailed, agent.AgentRunning, err.Error())
		logger.WithError(err).Error("merging host config")
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(libbeatCfg)
	if err != nil {
		agent.UpdateStatusWithPrevious(agent.AgentFailed, agent.AgentRunning, err.Error())
		logger.WithError(err).Error("reading hosts")
		return outputs.Fail(err)
	}

	logger = logger.WithField("hosts", hosts).WithField("config", traceCfg)
	logger.Tracef("initializing traceability client")

	var transportGroup outputs.Group
	isSingleEntry := agent.GetCentralConfig().GetSingleURL() != ""

	if IsTCPTransport() {
		// For Single entry point register dialer factory for sni scheme and set the
		// proxy url with sni scheme. When libbeat will register its dialer and sees
		// proxy url with sni scheme, it will invoke the factory to construct the dialer
		// The dialer will be invoked as proxy dialer in the libbeat dialer chain
		// (proxy dialer, stat dialer, tls dialer).
		if isSingleEntry {
			// Register dialer factory with sni scheme for single entry point
			proxy.RegisterDialerType("sni", ingestionSingleEntryDialer)
			// If real proxy configured(not the sni proxy set here), validate the scheme
			// since libbeats proxy dialer will not be invoked.
			if traceCfg.Proxy.URL != "" {
				proxCfg := &transport.ProxyConfig{
					URL:          traceCfg.Proxy.URL,
					LocalResolve: traceCfg.Proxy.LocalResolve,
				}
				err := proxCfg.Validate()
				if err != nil {
					logger.WithError(err).Error("validating proxy config")
					outputs.Fail(err)
				}
			}
			// Replace the proxy URL to sni by setting the environment variable
			// Libbeat parses the yaml file and replaces the value from yaml
			// with overridden environment variable.
			// Set the sni host to the ingestion service host to allow the
			// single entry dialer to receive the target address
			os.Setenv("TRACEABILITY_PROXYURL", "sni://"+traceCfg.Hosts[0])
		}

		transportGroup, err = makeLogstashClient(indexManager, beat, observer, libbeatCfg)
	} else {
		transportGroup, err = makeHTTPClient(beat, observer, traceCfg, hosts, agent.GetUserAgent(), isSingleEntry)
	}

	if err != nil {
		logger.WithError(err).Error("creating traceability client")
		return outputs.Fail(err)
	}

	traceabilityGroup := outputs.Group{
		BatchSize: transportGroup.BatchSize,
		Retry:     transportGroup.Retry,
	}
	clients := make([]outputs.Client, 0)

	for _, client := range transportGroup.Clients {
		outputClient := &Client{
			transportClient: client,
			logger:          logger.WithComponent("traceabilityClient").WithPackage("sdk.traceability"),
		}
		clients = append(clients, outputClient)
		addClient(outputClient)
	}
	traceabilityGroup.Clients = clients
	return traceabilityGroup, nil
}

// validateProtocolPort - validate the protocol matches the port
func validateProtocolPort() {
	isSingleEntry := agent.GetCentralConfig().GetSingleURL() != ""
	if isSingleEntry {
		// get the expected protocol for single entry host
		traceCfg.Protocol = agent.GetCentralConfig().GetTraceabilityProtocol()
	}

	// Validate that the port matches the
	if len(traceCfg.Hosts) == 0 {
		return
	}
	h, p := splitHostPort()
	if p == tcpPort && IsHTTPTransport() {
		traceCfg.Hosts[0] = fmt.Sprintf("%s:%s", h, defaultPort)
	} else if p == defaultPort && IsTCPTransport() {
		traceCfg.Hosts[0] = fmt.Sprintf("%s:%s", h, tcpPort)
	}
}

func makeLogstashClient(indexManager outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	libbeatCfg *common.Config,
) (outputs.Group, error) {
	factory := outputs.FindFactory("logstash")
	if factory == nil {
		return outputs.Group{}, nil
	}
	group, err := factory(indexManager, beat, observer, libbeatCfg)
	return group, err
}

// Factory method for creating dialer for sni scheme
// Setup the single entry point dialer with single entry host mapping based
// on central config and traceability proxy url from original config that gets
// read by traceability output factory(makeTraceabilityAgent)
func ingestionSingleEntryDialer(proxyURL *url.URL, parentDialer proxy.Dialer) (proxy.Dialer, error) {
	var traceProxyURL *url.URL
	var err error
	if traceCfg != nil && traceCfg.Proxy.URL != "" {
		traceProxyURL, err = url.Parse(traceCfg.Proxy.URL)
		if err != nil {
			return nil, fmt.Errorf("proxy could not be parsed. %s", err.Error())
		}
	}
	var singleEntryHostMap map[string]string
	if agent.GetCentralConfig() != nil {
		cfgSingleURL := agent.GetCentralConfig().GetSingleURL()
		if cfgSingleURL != "" {
			// cfgSingleURL should not be empty as the factory method is registered based on that check
			singleEntryURL, err := url.Parse(cfgSingleURL)
			if err == nil && traceCfg != nil {
				singleEntryHostMap = map[string]string{
					traceCfg.Hosts[0]: util.ParseAddr(singleEntryURL),
				}
			}
		}
	}

	dialer := util.NewDialer(traceProxyURL, singleEntryHostMap)
	return dialer, nil
}

func makeHTTPClient(beat beat.Info, observer outputs.Observer, traceCfg *Config, hosts []string, userAgent string, isSingleEntry bool) (outputs.Group, error) {
	tls, err := tlscommon.LoadTLSConfig(traceCfg.TLS)
	if err != nil {
		agent.UpdateStatusWithPrevious(agent.AgentFailed, agent.AgentRunning, err.Error())
		return outputs.Fail(err)
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		hostURL, err := common.MakeURL(traceCfg.Protocol, "/", host, 443)
		if err != nil {
			return outputs.Fail(err)
		}
		proxyURL, err := url.Parse(traceCfg.Proxy.URL)
		if err != nil {
			return outputs.Fail(err)
		}
		var client outputs.NetworkClient
		client, err = NewHTTPClient(HTTPClientSettings{
			BeatInfo:         beat,
			URL:              hostURL,
			Proxy:            proxyURL,
			TLS:              tls,
			Timeout:          traceCfg.Timeout,
			CompressionLevel: traceCfg.CompressionLevel,
			Observer:         observer,
			UserAgent:        userAgent,
			IsSingleEntry:    isSingleEntry,
		})

		if err != nil {
			return outputs.Fail(err)
		}
		client = outputs.WithBackoff(client, traceCfg.Backoff.Init, traceCfg.Backoff.Max)
		clients[i] = client
	}

	return outputs.SuccessNet(traceCfg.LoadBalance, traceCfg.BulkMaxSize, traceCfg.MaxRetries, clients)
}

// SetTransportClient - set the transport client
func (client *Client) SetTransportClient(outputClient outputs.Client) {
	client.Lock()
	defer client.Unlock()
	client.transportClient = outputClient
}

// SetTransportClient - set the transport client
func (client *Client) getTransportClient() outputs.Client {
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

	networkClient := client.getTransportClient().(outputs.NetworkClient)
	err := networkClient.Connect()
	if err != nil {
		return err
	}
	return nil
}

// Close publish a single event to output.
func (client *Client) Close() error {
	// do not attempt to close a connection in offline mode, it was never established
	if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		return nil
	}

	err := client.getTransportClient().Close()
	if err != nil {
		return err
	}
	return nil
}

// Publish sends events to the clients sink.
func (client *Client) Publish(ctx context.Context, batch publisher.Batch) error {
	events := batch.Events()
	if len(events) == 0 {
		batch.ACK()
		return nil // nothing to do
	}
	_, isMetric := events[0].Content.Meta["metric"]

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
			updateEvent(batch, updatedEvents)
		}

		sampledEvents, err := sampling.FilterEvents(batch.Events())
		if err != nil {
			logger.Error(err.Error())
		}
		updateEvent(batch, sampledEvents)
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

// updateEvent - updates the private field events in publisher.Batch
func updateEvent(batch publisher.Batch, events []publisher.Event) {
	pointerVal := reflect.ValueOf(batch)
	val := reflect.Indirect(pointerVal)

	member := val.FieldByName("events")
	ptrToEvents := unsafe.Pointer(member.UnsafeAddr())
	realPtrToEvents := (*[]publisher.Event)(ptrToEvents)
	*realPtrToEvents = events
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

func splitHostPort() (string, string) {
	// Split the host and port from the URL
	if len(traceCfg.Hosts) == 0 {
		return "", fmt.Sprint(defaultPort)
	}
	host, port, err := net.SplitHostPort(traceCfg.Hosts[0])
	if err != nil {
		return "", fmt.Sprint(defaultPort)
	}
	return host, port
}
