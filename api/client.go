package api

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
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

// Client - the http client to use when communicating to an API
type Client struct {
	httpClient *http.Client
}

// NewClient - creates a new API client using the http client sent in
func NewClient(cfg config.TLSConfig) *Client {
	var httpCli *http.Client

	if cfg == nil {
		httpCli = http.DefaultClient
	} else {
		httpCli = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: cfg.BuildTLSConfig(),
			},
		}
	}

	httpCli.Timeout = time.Second * 10
	return &Client{
		httpClient: httpCli,
	}
}

func (c *Client) getURLEncodedQueryParams(queryParams map[string]string) string {
	params := url.Values{}
	for key, value := range queryParams {
		params.Add(key, value)
	}
	return params.Encode()
}

func (c *Client) prepareAPIRequest(request Request) (*http.Request, error) {
	requestURL := request.URL
	if len(request.QueryParams) != 0 {
		requestURL += "?" + c.getURLEncodedQueryParams(request.QueryParams)
	}
	req, err := http.NewRequest(request.Method, requestURL, bytes.NewBuffer(request.Body))
	if err != nil {
		return req, err
	}
	for key, value := range request.Headers {
		req.Header.Set(key, value)
	}
	return req, err
}

func (c *Client) prepareAPIResponse(res *http.Response) (*Response, error) {
	body, err := ioutil.ReadAll(res.Body)
	response := Response{
		Code:    res.StatusCode,
		Body:    body,
		Headers: res.Header,
	}
	return &response, err
}

// Send - send the http request and returns the API Response
func (c *Client) Send(request Request) (*Response, error) {
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
