package traceability

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/stretchr/testify/assert"
)

func createCentralCfg(url, env string) *config.CentralConfiguration {
	cfg := config.NewCentralConfig(config.DiscoveryAgent).(*config.CentralConfiguration)
	cfg.URL = url
	cfg.SingleURL = ""
	cfg.TenantID = "123456"
	cfg.Environment = env
	authCfg := cfg.Auth.(*config.AuthConfiguration)
	authCfg.URL = url + "/auth"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "serviceaccount_1234"
	authCfg.PrivateKey = "../transaction/testdata/private_key.pem"
	authCfg.PublicKey = "../transaction/testdata/public_key"
	cfg.GetMetricReportingConfig().(*config.MetricReportingConfiguration).Schedule = "* * * * *" // every minute
	cfg.GetUsageReportingConfig().(*config.UsageReportingConfiguration).Offline = false
	return cfg
}

func createTransport(cfg *Config) ([]*Client, error) {
	return NewClient(cfg)
}

func createBatch(msgValue string) *MockBatch {
	return &MockBatch{
		acked:      false,
		retryCount: 0,
		events:     createEvent(msgValue),
	}
}

func createEvent(msgValue string) []event.Event {
	fieldsData := event.MapStr{
		"message": msgValue,
	}
	return []event.Event{
		{
			Timestamp: time.Now(),
			Meta:      event.MapStr{sampling.SampleKey: true},
			Private:   nil,
			Fields:    fieldsData,
		},
	}
}

type mockHTTPServer struct {
	serverMessages   []map[string]interface{}
	responseStatus   int
	requestUserAgent string

	server *httptest.Server
}

func newMockHTTPServer() *mockHTTPServer {
	mockServer := &mockHTTPServer{}
	mockServer.server = httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		switch req.RequestURI {
		case "/auth/realms/Broker/protocol/openid-connect/token":
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		case "/":
			if req.Method == "POST" {
				if mockServer.responseStatus != 0 {
					resp.WriteHeader(mockServer.responseStatus)
					return
				}
				mockServer.requestUserAgent = req.Header.Get("User-Agent")
				mockServer.ResetMessages()
				var body []byte
				contentEncoding := req.Header["Content-Encoding"]
				if contentEncoding != nil && contentEncoding[0] == "gzip" {
					body, _ = mockServer.decompressGzipContent(req.Body)
				} else {
					body, _ = io.ReadAll(req.Body)
				}
				json.Unmarshal(body, &mockServer.serverMessages)
				resp.Write([]byte("ok"))
			}
			resp.Write([]byte("ok"))
		}
	}))
	return mockServer
}

func (s *mockHTTPServer) ResetStatus() {
	s.responseStatus = 0
}

func (s *mockHTTPServer) ResetMessages() {
	s.serverMessages = make([]map[string]interface{}, 0)
}

func (s *mockHTTPServer) GetMessages() []map[string]interface{} {
	return s.serverMessages
}

func (s *mockHTTPServer) GetUserAgent() string {
	return s.requestUserAgent
}

func (s *mockHTTPServer) Close() {
	s.server.Close()
}
func (s *mockHTTPServer) decompressGzipContent(gzipBufferReader io.Reader) ([]byte, error) {
	gzipReader, err := gzip.NewReader(gzipBufferReader)
	if err != nil {
		return nil, err
	}
	plainContent, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}
	return plainContent, nil
}

type MockBatch struct {
	acked      bool
	retryCount int

	events []event.Event
}

func (b *MockBatch) Events() []event.Event            { return b.events }
func (b *MockBatch) SetEvents(events []event.Event)   { b.events = events }
func (b *MockBatch) ACK()                             { b.acked = true }
func (b *MockBatch) Drop()                            {}
func (b *MockBatch) Retry()                           {}
func (b *MockBatch) Cancelled()                       {}
func (b *MockBatch) RetryEvents(events []event.Event) { b.retryCount++ }
func (b *MockBatch) CancelledEvents(events []event.Event) {}

type testEventProcessor struct {
	msgValue string
}

func (t *testEventProcessor) Process(events []event.Event) []event.Event {
	return createEvent(t.msgValue)
}

