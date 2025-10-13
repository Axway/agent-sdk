package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	log "github.com/Axway/agent-sdk/pkg/util/log"
)

// MockHTTPClient - use for mocking the HTTP client
type MockHTTPClient struct {
	Client
	Response      *Response // this for if you want to set your own dummy response
	ResponseCode  int       // this for if you only care about a particular response code
	ResponseError error

	RespCount int
	Responses []MockResponse
	Requests  []Request // lists all requests the client has received
	sync.Mutex
}

// MockResponse - use for mocking the MockHTTPClient responses
type MockResponse struct {
	FileName  string
	RespData  string
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
		dat, err = io.ReadAll(responseFile)
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
	c.Lock()
	defer c.Unlock()
	c.RespCount = 0
	c.Responses = responses
}

// Send -
func (c *MockHTTPClient) Send(request Request) (*Response, error) {
	c.Lock()
	defer c.Unlock()

	c.Requests = append(c.Requests, request)

	fmt.Printf("%v - %v\n", request.Method, request.URL)

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
	var err error
	if c.RespCount >= len(c.Responses) {
		err := fmt.Errorf("error: received more requests than saved responses. failed on request: %s", request.URL)
		log.Error(err)
		return nil, err
	}

	fileName := c.Responses[c.RespCount].FileName

	dat := []byte(c.Responses[c.RespCount].RespData)

	var responseFile *os.File

	if fileName != "" {
		responseFile, err = os.Open(fileName)
		if err != nil {
			return nil, err
		}

		dat, err = io.ReadAll(responseFile)
		if err != nil {
			return nil, err
		}
	}

	response := Response{
		Code:    c.Responses[c.RespCount].RespCode,
		Body:    dat,
		Headers: map[string][]string{},
	}

	if c.Responses[c.RespCount].ErrString != "" {
		err = errors.New(c.Responses[c.RespCount].ErrString)
	}
	c.RespCount++
	return &response, err
}
