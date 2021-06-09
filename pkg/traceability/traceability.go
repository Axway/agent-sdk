package traceability

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"time"
	"unsafe"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/publisher"
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
	hcTypeTCP                     = "tcp"
	hcTypeHTTP                    = "http"
)

// Client - struct
type Client struct {
	transportClient outputs.Client
}

type traceabilityAgentHealthChecker struct {
	protocol string
	host     string
	proxyURL string
	timeout  time.Duration
	// checkStatus hc.CheckStatus
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

func makeTraceabilityAgent(
	indexManager outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	libbeatCfg *common.Config,
) (outputs.Group, error) {
	traceCfg, err := readConfig(libbeatCfg, beat)
	if err != nil {
		agent.UpdateStatus(agent.AgentFailed, err.Error())
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(libbeatCfg)
	if err != nil {
		agent.UpdateStatus(agent.AgentFailed, err.Error())
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

	registerHealthCheckers(hcTypeTCP, traceCfg)
	group, err := factory(indexManager, beat, observer, libbeatCfg)
	return group, err
}

func makeHTTPClient(beat beat.Info, observer outputs.Observer, traceCfg *Config, hosts []string) (outputs.Group, error) {
	tls, err := tlscommon.LoadTLSConfig(traceCfg.TLS)
	if err != nil {
		agent.UpdateStatus(agent.AgentFailed, err.Error())
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

	registerHealthCheckers(hcTypeHTTP, traceCfg)
	return outputs.SuccessNet(traceCfg.LoadBalance, traceCfg.BulkMaxSize, traceCfg.MaxRetries, clients)
}

// Connect establishes a connection to the clients sink.
func (client *Client) Connect() error {
	networkClient := client.transportClient.(outputs.NetworkClient)
	err := networkClient.Connect()
	if err != nil {
		return err
	}
	return nil
}

// Close publish a single event to output.
func (client *Client) Close() error {
	err := client.transportClient.Close()
	if err != nil {
		return err
	}
	return nil
}

// Publish sends events to the clients sink.
func (client *Client) Publish(batch publisher.Batch) error {
	events := batch.Events()
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

	publishCount := len(batch.Events())
	log.Infof("Publishing %d events", publishCount)
	//update the local activity timestamp for the event to compare against
	agent.UpdateLocalActivityTime()
	err = client.transportClient.Publish(batch)
	if err != nil {
		return err
	}
	log.Infof("Published %d events", publishCount-len(batch.Events()))
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

func registerHealthCheckers(hcType string, config *Config) {
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
		jobs.RegisterIntervalJob(hcJob, config.Timeout)
	}
}

func (ta *traceabilityAgentHealthChecker) healthcheck(name string) *hc.Status {
	if ta.protocol == "tcp" {
		return ta.tcpHealthcheck(name)
	}
	return ta.httpHealthcheck(name)
}

func (ta *traceabilityAgentHealthChecker) tcpHealthcheck(host string) *hc.Status {
	// Create the default status
	status := &hc.Status{
		Result: hc.OK,
	}

	hostURL := ta.host
	if ta.proxyURL != "" {
		hostURL = ta.proxyURL
	}
	_, err := net.DialTimeout(ta.protocol, hostURL, ta.timeout)
	if err != nil {
		status = &hc.Status{
			Result:  hc.FAIL,
			Details: fmt.Sprintf("%s Failed. %s", host, err.Error()),
		}
	}

	return status
}

func (ta *traceabilityAgentHealthChecker) httpHealthcheck(host string) *hc.Status {
	// Create the default status
	status := &hc.Status{
		Result: hc.OK,
	}

	request := api.Request{
		Method: http.MethodConnect,
		URL:    ta.protocol + "://" + ta.host,
	}

	client := api.NewClient(nil, ta.proxyURL)
	response, err := client.Send(request)
	if err != nil {
		status = &hc.Status{
			Result:  hc.FAIL,
			Details: fmt.Sprintf("%s Failed. %s", host, err.Error()),
		}
		return status
	}
	if response.Code == http.StatusRequestTimeout {
		status = &hc.Status{
			Result:  hc.FAIL,
			Details: fmt.Sprintf("%s Failed. HTTP response: %v", host, response.Code),
		}
	}

	return status
}
