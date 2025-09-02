package api

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mocHTTPServer struct {
	reqMethod    string
	reqBody      []byte
	reqUserAgent string

	expectedHeader   map[string]string
	processedHeaders bool

	expectedQueryParams  map[string]string
	processedQueryParams bool

	respBody []byte
	respCode int
	server   *httptest.Server
}

func (m *mocHTTPServer) reset() {
	m.reqMethod = ""
	m.reqBody = nil
	m.reqUserAgent = ""
	m.expectedHeader = nil
	m.processedHeaders = false
	m.expectedQueryParams = nil
	m.processedQueryParams = false
	m.respBody = nil
	m.respCode = 0
}

func (m *mocHTTPServer) handleReq(resp http.ResponseWriter, req *http.Request) {
	m.reqMethod = req.Method
	m.reqUserAgent = req.Header.Get("user-agent")
	m.processHeaders(req)
	m.processQueryParams(req)
	m.readReqBody(req)
	m.writeResponse(resp)
}

func (m *mocHTTPServer) processHeaders(req *http.Request) {
	m.processedHeaders = true
	for header, val := range m.expectedHeader {
		reqHdrVal := req.Header.Get(header)
		if val != reqHdrVal {
			m.processedHeaders = false
			return
		}
	}
}

func (m *mocHTTPServer) processQueryParams(req *http.Request) {
	m.processedQueryParams = true
	for param, val := range m.expectedQueryParams {
		reqParamVal := req.URL.Query().Get(param)
		if val != reqParamVal {
			m.processedQueryParams = false
			return
		}
	}
}

func (m *mocHTTPServer) readReqBody(req *http.Request) {
	var reqBuffer bytes.Buffer
	writer := bufio.NewWriter(&reqBuffer)
	_, _ = io.CopyN(writer, req.Body, 1024)
	m.reqBody = reqBuffer.Bytes()
}

func (m *mocHTTPServer) writeResponse(resp http.ResponseWriter) {
	if m.respBody != nil && len(m.respBody) > 0 {
		resp.Write(m.respBody)
	} else {
		resp.WriteHeader(m.respCode)
	}
}

func newMockHTTPServer() *mocHTTPServer {
	mockServer := &mocHTTPServer{}
	mockServer.server = httptest.NewServer(http.HandlerFunc(mockServer.handleReq))
	return mockServer
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name                string
		tls                 config.TLSConfig
		proxyURL            string
		timeout             string
		agentDialerExists   bool
		expectedTimeout     time.Duration
		expectedProxyScheme string
	}{
		{
			name:                "insecure-no-proxy-default-timeout",
			tls:                 nil,
			proxyURL:            "",
			timeout:             "",
			agentDialerExists:   false,
			expectedTimeout:     defaultTimeout,
			expectedProxyScheme: "",
		},
		{
			name:                "insecure-no-proxy-custom-timeout",
			tls:                 nil,
			proxyURL:            "",
			timeout:             "30s",
			agentDialerExists:   false,
			expectedTimeout:     30 * time.Second,
			expectedProxyScheme: "",
		},
		{
			name:                "insecure-http-proxy-custom-timeout",
			tls:                 nil,
			proxyURL:            "http://localhost:8080",
			timeout:             "30s",
			agentDialerExists:   true,
			expectedTimeout:     30 * time.Second,
			expectedProxyScheme: "http",
		},
		{
			name:                "secure-http-proxy-custom-timeout",
			tls:                 config.NewTLSConfig(),
			proxyURL:            "http://localhost:8080",
			timeout:             "30s",
			agentDialerExists:   true,
			expectedTimeout:     30 * time.Second,
			expectedProxyScheme: "http",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("HTTP_CLIENT_TIMEOUT", tc.timeout)
			c := NewClient(tc.tls, tc.proxyURL)
			hc, ok := c.(*httpClient)
			assert.True(t, ok)
			assert.Equal(t, hc.timeout, tc.expectedTimeout)
			httpTransport := hc.httpClient.Transport.(*http.Transport)
			assert.NotNil(t, httpTransport)
			if tc.tls != nil {
				assert.NotNil(t, httpTransport.TLSClientConfig)
			} else {
				assert.Nil(t, httpTransport.TLSClientConfig)
			}
			if tc.agentDialerExists {
				assert.NotNil(t, hc.dialer)
				assert.Equal(t, tc.expectedProxyScheme, hc.dialer.GetProxyScheme())
			} else {
				assert.Nil(t, hc.dialer)
			}
		})
	}
}

func TestNewSingleEntryClient(t *testing.T) {
	tests := []struct {
		name               string
		tls                config.TLSConfig
		proxyURL           string
		singleURL          string
		singleEntryFilter  []string
		expectedClientType string
	}{
		{
			name:     "no-single-entry",
			tls:      nil,
			proxyURL: "",
		},
		{
			name:              "insecure-no-proxy-default-timeout",
			tls:               nil,
			proxyURL:          "",
			singleURL:         "http://test",
			singleEntryFilter: []string{"http://abc"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			SetConfigAgent("", tc.singleURL, tc.singleEntryFilter)
			c := NewSingleEntryClient(tc.tls, tc.proxyURL, defaultTimeout)
			hc, ok := c.(*httpClient)
			assert.True(t, ok)
			httpTransport := hc.httpClient.Transport.(*http.Transport)
			assert.NotNil(t, httpTransport)
			if cfgAgent.singleURL != "" {
				assert.NotNil(t, hc.dialer)
				assert.Equal(t, len(hc.singleEntryHostMap), len(cfgAgent.singleEntryFilter))
				singleEntryURL, _ := url.Parse(cfgAgent.singleURL)
				singleEntryAddr := util.ParseAddr(singleEntryURL)
				for _, filterURL := range cfgAgent.singleEntryFilter {
					u, _ := url.Parse(filterURL)
					mappedAddr, ok := hc.singleEntryHostMap[util.ParseAddr(u)]
					assert.True(t, ok)
					assert.Equal(t, singleEntryAddr, mappedAddr)
				}
			} else {
				assert.Nil(t, hc.dialer)
			}
		})
	}
}

