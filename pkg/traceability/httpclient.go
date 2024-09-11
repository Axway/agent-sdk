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
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/outputs/transport"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/google/uuid"
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
}

// Connection struct
type Connection struct {
	sync.Mutex
	URL       string
	http      *http.Client
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

	client := &HTTPClient{
		Connection: Connection{
			URL: s.URL,
			http: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: s.TLS.ToConfig(),
					Proxy:           util.GetProxyURL(s.Proxy),
				},
				Timeout: s.Timeout,
			},
			encoder:   encoder,
			userAgent: s.UserAgent,
		},
		compressionLevel: compression,
		proxyURL:         s.Proxy,
		headers:          s.Headers,
		beatInfo:         s.BeatInfo,
		logger:           logger,
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
			Timeout:          client.http.Timeout,
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
	req, err := http.NewRequest("POST", url, body)
	if log.IsHTTPLogTraceEnabled() {
		req = log.NewRequestWithTraceContext(uuid.New().String(), req)
	}

	if err != nil {
		return 0, nil, err
	}

	err = conn.addHeaders(&req.Header, body, eventTime)
	if err != nil {
		return 0, nil, err
	}

	return conn.execHTTPRequest(req, headers)
}

func (conn *Connection) addHeaders(header *http.Header, body io.Reader, eventTime time.Time) error {
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		return err
	}

	header.Add("Authorization", "Bearer "+token)
	header.Add("Capture-Org-ID", agent.GetCentralConfig().GetTenantID())
	header.Add("User-Agent", conn.userAgent)
	header.Add("Timestamp", strconv.FormatInt(eventTime.UTC().Unix(), 10))

	if body != nil {
		conn.encoder.AddHeader(header)
	}
	return nil
}

func (conn *Connection) execHTTPRequest(req *http.Request, headers map[string]string) (int, []byte, error) {
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	resp, err := conn.http.Do(req)
	if err != nil {
		conn.updateConnected(false)
		return 0, nil, err
	}
	defer closing(resp.Body)

	status := resp.StatusCode
	if status >= 300 {
		conn.updateConnected(false)
		return status, nil, fmt.Errorf("%v", resp.Status)
	}
	obj, err := io.ReadAll(resp.Body)
	if err != nil {
		conn.updateConnected(false)
		return status, nil, err
	}
	return status, obj, nil
}

func closing(c io.Closer) {
	c.Close()
}

func (client *HTTPClient) makeHTTPEvent(v *beat.Event) json.RawMessage {
	var eventData json.RawMessage
	msg := v.Fields["message"].(string)
	json.Unmarshal([]byte(msg), &eventData)

	return eventData
}