func TestParseConfig(t *testing.T) {
	agent.Initialize(createCentralCfg("http://localhost:8888", "v7"))

	tests := map[string]struct {
		raw     map[string]interface{}
		wantErr string
	}{
		"compression level out of bounds": {
			raw: map[string]interface{}{
				"compression_level": 20,
			},
			wantErr: "requires value <= 9 accessing 'compression_level'",
		},
		"valid full config round trip": {
			raw: map[string]interface{}{
				"hosts":             []string{"phoenix.datasearch.axway.com:443"},
				"protocol":          "https",
				"compression_level": 3,
				"bulk_max_size":     256,
				"max_retries":       5,
				"loadbalance":       true,
				"ssl": map[string]interface{}{
					"verification_mode": "full",
					"cipher_suites":     []string{"ECDHE-RSA-AES-128-GCM-SHA256"},
				},
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			cfg, err := ParseConfig(tc.raw)
			if tc.wantErr != "" {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, cfg)
		})
	}
}

func TestCreateHTTPClient(t *testing.T) {
	cfg := createCentralCfg("http://localhost:8888", "v7")
	agent.Initialize(cfg)

	tests := map[string]struct {
		hosts       []string
		proxy       ProxyConfig
		wantErr     bool
		wantErrMsg  string
		wantClients int
	}{
		"invalid port": {
			hosts:      []string{"somehost:invalidport"},
			wantErr:    true,
			wantErrMsg: "invalid port",
		},
		"bad proxy URL": {
			hosts:   []string{"somehost"},
			proxy:   ProxyConfig{URL: "bogus\\:bogus"},
			wantErr: true,
		},
		"valid host and no proxy": {
			hosts:       []string{"somehost"},
			wantClients: 1,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			testConfig := DefaultConfig()
			testConfig.Hosts = tc.hosts
			testConfig.Proxy = tc.proxy

			clients, err := createTransport(testConfig)
			if tc.wantErr {
				assert.NotNil(t, err)
				if tc.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tc.wantErrMsg)
				}
				assert.Nil(t, clients)
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, tc.wantClients, len(clients))
			assert.NotNil(t, clients[0])
			assert.True(t, IsHTTPTransport())
			assert.Equal(t, 3, GetMaxRetries())
		})
	}
}

func TestValidateCfgRemovedProtocolPortHost(t *testing.T) {
	tests := map[string]struct {
		cfg     *Config
		wantErr error
	}{
		"tcp protocol removed": {
			cfg:     &Config{Protocol: "tcp"},
			wantErr: ErrTCPProtocolRemoved,
		},
		"lumberjack port 5044 removed": {
			cfg:     &Config{Protocol: "https", Hosts: []string{"phoenix.datasearch.axway.com:5044"}},
			wantErr: ErrPort5044Removed.FormatError("phoenix.datasearch.axway.com:5044"),
		},
		"ingestion host removed": {
			cfg:     &Config{Protocol: "https", Hosts: []string{"ingestion.datasearch.axway.com:443"}},
			wantErr: ErrIngestionHostRemoved.FormatError("ingestion.datasearch.axway.com:443"),
		},
		"ingestion-http host removed": {
			cfg:     &Config{Protocol: "https", Hosts: []string{"ingestion-http.datasearch.axway.com:443"}},
			wantErr: ErrIngestionHostRemoved.FormatError("ingestion-http.datasearch.axway.com:443"),
		},
		"ingestion-lumberjack host removed": {
			cfg:     &Config{Protocol: "https", Hosts: []string{"ingestion-lumberjack.datasearch.axway.com:443"}},
			wantErr: ErrIngestionHostRemoved.FormatError("ingestion-lumberjack.datasearch.axway.com:443"),
		},
		"valid phoenix https host passes": {
			cfg: &Config{Protocol: "https", Hosts: []string{"phoenix.datasearch.axway.com:443"}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.cfg.ValidateCfg()
			if tc.wantErr == nil {
				assert.Nil(t, err)
				return
			}
			assert.NotNil(t, err)
			assert.Equal(t, tc.wantErr.Error(), err.Error())
		})
	}
}

func TestHTTPTransportWithJSONEncoding(t *testing.T) {
	s := newMockHTTPServer()
	defer s.Close()
	config.AgentTypeName = "TraceabilityAgent"
	config.AgentVersion = "0.0.1-abc"
	config.SDKVersion = "0.0.1"

	cfg := createCentralCfg(s.server.URL, "v7")
	agent.Initialize(cfg)

	url, _ := url.Parse(s.server.URL)
	testConfig := DefaultConfig()
	testConfig.Protocol = "http"
	testConfig.CompressionLevel = 0
	testConfig.Hosts = []string{url.Hostname() + ":" + url.Port()}

	clients, err := createTransport(testConfig)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(clients))
	traceabilityClient := clients[0]
	batch := createBatch("{\"f1\":\"test\"}")
	traceabilityClient.Connect()
	agent.StartAgentStatusUpdate()
	err = traceabilityClient.Publish(context.Background(), batch)
	traceabilityClient.Close()

	assert.Nil(t, err)
	publishedMessages := s.GetMessages()
	reqUA := s.GetUserAgent()
	assert.NotEmpty(t, reqUA)
	assert.NotNil(t, publishedMessages)
	assert.Equal(t, 1, len(publishedMessages))
	msg := publishedMessages[0]
	assert.Nil(t, err)
	assert.Equal(t, "test", msg["f1"])
	assert.True(t, batch.acked)
}

