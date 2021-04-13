package traceability

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/appcelerator/chimera-client-go/chimera"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/outputs/transport"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/google/uuid"
)

// ChimeraClient struct
type ChimeraClient struct {
	chimeraClient    *chimera.Client
	compressionLevel int
	observer         outputs.Observer
	headers          map[string]string
	beatInfo         beat.Info
	host             string
	proxy            *url.URL
	tlsConfig        *transport.TLSConfig
	timeout          time.Duration
}

// ChimeraClientSettings struct
type ChimeraClientSettings struct {
	BeatInfo         beat.Info
	Host             string
	AuthToken        string
	Proxy            *url.URL
	TLS              *transport.TLSConfig
	Index            outil.Selector
	Pipeline         *outil.Selector
	Timeout          time.Duration
	CompressionLevel int
	Observer         outputs.Observer
	Headers          map[string]string
}

// Meta defines common event metadata to be stored in '@metadata'
type chimeraEventMetadata struct {
	Beat    string `json:"beat"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

// NewChimeraClient instantiate a client.
func NewChimeraClient(s ChimeraClientSettings) (*ChimeraClient, error) {
	ctx := context.Background()
	opts := chimera.ClientOptions{
		Protocol:      chimera.HTTPS,
		Host:          s.Host,
		AuthKey:       s.AuthToken,
		TLSConfig:     s.TLS.ToConfig(),
		ProxyResolver: util.GetProxyURL(s.Proxy),
		Timeout:       s.Timeout,
	}

	chimeraClient, err := chimera.NewClient(ctx, opts)
	if err != nil {
		return nil, err
	}
	client := &ChimeraClient{
		chimeraClient: chimeraClient,
		headers:       s.Headers,
		beatInfo:      s.BeatInfo,
		host:          s.Host,
		proxy:         s.Proxy,
		tlsConfig:     s.TLS,
		timeout:       s.Timeout,
	}

	return client, nil
}

// Connect establishes a connection to the clients sink.
func (client *ChimeraClient) Connect() error {
	return nil
}

// Close publish a single event to output.
func (client *ChimeraClient) Close() error {
	return nil
}

// Publish sends events to the clients sink.
func (client *ChimeraClient) Publish(batch publisher.Batch) error {
	events := batch.Events()
	rest, err := client.publishEvents(events)
	if len(rest) == 0 {
		batch.ACK()
	} else {
		batch.RetryEvents(rest)
	}
	return err
}

func (client *ChimeraClient) String() string {
	return client.host
}

// Clone clones a client.
func (client *ChimeraClient) Clone() *ChimeraClient {
	c, _ := NewChimeraClient(
		ChimeraClientSettings{
			BeatInfo:         client.beatInfo,
			Host:             client.host,
			Proxy:            client.proxy,
			TLS:              client.tlsConfig,
			Timeout:          client.timeout,
			CompressionLevel: client.compressionLevel,
			Headers:          client.headers,
		},
	)
	return c
}

// publishEvents - posts all events to the chimera endpoint.
func (client *ChimeraClient) publishEvents(data []publisher.Event) ([]publisher.Event, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var events = make([]chimera.Message, len(data))
	ctx := context.Background()
	for i, event := range data {
		events[i] = client.makeChimeraEvent(&event.Content)

	}
	_, err := client.chimeraClient.PublishMessages(ctx, events, chimera.PublishOptions{})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (client *ChimeraClient) makeChimeraEvent(v *beat.Event) chimera.Message {
	uuid, _ := uuid.NewUUID()
	var eventContent json.RawMessage
	msg := v.Fields["message"].(string)
	json.Unmarshal([]byte(msg), &eventContent)
	chimeraEvent := chimera.Message{
		ID:      uuid.String(),
		Content: []byte(eventContent),
	}

	return chimeraEvent
}
