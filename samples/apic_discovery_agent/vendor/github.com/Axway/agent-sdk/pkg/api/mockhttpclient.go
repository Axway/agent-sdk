package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// MockHTTPClient - use for mocking the HTTP client
type MockHTTPClient struct {
	Client
	Response      *Response // this for if you want to set your own dummy response
	ResponseCode  int       // this for if you only care about a particular response code
	ResponseError error

	RespCount int
	Responses []MockResponse

	Requests []Request // lists all requests the client has received
}

// MockResponse - use for mocking the MockHTTPClient responses
type MockResponse struct {
	FileName  string
	RespCode  int
	ErrString string
}

// SetResponse -
// if you care about the response content and the code, pass both in
// if you only care about the code, pass "" for the filepath
func (c *MockHTTPClient) SetResponse(filepath string, code int) {
	var dat []byte
	if filepath != "" {
		responseFile, err := os.Open(filepath)
		if err != nil {
			c.ResponseCode = http.StatusInternalServerError
			return
		}
		dat, err = ioutil.ReadAll(responseFile)
		if err != nil {
			c.ResponseCode = http.StatusInternalServerError
			return
		}
	}

	c.Response = &Response{
		Code:    code,
		Body:    dat,
		Headers: map[string][]string{},
	}
}

// SetResponses -
// if you care about the response content and the code, pass both in
// if you only care about the code, pass "" for the filepath
func (c *MockHTTPClient) SetResponses(responses []MockResponse) {
	c.RespCount = 0
	c.Responses = responses
}

// Send -
func (c *MockHTTPClient) Send(request Request) (*Response, error) {
	c.Requests = append(c.Requests, request)
	if c.Responses != nil && len(c.Responses) > 0 {
		return c.sendMultiple(request)
	}
	if c.Response != nil {
		return c.Response, nil
	}
	if c.ResponseError != nil {
		return nil, c.ResponseError
	}
	if c.ResponseCode != 0 {
		return &Response{
			Code: c.ResponseCode,
		}, nil
	}
	return nil, nil
}

func (c *MockHTTPClient) sendMultiple(request Request) (*Response, error) {
	responseFile, _ := os.Open(c.Responses[c.RespCount].FileName) // APIC Environments
	dat, _ := ioutil.ReadAll(responseFile)

	response := Response{
		Code:    c.Responses[c.RespCount].RespCode,
		Body:    dat,
		Headers: map[string][]string{},
	}

	var err error
	if c.Responses[c.RespCount].ErrString != "" {
		err = fmt.Errorf(c.Responses[c.RespCount].ErrString)
	}
	c.RespCount++
	return &response, err
}
