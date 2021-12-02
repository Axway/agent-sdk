package traceability

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

var logstashClientCreateCalled = false

func init() {
	logstashFactory := func(
		indexManager outputs.IndexManager,
		beat beat.Info,
		observer outputs.Observer,
		cfg *common.Config,
	) (outputs.Group, error) {
		logstashClientCreateCalled = true
		return outputs.SuccessNet(false, 1, 1, nil)
	}
	outputs.RegisterType("logstash", logstashFactory)
}

func createCentralCfg(url, env string) *config.CentralConfiguration {
	cfg := config.NewCentralConfig(config.DiscoveryAgent).(*config.CentralConfiguration)
	cfg.URL = url
	cfg.TenantID = "123456"
	cfg.Environment = env
	authCfg := cfg.Auth.(*config.AuthConfiguration)
	authCfg.URL = url + "/auth"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "DOSA_1111"
	authCfg.PrivateKey = "../transaction/testdata/private_key.pem"
	authCfg.PublicKey = "../transaction/testdata/public_key"
	cfg.GetUsageReportingConfig().(*config.UsageReportingConfiguration).Interval = 30 * time.Second
	cfg.GetUsageReportingConfig().(*config.UsageReportingConfiguration).Offline = false
	return cfg
}

func createTransport(config *Config) (outputs.Group, error) {
	info := beat.Info{
		Beat:        "test-beat",
		IndexPrefix: "",
		Version:     "1.0",
	}
	// defcfg := DefaultConfig()
	commonCfg, _ := common.NewConfigFrom(config)
	return makeTraceabilityAgent(nil, info, nil, commonCfg)
}

func createBatch(msgValue string) *MockBatch {
	return &MockBatch{
		acked:      false,
		retryCount: 0,
		events:     createEvent(msgValue),
	}
}

func createEvent(msgValue string) []publisher.Event {
	fieldsData := common.MapStr{
		"message": msgValue,
	}
	return []publisher.Event{
		{
			Content: beat.Event{
				Timestamp: time.Now(),
				Meta:      common.MapStr{sampling.SampleKey: true},
				Private:   nil,
				Fields:    fieldsData,
			},
		},
	}
}

type mockHTTPServer struct {
	serverMessages []map[string]interface{}
	responseStatus int

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
				mockServer.ResetMessages()
				var body []byte
				contentEncoding := req.Header["Content-Encoding"]
				if contentEncoding != nil && contentEncoding[0] == "gzip" {
					body, _ = mockServer.decompressGzipContent(req.Body)
				} else {
					body, _ = ioutil.ReadAll(req.Body)
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

func (s *mockHTTPServer) Close() {
	s.server.Close()
}
func (s *mockHTTPServer) decompressGzipContent(gzipBufferReader io.Reader) ([]byte, error) {
	gzipReader, err := gzip.NewReader(gzipBufferReader)
	if err != nil {
		return nil, err
	}
	plainContent, err := ioutil.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}
	return plainContent, nil
}

type MockBatch struct {
	acked      bool
	retryCount int

	events []publisher.Event
}

func (b *MockBatch) Events() []publisher.Event                { return b.events }
func (b *MockBatch) ACK()                                     { b.acked = true }
func (b *MockBatch) Drop()                                    {}
func (b *MockBatch) Retry()                                   {}
func (b *MockBatch) Cancelled()                               {}
func (b *MockBatch) RetryEvents(events []publisher.Event)     { b.retryCount++ }
func (b *MockBatch) CancelledEvents(events []publisher.Event) {}

type testEventProcessor struct {
	msgValue string
}

func (t *testEventProcessor) Process(events []publisher.Event) []publisher.Event {
	return createEvent(t.msgValue)
}

func TestCreateLogstashClient(t *testing.T) {
	s := newMockHTTPServer()
	defer s.Close()

	cfg := createCentralCfg(s.server.URL, "v7")
	agent.Initialize(cfg)

	group, err := createTransport(nil)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "config is nil")
	assert.NotNil(t, group)
	assert.Nil(t, group.Clients)
	assert.False(t, logstashClientCreateCalled)
	testConfig := DefaultConfig()

	group, err = createTransport(testConfig)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "empty array accessing 'hosts'")
	assert.NotNil(t, group)
	assert.Nil(t, group.Clients)
	assert.False(t, logstashClientCreateCalled)

	testConfig.Hosts = []string{
		"somehost",
		"someotherhost",
	}
	group, err = createTransport(testConfig)
	assert.Nil(t, err)
	assert.NotNil(t, group)
	assert.NotNil(t, group.Clients)
	assert.True(t, logstashClientCreateCalled)

	testConfig.Pipelining = 5
	testConfig.Hosts = []string{
		"somehost2",
	}
	group, err = createTransport(testConfig)
	assert.Nil(t, err)
	assert.NotNil(t, group)
	assert.True(t, logstashClientCreateCalled)
	traceabilityClient := group.Clients[0].(*Client)
	assert.NotNil(t, traceabilityClient)
	assert.False(t, IsHTTPTransport())
	assert.Equal(t, 3, GetMaxRetries())
}

