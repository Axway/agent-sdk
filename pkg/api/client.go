package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/google/uuid"
)

// HTTP const definitions
const (
	GET    string = http.MethodGet
	POST   string = http.MethodPost
	PUT    string = http.MethodPut
	PATCH  string = http.MethodPatch
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
	FormData    map[string]string
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
	logger             log.FieldLogger
	httpClient         *http.Client
	timeout            time.Duration
	dialer             util.Dialer
	singleEntryHostMap map[string]string
	singleURL          string
}

type configAgent struct {
	agentName         string
	environmentName   string
	isDocker          bool
	isGRPC            bool
	singleURL         string
	singleEntryFilter []string
}

var cfgAgent *configAgent
var cfgAgentMutex *sync.Mutex

func init() {
	cfgAgent = &configAgent{}
	cfgAgentMutex = &sync.Mutex{}
}

// SetConfigAgent -
func SetConfigAgent(env string, isGRPC, isDocker bool, agentName, singleURL string, singleEntryFilter []string) {
	cfgAgentMutex.Lock()
	defer cfgAgentMutex.Unlock()
	cfgAgent.environmentName = env
	cfgAgent.isGRPC = isGRPC
	cfgAgent.isDocker = isDocker
	cfgAgent.agentName = agentName
	cfgAgent.singleURL = singleURL
	if cfgAgent.singleEntryFilter != nil {
		cfgAgent.singleEntryFilter = append(cfgAgent.singleEntryFilter, singleEntryFilter...)
	} else {
		cfgAgent.singleEntryFilter = singleEntryFilter
	}
}

type ClientOpt func(*httpClient)

func WithTimeout(timeout time.Duration) func(*httpClient) {
	return func(h *httpClient) {
		h.timeout = timeout
	}
}

func WithSingleURL() func(*httpClient) {
	return func(h *httpClient) {
		h.singleURL = ""
		if cfgAgent != nil {
			h.singleURL = cfgAgent.singleURL
			if h.singleURL != "" {
				h.singleEntryHostMap = initializeSingleEntryMapping(h.singleURL, cfgAgent.singleEntryFilter)
			}
		}
	}
}

// NewClient - creates a new HTTP client
func NewClient(tlsCfg config.TLSConfig, proxyURL string, options ...ClientOpt) Client {
	timeout := getTimeoutFromEnvironment()
	client := newClient(timeout)

	for _, o := range options {
		o(client)
	}

	client.initialize(tlsCfg, proxyURL)
	return client
}

// NewClientWithTimeout - creates a new HTTP client, with a timeout
func NewClientWithTimeout(tlsCfg config.TLSConfig, proxyURL string, timeout time.Duration) Client {
	log.DeprecationWarningReplace("NewClientWithTimeout", "NewClient and WithTimeout optional func")
	return NewClient(tlsCfg, proxyURL, WithTimeout(timeout))
}

// NewSingleEntryClient - creates a new HTTP client for single entry point with a timeout
func NewSingleEntryClient(tlsCfg config.TLSConfig, proxyURL string, timeout time.Duration) Client {
	log.DeprecationWarningReplace("NewSingleEntryClient", "NewClient and WithSingleURL optional func")
	return NewClient(tlsCfg, proxyURL, WithTimeout(timeout), WithSingleURL())
}

func newClient(timeout time.Duration) *httpClient {
	return &httpClient{
		timeout: timeout,
		logger: log.NewFieldLogger().
			WithComponent("httpClient").
			WithPackage("sdk.api"),
	}
}

func initializeSingleEntryMapping(singleEntryURL string, singleEntryFilter []string) map[string]string {
	hostMapping := make(map[string]string)
	entryURL, err := url.Parse(singleEntryURL)
	if err == nil {
		for _, filteredURL := range singleEntryFilter {
			svcURL, err := url.Parse(filteredURL)
			if err == nil {
				hostMapping[util.ParseAddr(svcURL)] = util.ParseAddr(entryURL)
			}
		}
	}
	return hostMapping
}

func parseProxyURL(proxyURL string) *url.URL {
	if proxyURL != "" {
		pURL, err := url.Parse(proxyURL)
		if err == nil {
			return pURL
		}
		log.Errorf("Error parsing proxyURL from config; creating a non-proxy client: %s", err.Error())
	}
	return nil
}

func (c *httpClient) initialize(tlsCfg config.TLSConfig, proxyURL string) {
	c.httpClient = c.createClient(tlsCfg)
	if c.singleURL == "" && proxyURL == "" {
		return
	}

	c.dialer = util.NewDialer(parseProxyURL(proxyURL), c.singleEntryHostMap)
	c.httpClient.Transport.(*http.Transport).DialContext = c.httpDialer
}

func (c *httpClient) createClient(tlsCfg config.TLSConfig) *http.Client {
	if tlsCfg != nil {
		return c.createHTTPSClient(tlsCfg)
	}
	return c.createHTTPClient()
}

func (c *httpClient) createHTTPClient() *http.Client {
	httpClient := &http.Client{
		Transport: &http.Transport{},
		Timeout:   c.timeout,
	}
	return httpClient
}

func (c *httpClient) createHTTPSClient(tlsCfg config.TLSConfig) *http.Client {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg.BuildTLSConfig(),
		},
		Timeout: c.timeout,
	}
	return httpClient
}

func (c *httpClient) httpDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	return c.dialer.DialContext(ctx, network, addr)
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
	var req *http.Request
	var err error
	if request.FormData != nil {
		formData := make(url.Values)
		for k, v := range request.FormData {
			formData.Add(k, v)
		}

		req, err = http.NewRequestWithContext(ctx, request.Method, requestURL, strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequestWithContext(ctx, request.Method, requestURL, bytes.NewBuffer(request.Body))
	}

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
		cfgAgentMutex.Lock()
		defer cfgAgentMutex.Unlock()
		deploymentType := "binary"
		if cfgAgent.isDocker {
			deploymentType = "docker"
		}
		ua := fmt.Sprintf("%s/%s SDK/%s %s %s %s", config.AgentTypeName, config.AgentVersion, config.SDKVersion, cfgAgent.environmentName, cfgAgent.agentName, deploymentType)
		if cfgAgent.isGRPC {
			ua = fmt.Sprintf("%s reactive", ua)
		}
		req.Header.Set("User-Agent", ua)
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
		if err != nil {
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
	receivedData := int64(0)
	defer func() {
		duration := time.Since(startTime)
		targetURL := req.URL.String()
		if c.dialer != nil {
			svcHost := util.ParseAddr(req.URL)
			if entryHost, ok := c.singleEntryHostMap[svcHost]; ok {
				targetURL = req.URL.Scheme + "://" + entryHost + req.URL.Path
			}
		}

		logger := c.logger.
			WithField("id", reqID).
			WithField("method", req.Method).
			WithField("status", statusCode).
			WithField("duration(ms)", duration.Milliseconds()).
			WithField("url", targetURL)

		if req.ContentLength > 0 {
			logger = logger.WithField("sent(bytes)", req.ContentLength)
		}

		if receivedData > 0 {
			logger = logger.WithField("received(bytes)", receivedData)
		}

		if err != nil {
			logger.WithError(err).
				Trace("request failed")
		} else {
			logger.Trace("request succeeded")
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
	receivedData = res.ContentLength
	parseResponse, err := c.prepareAPIResponse(res, timer)

	return parseResponse, err
}
