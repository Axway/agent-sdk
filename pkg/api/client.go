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
	httpClient         *http.Client
	timeout            time.Duration
	dialer             util.Dialer
	singleEntryHostMap map[string]string
}

type configAgent struct {
	agentName         string
	environmentName   string
	isDocker          bool
	singleURL         string
	singleEntryFilter []string
}

var cfgAgent *configAgent

func init() {
	cfgAgent = &configAgent{}
}

// SetConfigAgent -
func SetConfigAgent(env string, isDocker bool, agentName, singleURL string, singleEntryFilter []string) {
	cfgAgent.environmentName = env
	cfgAgent.isDocker = isDocker
	cfgAgent.agentName = agentName
	cfgAgent.singleURL = singleURL
	if cfgAgent.singleEntryFilter != nil {
		cfgAgent.singleEntryFilter = append(cfgAgent.singleEntryFilter, singleEntryFilter...)
	} else {
		cfgAgent.singleEntryFilter = singleEntryFilter
	}
}

// AddSingleEntryFilterURL - adds a url for single entry point filter
// TODO - move this method to client and update the single entry host mapping
func AddSingleEntryFilterURL(filterURL string) {
	if cfgAgent.singleEntryFilter != nil {
		cfgAgent.singleEntryFilter = append(cfgAgent.singleEntryFilter, filterURL)
	}
}

// NewClient - creates a new HTTP client
func NewClient(cfg config.TLSConfig, proxyURL string) Client {
	timeout := getTimeoutFromEnvironment()
	return NewClientWithTimeout(cfg, proxyURL, timeout)
}

// NewClientWithTimeout - creates a new HTTP client, with a timeout
func NewClientWithTimeout(tlsCfg config.TLSConfig, proxyURL string, timeout time.Duration) Client {
	client := &httpClient{
		timeout: timeout,
	}
	client.initialize(tlsCfg, proxyURL, "")

	return client
}

// NewSingleEntryClient - creates a new HTTP client for single entry point with a timeout
func NewSingleEntryClient(tlsCfg config.TLSConfig, proxyURL string, timeout time.Duration) Client {
	client := &httpClient{
		timeout: timeout,
	}
	if cfgAgent.singleURL != "" {
		client.singleEntryHostMap = initializeSingleEntryMapping(cfgAgent.singleURL, cfgAgent.singleEntryFilter)
	}

	client.initialize(tlsCfg, proxyURL, cfgAgent.singleURL)
	return client
}

func initializeSingleEntryMapping(singleEntryURL string, singleEntryFilter []string) map[string]string {
	hostMapping := make(map[string]string)
	entryURL, err := url.Parse(singleEntryURL)
	if err == nil {
		entryPort := util.ParsePort(entryURL)
		for _, filteredURL := range singleEntryFilter {
			svcURL, err := url.Parse(filteredURL)
			if err == nil {
				svcPort := util.ParsePort(svcURL)
				hostMapping[fmt.Sprintf("%s:%d", svcURL.Host, svcPort)] = fmt.Sprintf("%s:%d", entryURL.Host, entryPort)
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

func (c *httpClient) initialize(tlsCfg config.TLSConfig, proxyURL, singleEntryURL string) {
	c.httpClient = c.createClient(tlsCfg)
	if singleEntryURL == "" && proxyURL == "" {
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
		deploymentType := "binary"
		if cfgAgent.isDocker {
			deploymentType = "docker"
		}
		ua := fmt.Sprintf("%s/%s SDK/%s %s %s %s", config.AgentTypeName, config.AgentVersion, config.SDKVersion, cfgAgent.environmentName, cfgAgent.agentName, deploymentType)
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
	defer func() {
		duration := time.Since(startTime)
		targetURL := req.URL.String()
		if c.dialer != nil && (c.dialer.GetProxyScheme() != "socks5" && c.dialer.GetProxyScheme() != "socks5h") {
			svcHost := fmt.Sprintf("%s:%d", req.URL.Host, util.ParsePort(req.URL))
			if entryHost, ok := c.singleEntryHostMap[svcHost]; ok {
				targetURL = req.URL.Scheme + "://" + entryHost + req.URL.Path
			}
		}
		if err != nil {
			log.Tracef("[ID:%s] %s [%dms] - ERR - %s - %s", reqID, req.Method, duration.Milliseconds(), targetURL, err.Error())
		} else {
			log.Tracef("[ID:%s] %s [%dms] - %d - %s", reqID, req.Method, duration.Milliseconds(), statusCode, targetURL)
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