func TestSend(t *testing.T) {
	config.AgentTypeName = "Test"
	config.AgentVersion = "1.0"
	config.SDKVersion = "1.0"
	httpServer := newMockHTTPServer()
	hostname, _ := os.Hostname()
	runtimeID := uuid.New().String()

	tests := []struct {
		name              string
		method            string
		url               string
		queryParam        map[string]string
		header            map[string]string
		body              []byte
		respBody          []byte
		respCode          int
		expectedUserAgent string
		envName           string
		agentName         string
		isDocker          bool
		isGRPC            bool
		runtimeID         string
	}{
		{
			name:   "invalid-url",
			url:    "socks://invalid-url",
			method: GET,
		},
		{
			name:              "get-request-with-queryparam-header",
			url:               "http://test",
			method:            GET,
			queryParam:        map[string]string{"param1": "value1"},
			header:            map[string]string{"header1": "value1"},
			respCode:          200,
			respBody:          []byte{},
			envName:           "env",
			isDocker:          false,
			isGRPC:            true,
			agentName:         "agent",
			runtimeID:         runtimeID,
			expectedUserAgent: fmt.Sprintf("Test/1.0 (sdkVer:1.0; env:env; agent:agent; reactive:true; hostname:%s; runtimeID:%s)", hostname, runtimeID),
		},
		{
			name:              "post-request-with-response",
			url:               "http://test",
			method:            POST,
			body:              []byte("test-req"),
			respCode:          200,
			respBody:          []byte("test-resp"),
			envName:           "env",
			isDocker:          true,
			agentName:         "agent",
			isGRPC:            false,
			runtimeID:         runtimeID,
			expectedUserAgent: fmt.Sprintf("Test/1.0 (sdkVer:1.0; env:env; agent:agent; reactive:false; hostname:%s; runtimeID:%s)", hostname, runtimeID),
		},
		{
			name:              "override-user-agent",
			url:               "http://test",
			method:            GET,
			header:            map[string]string{"user-agent": "test"},
			respCode:          401,
			respBody:          []byte{},
			envName:           "env",
			isDocker:          true,
			agentName:         "agent",
			isGRPC:            false,
			runtimeID:         runtimeID,
			expectedUserAgent: "test",
		},
		{
			name:              "get-request-with-runtimeID",
			url:               "http://test",
			method:            GET,
			queryParam:        map[string]string{"param1": "value1"},
			header:            map[string]string{"header1": "value1"},
			respCode:          200,
			respBody:          []byte{},
			envName:           "env",
			isDocker:          false,
			isGRPC:            true,
			agentName:         "agent",
			runtimeID:         runtimeID,
			expectedUserAgent: fmt.Sprintf("Test/1.0 (sdkVer:1.0; env:env; agent:agent; reactive:true; hostname:%s; runtimeID:%s)", hostname, runtimeID),
		},
		{
			name:              "post-request-with-runtimeID",
			url:               "http://test",
			method:            POST,
			body:              []byte("test-req"),
			respCode:          200,
			respBody:          []byte("test-resp"),
			envName:           "env",
			isDocker:          false,
			isGRPC:            true,
			agentName:         "agent",
			runtimeID:         runtimeID,
			expectedUserAgent: fmt.Sprintf("Test/1.0 (sdkVer:1.0; env:env; agent:agent; reactive:true; hostname:%s; runtimeID:%s)", hostname, runtimeID),
		},
		{
			name:              "non-grpc-with-runtimeID",
			url:               "http://test",
			method:            GET,
			respCode:          200,
			respBody:          []byte{},
			envName:           "env",
			isDocker:          false,
			isGRPC:            false,
			agentName:         "agent",
			runtimeID:         runtimeID,
			expectedUserAgent: fmt.Sprintf("Test/1.0 (sdkVer:1.0; env:env; agent:agent; reactive:false; hostname:%s; runtimeID:%s)", hostname, runtimeID),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ua := util.NewUserAgent(
				config.AgentTypeName,
				config.AgentVersion,
				config.SDKVersion,
				tc.envName,
				tc.agentName,
				tc.isGRPC,
				tc.runtimeID,
			).FormatUserAgent()
			SetConfigAgent(ua, httpServer.server.URL, []string{"http://test"})
			httpServer.reset()
			httpServer.respBody = tc.respBody
			httpServer.respCode = tc.respCode
			httpServer.expectedHeader = tc.header
			httpServer.expectedQueryParams = tc.queryParam

			c := NewSingleEntryClient(nil, "", defaultTimeout)
			assert.NotNil(t, c)

			req := Request{
				Method:      tc.method,
				URL:         tc.url,
				QueryParams: tc.queryParam,
				Headers:     tc.header,
				Body:        tc.body,
			}

			res, err := c.Send(req)
			if tc.name == "invalid-url" {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, tc.respCode, res.Code)
				assert.Equal(t, tc.respBody, res.Body)
				assert.Equal(t, tc.method, httpServer.reqMethod)

				// Always check for the new user agent pattern with runtimeID
				if tc.expectedUserAgent == "test" {
					assert.Equal(t, tc.expectedUserAgent, httpServer.reqUserAgent)
				} else {
					assert.Equal(t, tc.expectedUserAgent, httpServer.reqUserAgent)
				}

				assert.True(t, httpServer.processedHeaders)
				assert.True(t, httpServer.processedQueryParams)
			}
		})
	}
}
