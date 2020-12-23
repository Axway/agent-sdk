package traceability

import (
	"net/url"
	"reflect"
	"unsafe"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/agent"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
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
)

var debugf = logp.MakeDebug("traceability")

// Client - struct
type Client struct {
	transportClient outputs.Client
}

func init() {
	outputs.RegisterType("traceability", makeTraceabilityAgent)
}

// SetOutputEventProcessor -
func SetOutputEventProcessor(eventProcessor OutputEventProcessor) {
	outputEventProcessor = eventProcessor
}

func makeTraceabilityAgent(
	indexManager outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	config, err := readConfig(cfg, beat)
	if err != nil {
		agent.UpdateStatus(agent.AgentFailed, err.Error())
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		agent.UpdateStatus(agent.AgentFailed, err.Error())
		return outputs.Fail(err)
	}
	if len(hosts) > 1 {
		return outputs.Fail(ErrInvalidConfig.FormatError("multiple host for traceability output not supported"))
	}

	var transportGroup outputs.Group
	if config.Protocol == "https" || config.Protocol == "http" {
		transportGroup, err = makeHTTPClient(beat, observer, config, hosts)
	} else {
		transportGroup, err = makeLogstashClient(indexManager, beat, observer, cfg)
	}

	if err != nil {
		return outputs.Fail(err)
	}

	traceabilityGroup := outputs.Group{
		Clients: []outputs.Client{
			&Client{
				transportClient: transportGroup.Clients[0], // Only one client allowed for host
			},
		},
		BatchSize: transportGroup.BatchSize,
		Retry:     transportGroup.Retry,
	}
	return traceabilityGroup, nil
}

func makeLogstashClient(indexManager outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	factory := outputs.FindFactory("logstash")
	if factory == nil {
		return outputs.Group{}, nil
	}
	group, err := factory(indexManager, beat, observer, cfg)
	return group, err
}

func makeHTTPClient(beat beat.Info, observer outputs.Observer, config *Config, hosts []string) (outputs.Group, error) {

	tls, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		agent.UpdateStatus(agent.AgentFailed, err.Error())
		return outputs.Fail(err)
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		hostURL, err := common.MakeURL(config.Protocol, "/", host, 443)
		if err != nil {
			return outputs.Fail(err)
		}
		proxyURL, err := url.Parse(config.Proxy.URL)
		if err != nil {
			return outputs.Fail(err)
		}
		var client outputs.NetworkClient
		client, err = NewHTTPClient(HTTPClientSettings{
			BeatInfo:         beat,
			URL:              hostURL,
			Proxy:            proxyURL,
			TLS:              tls,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
			Observer:         observer,
		})

		if err != nil {
			return outputs.Fail(err)
		}
		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
		clients[i] = client
	}
	return outputs.SuccessNet(config.LoadBalance, config.BulkMaxSize, config.MaxRetries, clients)
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
		} else {
			batch.ACK()
			return nil
		}
	}

	publishCount := len(batch.Events())
	logp.Info("Publishing %d events", publishCount)
	err := client.transportClient.Publish(batch)
	if err != nil {
		return err
	}
	logp.Info("Published %d events", publishCount-len(batch.Events()))
	return nil
}

func (client *Client) String() string {
	return "traceability"
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
