package api

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util"
	log "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
)

// HTTP const definitions
const (
	GET    string = http.MethodGet
	POST   string = http.MethodPost
	PUT    string = http.MethodPut
	DELETE string = http.MethodDelete
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
}

// NewClient - creates a new HTTP client
func NewClient(cfg config.TLSConfig, proxyURL string) Client {
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

	httpCli.Timeout = time.Second * 10
	return &httpClient{
		httpClient: httpCli,
	}
}

func (c *httpClient) getURLEncodedQueryParams(queryParams map[string]string) string {
	params := url.Values{}
	for key, value := range queryParams {
		params.Add(key, value)
	}
	return params.Encode()
}

func (c *httpClient) prepareAPIRequest(request Request) (*http.Request, error) {
	requestURL := request.URL
	if len(request.QueryParams) != 0 {
		requestURL += "?" + c.getURLEncodedQueryParams(request.QueryParams)
	}
	req, err := http.NewRequest(request.Method, requestURL, bytes.NewBuffer(request.Body))
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
		req.Header.Set("User-Agent", config.AgentTypeName+"/"+config.AgentVersion)
	}
	return req, err
}

func (c *httpClient) prepareAPIResponse(res *http.Response) (*Response, error) {
	body, err := ioutil.ReadAll(res.Body)
	response := Response{
		Code:    res.StatusCode,
		Body:    body,
		Headers: res.Header,
	}
	return &response, err
}

// Send - send the http request and returns the API Response
func (c *httpClient) Send(request Request) (*Response, error) {
	req, err := c.prepareAPIRequest(request)
	if err != nil {
		return nil, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return c.prepareAPIResponse(res)
}
