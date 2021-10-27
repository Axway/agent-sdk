package traceability

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path"
	"reflect"
	"time"
	"unsafe"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/publisher"

	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

// OutputEventProcessor - P
type OutputEventProcessor interface {
	Process(events []publisher.Event) []publisher.Event
}

var outputEventProcessor OutputEventProcessor

const (
	minWindowSize             int = 1
	defaultStartMaxWindowSize int = 10
	defaultPort                   = 5044
	traceabilityStr               = "traceability"
)

var traceabilityClients []*Client

// GetClient - returns a random client from the clients array
var GetClient = getClient

func getClient() (*Client, error) {
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
	transportClient outputs.Client
}

type traceabilityAgentHealthChecker struct {
	protocol string
	host     string
	proxyURL string
	timeout  time.Duration
	// TBD. Remove in future when Jobs interface is complete
	hcJob *condorHealthCheckJob
}

func init() {
	outputs.RegisterType(traceabilityStr, makeTraceabilityAgent)
}

// SetOutputEventProcessor -
func SetOutputEventProcessor(eventProcessor OutputEventProcessor) {
	outputEventProcessor = eventProcessor
}

// GetDataDirPath - Returns the path of the data directory
func GetDataDirPath() string {
	return paths.Paths.Data
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
	traceCfg, err := readConfig(libbeatCfg, beat)
	if err != nil {
		agent.UpdateStatusWithPrevious(agent.AgentFailed, agent.AgentRunning, err.Error())
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(libbeatCfg)
	if err != nil {
		agent.UpdateStatusWithPrevious(agent.AgentFailed, agent.AgentRunning, err.Error())
		return outputs.Fail(err)
	}

	var transportGroup outputs.Group
	if traceCfg.Protocol == "https" || traceCfg.Protocol == "http" {
		transportGroup, err = makeHTTPClient(beat, observer, traceCfg, hosts)
	} else {
		transportGroup, err = makeLogstashClient(indexManager, beat, observer, libbeatCfg, traceCfg)
	}

	if err != nil {
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
		}
		clients = append(clients, outputClient)
		traceabilityClients = append(traceabilityClients, outputClient)
	}
	traceabilityGroup.Clients = clients
	return traceabilityGroup, nil
}

func makeLogstashClient(indexManager outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	libbeatCfg *common.Config,
	traceCfg *Config,
) (outputs.Group, error) {
	factory := outputs.FindFactory("logstash")
	if factory == nil {
		return outputs.Group{}, nil
	}

	// only run the health check if in online mode
	if !agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		err := registerHealthCheckers(traceCfg)
		if err != nil {
			return outputs.Group{}, err
		}
	}
	group, err := factory(indexManager, beat, observer, libbeatCfg)
	return group, err
}

func makeHTTPClient(beat beat.Info, observer outputs.Observer, traceCfg *Config, hosts []string) (outputs.Group, error) {
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
		})

		if err != nil {
			return outputs.Fail(err)
		}
		client = outputs.WithBackoff(client, traceCfg.Backoff.Init, traceCfg.Backoff.Max)
		clients[i] = client
	}

	registerHealthCheckers(traceCfg)
	return outputs.SuccessNet(traceCfg.LoadBalance, traceCfg.BulkMaxSize, traceCfg.MaxRetries, clients)
}

// SetTransportClient - set the transport client
func (client *Client) SetTransportClient(outputClient outputs.Client) {
	client.transportClient = outputClient
}

// Connect establishes a connection to the clients sink.
func (client *Client) Connect() error {
	// do not attempt to establish a connection in offline mode
	if agent.GetCentralConfig().GetUsageReportingConfig().IsOfflineMode() {
		return nil
	}

	networkClient := client.transportClient.(outputs.NetworkClient)
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

	err := client.transportClient.Close()
	if err != nil {
		return err
	}
	return nil
}

// Publish sends events to the clients sink.
func (client *Client) Publish(batch publisher.Batch) error {
	events := batch.Events()

	eventType := "metric"
	isMetric := false
	if len(events) > 0 {
		_, isMetric = events[0].Content.Meta["metric"]
	}

	if !isMetric {
		eventType = "transaction"
		if outputEventProcessor != nil {
			updatedEvents := outputEventProcessor.Process(events)
			if len(updatedEvents) > 0 {
				updateEvent(batch, updatedEvents)
				events = batch.Events() // update the events, for changes from outputEventProcessor
			} else {
				batch.ACK()
				return nil
			}
		}

		sampledEvents, err := sampling.FilterEvents(events)
		if err != nil {
			log.Error(err.Error())
		} else {
			updateEvent(batch, sampledEvents)
		}
	}

	publishCount := len(batch.Events())

	if publishCount > 0 {
		log.Infof("Creating %d %s events", publishCount, eventType)
	}

	err := client.transportClient.Publish(batch)
	if err != nil {
		log.Errorf("Failed to publish %s event : %s", eventType, err.Error())
		return err
	}

	if publishCount-len(batch.Events()) > 0 {
		log.Infof("%d %s events have been published", publishCount-len(batch.Events()), eventType)
	}

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
	// register a unique healthchecker for each potential host
	for i := range config.Hosts {
		ta := &traceabilityAgentHealthChecker{
			protocol: config.Protocol,
			host:     config.Hosts[i],
			proxyURL: config.Proxy.URL,
			timeout:  config.Timeout,
		}

		hcJob := &condorHealthCheckJob{
			agentHealthChecker: ta,
		}

		// TBD. Remove in future when Jobs interface is complete
		ta.hcJob = hcJob

		_, err := jobs.RegisterIntervalJobWithName(hcJob, config.Timeout, "Traceability Healthcheck")
		if err != nil {
			return err
		}

		// TBD. Remove in future when Jobs interface is complete
		err = registerHealthChecker(hcJob, ta.host)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO: From here down all temporary until Jobs interface finishes full implementation
func registerHealthChecker(hcJob *condorHealthCheckJob, host string) error {
	checkStatus := hcJob.agentHealthChecker.connectionHealthcheck

	_, err := hc.RegisterHealthcheck("Traceability Agent", host, checkStatus)
	if err != nil {
		return err
	}
	return nil
}

func (ta *traceabilityAgentHealthChecker) connectionHealthcheck(host string) *hc.Status {
	// Create the default status
	status := &hc.Status{
		Result: hc.OK,
	}

	err := ta.hcJob.checkConnections(healthcheckCondor)
	if err != nil {
		status = &hc.Status{
			Result:  hc.FAIL,
			Details: fmt.Sprintf("%s Failed. %s", host, err.Error()),
		}
	}
	return status
}
