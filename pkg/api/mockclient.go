package api

import (
	"io/ioutil"
	"net/http"
	"os"
)

// MockClient - use for mocking the HTTP client
type MockClient struct {
	Client
	Response      *Response // this for if you want to set your own dummy response
	ResponseCode  int       // this for if only care about a particular response code
	ResponseError error
}

// SetResponse -
// if you care about the response content and the code, pass both in
// if you only care about the code, pass "" for the filepath
func (c *MockClient) SetResponse(filepath string, code int) {
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

// Send -
func (c *MockClient) Send(request Request) (*Response, error) {
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