func TestHTTPTransportWithOutputProcessor(t *testing.T) {
	s := newMockHTTPServer()
	defer s.Close()

	cfg := createCentralCfg(s.server.URL, "v7")
	agent.Initialize(cfg)

	url, _ := url.Parse(s.server.URL)
	testConfig := DefaultConfig()
	testConfig.Protocol = "http"
	testConfig.CompressionLevel = 0
	testConfig.Hosts = []string{
		url.Hostname() + ":" + url.Port(),
	}

	eventProcessor := &testEventProcessor{msgValue: "{\"f1\":\"test\"}"}
	SetOutputEventProcessor(eventProcessor)
	clients, err := createTransport(testConfig)
	assert.Nil(t, err)
	traceabilityClient := clients[0]
	batch := createBatch("{\"f0\":\"dummy\"}")

	traceabilityClient.Connect()
	agent.StartAgentStatusUpdate()
	err = traceabilityClient.Publish(context.Background(), batch)
	traceabilityClient.Close()
	assert.Nil(t, err)

	publishedMessages := s.GetMessages()
	assert.NotNil(t, publishedMessages)
	assert.Equal(t, 1, len(publishedMessages))
	msg := publishedMessages[0]
	assert.Equal(t, "test", msg["f1"])
	assert.Nil(t, msg["f0"])
	assert.True(t, batch.acked)

	SetOutputEventProcessor(nil)
}

func TestHTTPTransportWithGzipEncoding(t *testing.T) {
	s := newMockHTTPServer()
	defer s.Close()

	cfg := createCentralCfg(s.server.URL, "v7")
	agent.Initialize(cfg)

	url, _ := url.Parse(s.server.URL)
	testConfig := DefaultConfig()
	testConfig.Protocol = "http"
	testConfig.CompressionLevel = 3
	testConfig.Hosts = []string{
		url.Hostname() + ":" + url.Port(),
	}

	clients, err := createTransport(testConfig)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(clients))
	traceabilityClient := clients[0]
	batch := createBatch("{\"f1\":\"test\"}")

	traceabilityClient.Connect()
	err = traceabilityClient.Publish(context.Background(), batch)
	assert.Nil(t, err)
	traceabilityClient.Close()

	publishedMessages := s.GetMessages()
	assert.NotNil(t, publishedMessages)
	assert.Equal(t, 1, len(publishedMessages))

	msg := publishedMessages[0]

	assert.Nil(t, err)
	assert.Equal(t, "test", msg["f1"])
	assert.True(t, batch.acked)
}

func TestHTTPTransportRetries(t *testing.T) {
	s := newMockHTTPServer()
	defer s.Close()

	cfg := createCentralCfg(s.server.URL, "v7")
	agent.Initialize(cfg)

	url, _ := url.Parse(s.server.URL)
	testConfig := DefaultConfig()
	testConfig.Protocol = "http"
	testConfig.CompressionLevel = 0
	testConfig.Hosts = []string{
		url.Hostname() + ":" + url.Port(),
	}

	clients, err := createTransport(testConfig)
	assert.Nil(t, err)
	traceabilityClient := clients[0]
	batch := createBatch("somemessage")

	s.responseStatus = 404
	traceabilityClient.Connect()
	err = traceabilityClient.Publish(context.Background(), batch)
	traceabilityClient.Close()
	assert.NotNil(t, err)
	assert.False(t, batch.acked)
	assert.Equal(t, 1, batch.retryCount)

	s.responseStatus = 500
	batch = createBatch("somemessage")
	clients, err = createTransport(testConfig)
	assert.Nil(t, err)

	traceabilityClient = clients[0]
	traceabilityClient.Connect()
	err = traceabilityClient.Publish(context.Background(), batch)
	traceabilityClient.Close()
	assert.NotNil(t, err)
	assert.False(t, batch.acked)
	assert.Equal(t, 1, batch.retryCount)
	publishedMessages := s.GetMessages()
	assert.Nil(t, publishedMessages)

	SetOutputEventProcessor(nil)
}
