package traceability

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
)

const testCentralURL = "http://localhost:8888"

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

func (b *MockBatch) Events() []event.Event                { return b.events }
func (b *MockBatch) SetEvents(events []event.Event)       { b.events = events }
func (b *MockBatch) ACK()                                 { b.acked = true }
func (b *MockBatch) Drop()                                {}
func (b *MockBatch) Retry()                               {}
func (b *MockBatch) Cancelled()                           {}
func (b *MockBatch) RetryEvents(events []event.Event)     { b.retryCount++ }
func (b *MockBatch) CancelledEvents(events []event.Event) {}

type testEventProcessor struct {
	msgValue string
}

func (t *testEventProcessor) Process(events []event.Event) []event.Event {
	return createEvent(t.msgValue)
}

func TestParseConfig(t *testing.T) {
	const testHost = "phoenix.datasearch.axway.com:443"

	// properties.Properties reads env var overrides through viper's AutomaticEnv, which pkg/cmd/root.go
	// normally connects once at agent startup. Replicate that here since this test bypasses root.go.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	agent.Initialize(createCentralCfg(testCentralURL, "v7"))

	tests := map[string]struct {
		envVars map[string]string
		assert  func(t *testing.T, cfg *Config)
	}{
		"defaults with no env vars set": {
			envVars: map[string]string{},
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 3, cfg.CompressionLevel)
				assert.Equal(t, 512, cfg.BulkMaxSize)
				assert.Equal(t, "https", cfg.Protocol)
				assert.Empty(t, cfg.Hosts)
			},
		},
		"compression level out of bounds falls back to the default": {
			envVars: map[string]string{
				"TRACEABILITY_COMPRESSIONLEVEL": "20",
			},
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 3, cfg.CompressionLevel)
			},
		},
		"valid full config round trip": {
			envVars: map[string]string{
				"TRACEABILITY_HOST":                 testHost,
				"TRACEABILITY_PROTOCOL":             "https",
				"TRACEABILITY_COMPRESSIONLEVEL":     "5",
				"TRACEABILITY_BULKMAXSIZE":          "256",
				"TRACEABILITY_MAXRETRIES":           "5",
				"TRACEABILITY_LOADBALANCE":          "true",
				"TRACEABILITY_SSL_VERIFICATIONMODE": "full",
				"TRACEABILITY_SSL_CIPHERSUITES":     "ECDHE-RSA-AES-128-GCM-SHA256",
			},
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, []string{testHost}, cfg.Hosts)
				assert.Equal(t, "https", cfg.Protocol)
				assert.Equal(t, 5, cfg.CompressionLevel)
				assert.Equal(t, 256, cfg.BulkMaxSize)
				assert.Equal(t, 5, cfg.MaxRetries)
				assert.True(t, cfg.LoadBalance)
				assert.Equal(t, "full", cfg.TLS.VerificationMode)
				assert.Equal(t, []string{"ECDHE-RSA-AES-128-GCM-SHA256"}, cfg.TLS.CipherSuites)
			},
		},
		"redaction show list matches mulesoft-agents' no-space-after-colon format": {
			envVars: map[string]string{
				"TRACEABILITY_REDACTION_PATH_SHOW": `[{keyMatch:".*"}]`,
			},
			assert: func(t *testing.T, cfg *Config) {
				assert.Equal(t, []redaction.Show{{KeyMatch: ".*"}}, cfg.Redaction.Path.Allowed)
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			props := properties.NewProperties(&cobra.Command{})
			AddConfigProperties(props)

			cfg, err := ParseConfig(props)
			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			tc.assert(t, cfg)
		})
	}
}

func TestCreateHTTPClient(t *testing.T) {
	cfg := createCentralCfg(testCentralURL, "v7")
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

// TestValidateProtocolPort verifies traceability ingestion protocol is forced to https whenever a
// single entry URL is configured (recognized or not), and left alone otherwise.
func TestValidateProtocolPort(t *testing.T) {
	tests := map[string]struct {
		singleURL        string
		configuredProto  string
		expectedProtocol string
	}{
		"unrecognized single entry URL overrides the configured protocol": {
			singleURL:        "https://sl1rd15app0514.pcloud.axway.int:28080",
			configuredProto:  "tcp",
			expectedProtocol: "https",
		},
		"recognized single entry URL overrides the configured protocol": {
			singleURL:        "https://ingestion.platform.axway.com",
			configuredProto:  "tcp",
			expectedProtocol: "https",
		},
		"no single entry URL leaves the configured protocol alone": {
			singleURL:        "",
			configuredProto:  "tcp",
			expectedProtocol: "tcp",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			cfg := createCentralCfg(testCentralURL, "v7")
			cfg.SingleURL = tc.singleURL
			agent.Initialize(cfg)

			traceCfg = &Config{Protocol: tc.configuredProto}
			validateProtocolPort()

			assert.Equal(t, tc.expectedProtocol, traceCfg.Protocol)
		})
	}
}
