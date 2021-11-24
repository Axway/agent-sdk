package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	log "github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/google/uuid"
)

// HTTP const definitions
const (
	GET    string = http.MethodGet
	POST   string = http.MethodPost
	PUT    string = http.MethodPut
	DELETE string = http.MethodDelete

	defaultTimeout     = time.Second * 60
	responseBufferSize = 2048
)

// Request - the request object used when communicating to an API
type Request struct {
	Method      string
	URL         string
	QueryParams map[string]string
	Headers     map[string]string
	Body        []byte
}

// Response - the response object given back when communicating to an API
type Response struct {
	Code    int
	Body    []byte
	Headers map[string][]string
}

// Client -
type Client interface {
	Send(request Request) (*Response, error)
}

type httpClient struct {
	Client
	httpClient *http.Client
	timeout    time.Duration
}

type configAgent struct {
	agentName       string
	environmentName string
	isDocker        bool
}

var cfgAgent *configAgent

func init() {
	cfgAgent = &configAgent{}
}

// SetConfigAgent -
func SetConfigAgent(env string, isDocker bool, agentName string) {
	cfgAgent.environmentName = env
	cfgAgent.isDocker = isDocker
	cfgAgent.agentName = agentName
}

// NewClient - creates a new HTTP client
func NewClient(cfg config.TLSConfig, proxyURL string) Client {
	timeout := getTimeoutFromEnvironment()
	return NewClientWithTimeout(cfg, proxyURL, timeout)
}

// NewClientWithTimeout - creates a new HTTP client, with a timeout
func NewClientWithTimeout(cfg config.TLSConfig, proxyURL string, timeout time.Duration) Client {
	httpCli := http.DefaultClient
	if cfg != nil {
		url, err := url.Parse(proxyURL)
		if err != nil {
			log.Errorf("Error parsing proxyURL from config; creating a non-proxy client: %s", err.Error())
		}
		httpCli = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: cfg.BuildTLSConfig(),
				Proxy:           util.GetProxyURL(url),
			},
		}
	}
	httpCli.Timeout = timeout
	return &httpClient{
		timeout:    timeout,
		httpClient: httpCli,
	}
}

func getTimeoutFromEnvironment() time.Duration {
	cfgHTTPClientTimeout := os.Getenv("HTTP_CLIENT_TIMEOUT")
	if cfgHTTPClientTimeout == "" {
		return defaultTimeout
	}
	timeout, err := time.ParseDuration(cfgHTTPClientTimeout)
	if err != nil {
		log.Tracef("Unable to parse the HTTP_CLIENT_TIMEOUT value, using the default http client timeout")
		return defaultTimeout
	}
	return timeout
}

func (c *httpClient) getURLEncodedQueryParams(queryParams map[string]string) string {
	params := url.Values{}
	for key, value := range queryParams {
		params.Add(key, value)
	}
	return params.Encode()
}

func (c *httpClient) prepareAPIRequest(ctx context.Context, request Request) (*http.Request, error) {
	requestURL := request.URL
	if len(request.QueryParams) != 0 {
		requestURL += "?" + c.getURLEncodedQueryParams(request.QueryParams)
	}
	req, err := http.NewRequestWithContext(ctx, request.Method, requestURL, bytes.NewBuffer(request.Body))
	if err != nil {
		return req, err
	}
	hasUserAgentHeader := false
	for key, value := range request.Headers {
		req.Header.Set(key, value)
		if strings.ToLower(key) == "user-agent" {
			hasUserAgentHeader = true
		}
	}
	if !hasUserAgentHeader {
		deploymentType := "binary"
		if cfgAgent.isDocker {
			deploymentType = "docker"
		}
		req.Header.Set("User-Agent", fmt.Sprintf("%s/%s SDK/%s %s %s %s", config.AgentTypeName, config.AgentVersion, config.SDKVersion, cfgAgent.environmentName, cfgAgent.agentName, deploymentType))
	}
	return req, err
}

func (c *httpClient) prepareAPIResponse(res *http.Response, timer *time.Timer) (*Response, error) {
	var err error
	var responeBuffer bytes.Buffer
	writer := bufio.NewWriter(&responeBuffer)
	for {
		// Reset the timeout timer for reading the response
		timer.Reset(c.timeout)
		_, err = io.CopyN(writer, res.Body, responseBufferSize)
		if err == io.EOF || err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
	}

	if err != nil {
		return nil, err
	}

	response := Response{
		Code:    res.StatusCode,
		Body:    responeBuffer.Bytes(),
		Headers: res.Header,
	}
	return &response, err
}

// Send - send the http request and returns the API Response
func (c *httpClient) Send(request Request) (*Response, error) {
	startTime := time.Now()
	ctx := context.Background()
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	req, err := c.prepareAPIRequest(cancelCtx, request)
	if err != nil {
		log.Errorf("Error preparing api request: %s", err.Error())
		return nil, err
	}
	reqID := uuid.New().String()
	// Logging for the HTTP request
	statusCode := 0
	defer func() {
		duration := time.Now().Sub(startTime)
		if err != nil {
			log.Tracef("[ID:%s] %s [%dms] - ERR - %s - %s", reqID, req.Method, duration.Milliseconds(), req.URL.String(), err.Error())
		} else {
			log.Tracef("[ID:%s] %s [%dms] - %d - %s", reqID, req.Method, duration.Milliseconds(), statusCode, req.URL.String())
		}
	}()

	// Start the timer to manage the timeout
	timer := time.AfterFunc(c.timeout, func() {
		cancel()
	})

	// Prevent reuse of the tcp connection to the same host
	req.Close = true

	if log.IsHTTPLogTraceEnabled() {
		req = log.NewRequestWithTraceContext(reqID, req)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	statusCode = res.StatusCode
	parseResponse, err := c.prepareAPIResponse(res, timer)

	return parseResponse, err
}
