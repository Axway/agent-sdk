package api

// MockClient - use for mocking the HTTP client
type MockClient struct {
	Client
	Response      *Response // this for if you want to set your own dummy response
	ResponseCode  int       // this for if only care about a particular response code
	ResponseError error
}

// func (c *MockClient) SetResponseCode(code int) {
// 	c.ResponseCode = code
// }

// func (c *MockClient) SetResponseError(rspError error) {
// 	c.ResponseError = rspError
// }

// func (c *MockClient) SetResponse(response *Response) {
// 	c.Response = response
// }

// Send -
func (c *MockClient) Send(request Request) (*Response, error) {
	if c.ResponseError != nil {
		return nil, c.ResponseError
	}
	if c.ResponseCode != 0 {
		return &Response{
			Code: c.ResponseCode,
		}, nil
	}
	if c.Response != nil {
		return c.Response, nil
	}
	return nil, nil
}