func TestCreateHTTPClientt(t *testing.T) {
	logstashClientCreateCalled = false

	testConfig := DefaultConfig()
	testConfig.Protocol = "http"

	testConfig.Hosts = []string{
		"somehost:invalidport",
	}
	group, err := createTransport(testConfig)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid port")
	assert.NotNil(t, group)
	assert.Nil(t, group.Clients)
	assert.False(t, logstashClientCreateCalled)

	testConfig.Hosts = []string{
		"somehost",
	}
	testConfig.Proxy = ProxyConfig{
		URL: "bogus\\:bogus",
	}

	group, err = createTransport(testConfig)
	assert.NotNil(t, err)
	assert.NotNil(t, group)
	assert.Nil(t, group.Clients)
	assert.False(t, logstashClientCreateCalled)

	testConfig.Proxy = ProxyConfig{}
	testConfig.CompressionLevel = 20
	group, err = createTransport(testConfig)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "requires value > 9 accessing 'compression_level'")
	assert.NotNil(t, group)
	assert.Nil(t, group.Clients)
	assert.False(t, logstashClientCreateCalled)

	testConfig.CompressionLevel = 0
	group, err = createTransport(testConfig)
	assert.Nil(t, err)
	assert.NotNil(t, group)
	assert.Equal(t, 1, len(group.Clients))
	traceabilityClient := group.Clients[0].(*Client)
	assert.NotNil(t, traceabilityClient)
	assert.False(t, logstashClientCreateCalled)
	assert.True(t, IsHTTPTransport())
	assert.Equal(t, 3, GetMaxRetries())
}

func TestHTTPTransportWithJSONEncoding(t *testing.T) {
	s := newMockHTTPServer()
	defer s.Close()

	cfg := createCentralCfg(s.server.URL, "v7")
	agent.Initialize(cfg)

	url, _ := url.Parse(s.server.URL)
	testConfig := DefaultConfig()
	testConfig.Protocol = "http"
	testConfig.CompressionLevel = 0
	testConfig.Hosts = []string{url.Hostname() + ":" + url.Port()}

	group, err := createTransport(testConfig)
	assert.Nil(t, err)
	assert.NotNil(t, group)
	traceabilityClient := group.Clients[0].(*Client)
	batch := createBatch("{\"f1\":\"test\"}")
	traceabilityClient.Connect()
	agent.StartAgentStatusUpdate()
	err = traceabilityClient.Publish(batch)
	traceabilityClient.Close()

	assert.Nil(t, err)
	publishedMessages := s.GetMessages()
	assert.NotNil(t, publishedMessages)
	assert.Equal(t, 1, len(publishedMessages))
	event := publishedMessages[0]
	assert.Nil(t, err)
	assert.Equal(t, "test", event["f1"])
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
	group, err := createTransport(testConfig)
	assert.Nil(t, err)
	traceabilityClient := group.Clients[0].(*Client)
	batch := createBatch("{\"f0\":\"dummy\"}")

	traceabilityClient.Connect()
	agent.StartAgentStatusUpdate()
	err = traceabilityClient.Publish(batch)
	traceabilityClient.Close()
	assert.Nil(t, err)

	publishedMessages := s.GetMessages()
	assert.NotNil(t, publishedMessages)
	assert.Equal(t, 1, len(publishedMessages))
	event := publishedMessages[0]
	assert.Equal(t, "test", event["f1"])
	assert.Nil(t, event["f0"])
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

	group, err := createTransport(testConfig)
	assert.Nil(t, err)
	assert.NotNil(t, group)
	traceabilityClient := group.Clients[0].(*Client)
	batch := createBatch("{\"f1\":\"test\"}")

	traceabilityClient.Connect()
	err = traceabilityClient.Publish(batch)
	assert.Nil(t, err)
	traceabilityClient.Close()

	publishedMessages := s.GetMessages()
	assert.NotNil(t, publishedMessages)
	assert.Equal(t, 1, len(publishedMessages))

	event := publishedMessages[0]

	assert.Nil(t, err)
	assert.Equal(t, "test", event["f1"])
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

	group, err := createTransport(testConfig)
	assert.Nil(t, err)
	traceabilityClient := group.Clients[0].(*Client)
	batch := createBatch("somemessage")

	s.responseStatus = 404
	traceabilityClient.Connect()
	err = traceabilityClient.Publish(batch)
	traceabilityClient.Close()
	assert.NotNil(t, err)
	assert.False(t, batch.acked)
	assert.Equal(t, 1, batch.retryCount)

	s.responseStatus = 500

	group, err = createTransport(testConfig)
	traceabilityClient = group.Clients[0].(*Client)
	traceabilityClient.Connect()
	err = traceabilityClient.Publish(batch)
	traceabilityClient.Close()
	assert.Nil(t, err)
	assert.True(t, batch.acked)
	assert.Equal(t, 1, batch.retryCount)
	publishedMessages := s.GetMessages()
	assert.Nil(t, publishedMessages)

	SetOutputEventProcessor(nil)
}
