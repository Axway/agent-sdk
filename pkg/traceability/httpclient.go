package traceability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/outputs/transport"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

const (
	// TransactionFlow - the transaction flow used for events
	TransactionFlow = "api-central-v8"
	// FlowHeader - the header key for the flow value
	FlowHeader = "axway-target-flow"
)

// HTTPClient struct
type HTTPClient struct {
	Connection
	tlsConfig        *transport.TLSConfig
	compressionLevel int
	proxyURL         *url.URL
	headers          map[string]string
	beatInfo         beat.Info
	logger           log.FieldLogger
	timeout          time.Duration
}

// HTTPClientSettings struct
type HTTPClientSettings struct {
	BeatInfo         beat.Info
	URL              string
	Proxy            *url.URL
	TLS              *transport.TLSConfig
	Index            outil.Selector
	Pipeline         *outil.Selector
	Timeout          time.Duration
	CompressionLevel int
	Observer         outputs.Observer
	Headers          map[string]string
	UserAgent        string
	IsSingleEntry    bool
}

// Connection struct
type Connection struct {
	sync.Mutex
	URL       string
	api       api.Client
	connected bool
	encoder   bodyEncoder
	userAgent string
}

// NewHTTPClient instantiate a client.
func NewHTTPClient(s HTTPClientSettings) (*HTTPClient, error) {
	var encoder bodyEncoder
	var err error
	compression := s.CompressionLevel
	if compression == 0 {
		encoder = newJSONEncoder(nil)
	} else {
		encoder, err = newGzipEncoder(compression, nil)
		if err != nil {
			return nil, err
		}
	}

	logger := log.NewFieldLogger().
		WithPackage("sdk.traceability").
		WithComponent("HTTPClient")

	opts := []api.ClientOpt{api.WithTimeout(s.Timeout)}
	if s.IsSingleEntry {
		opts = append(opts, api.WithSingleURLFor(s.URL))
	}

	tlsCfg := config.NewTLSConfig().(*config.TLSConfiguration)
	tlsCfg.LoadFrom(s.TLS.ToConfig())

	client := &HTTPClient{
		Connection: Connection{
			URL:       s.URL,
			api:       api.NewClient(tlsCfg, s.Proxy.String(), opts...),
			encoder:   encoder,
			userAgent: s.UserAgent,
		},
		compressionLevel: compression,
		proxyURL:         s.Proxy,
		headers:          s.Headers,
		beatInfo:         s.BeatInfo,
		logger:           logger,
		timeout:          s.Timeout,
	}

	return client, nil
}

// Connect establishes a connection to the clients sink.
func (client *HTTPClient) Connect() error {
	client.Connection.updateConnected(true)
	return nil
}

// Close publish a single event to output.
func (client *HTTPClient) Close() error {
	client.Connection.updateConnected(false)
	return nil
}

// Publish sends events to the clients sink.
func (client *HTTPClient) Publish(_ context.Context, batch publisher.Batch) error {
	events := batch.Events()
	err := client.publishEvents(events)
	if err == nil {
		batch.ACK()
	} else {
		batch.RetryEvents(events)
	}
	return err
}

func (client *HTTPClient) String() string {
	return client.URL
}

// Clone clones a client.
func (client *HTTPClient) Clone() *HTTPClient {
	c, _ := NewHTTPClient(
		HTTPClientSettings{
			BeatInfo:         client.beatInfo,
			URL:              client.URL,
			Proxy:            client.proxyURL,
			TLS:              client.tlsConfig,
			Timeout:          client.timeout,
			CompressionLevel: client.compressionLevel,
			Headers:          client.headers,
		},
	)
	return c
}

// publishEvents - posts all events to the http endpoint.
func (client *HTTPClient) publishEvents(data []publisher.Event) error {
	if len(data) == 0 {
		return nil
	}

	if !client.isConnected() {
		return ErrHTTPNotConnected
	}

	if client.headers == nil {
		client.headers = make(map[string]string)
	}

	var events = make([]json.RawMessage, len(data))
	timeStamp := time.Now()
	for i, event := range data {
		events[i] = client.makeHTTPEvent(&event.Content)
		if i == 0 {
			timeStamp = event.Content.Timestamp
			allFields, err := event.Content.Fields.GetValue("fields")
			if err != nil {
				client.headers[FlowHeader] = TransactionFlow
				continue
			}
			if flow, ok := allFields.(map[string]interface{})[FlowHeader]; !ok {
				client.headers[FlowHeader] = TransactionFlow
			} else {
				client.headers[FlowHeader] = flow.(string)
			}
		}
	}
	status, _, err := client.request(events, client.headers, timeStamp)
	if err != nil {
		client.logger.WithError(err).Error("transport error")
		return err
	}

	if status != http.StatusOK && status != http.StatusCreated { // server error or bad input
		client.logger.WithField("status", status).Error("failed to publish event")
		return fmt.Errorf("failed to publish event, status: %d", status)
	}

	return nil
}

func (conn *Connection) isConnected() bool {
	conn.Lock()
	defer conn.Unlock()
	return conn.connected
}

func (conn *Connection) updateConnected(update bool) {
	conn.Lock()
	defer conn.Unlock()
	conn.connected = update
}

func (conn *Connection) request(body interface{}, headers map[string]string, eventTime time.Time) (int, []byte, error) {
	urlStr := strings.TrimSuffix(conn.URL, "/")

	if err := conn.encoder.Marshal(body); err != nil {
		return 0, nil, ErrJSONEncodeFailed
	}
	return conn.execRequest(urlStr, conn.encoder.Reader(), headers, eventTime)
}

func (conn *Connection) execRequest(url string, body io.Reader, headers map[string]string, eventTime time.Time) (int, []byte, error) {
	data := make([]byte, 0)
	if body != nil {
		var err error
		data, err = io.ReadAll(body)
		if err != nil {
			return 0, nil, err
		}
	}

	err := conn.addHeaders(headers, body != nil, eventTime)
	if err != nil {
		return 0, nil, err
	}

	req := api.Request{
		Method:  http.MethodPost,
		URL:     url,
		Body:    data,
		Headers: headers,
	}

	return conn.execHTTPRequest(req)
}

func (conn *Connection) addHeaders(headers map[string]string, hasBody bool, eventTime time.Time) error {
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		return err
	}

	headers["Authorization"] = "Bearer " + token
	headers["Capture-Org-ID"] = agent.GetCentralConfig().GetTenantID()
	headers["User-Agent"] = conn.userAgent
	headers["Timestamp"] = strconv.FormatInt(eventTime.UTC().Unix(), 10)

	if hasBody {
		conn.encoder.AddHeader(headers)
	}

	return nil
}

func (conn *Connection) execHTTPRequest(req api.Request) (int, []byte, error) {
	resp, err := conn.api.Send(req)
	if err != nil {
		return 0, nil, err
	}
	return resp.Code, resp.Body, nil
}

func (client *HTTPClient) makeHTTPEvent(v *beat.Event) json.RawMessage {
	var eventData json.RawMessage
	msg := v.Fields["message"].(string)
	json.Unmarshal([]byte(msg), &eventData)

	return eventData
}
