package stream

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestResourceClientGet(t *testing.T) {
	tests := []struct {
		name      string
		code      int
		hasError  bool
		token     string
		tokenErr  error
		clientErr error
		tenantID  string
	}{
		{
			name:      "should retrieve a watch topic",
			code:      200,
			hasError:  false,
			token:     "Bearer token",
			tokenErr:  nil,
			clientErr: nil,
			tenantID:  "123",
		},
		{
			name:      "should return an error when the status code is not 200",
			code:      500,
			hasError:  true,
			token:     "Bearer token",
			tokenErr:  nil,
			clientErr: nil,
			tenantID:  "123",
		},
		{
			name:      "should return an error when the client returns an error",
			code:      200,
			hasError:  true,
			token:     "Bearer token",
			tokenErr:  nil,
			clientErr: fmt.Errorf("err"),
			tenantID:  "123",
		},
		{
			name:      "should return an error when the token getter returns error",
			code:      200,
			hasError:  true,
			token:     "Bearer token",
			tokenErr:  fmt.Errorf("err"),
			clientErr: nil,
			tenantID:  "123",
		},
		{
			name:      "should fail when the token is empty",
			code:      200,
			hasError:  true,
			token:     "",
			tokenErr:  fmt.Errorf("err"),
			clientErr: nil,
			tenantID:  "123",
		},
		{
			name:      "should fail when the tenant id is empty",
			code:      200,
			hasError:  true,
			token:     "",
			tokenErr:  fmt.Errorf("err"),
			clientErr: nil,
			tenantID:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &mockClient{
				t:    t,
				code: tc.code,
				err:  tc.clientErr,
				body: []byte("{}"),
			}
			getToken := &mockTokenGetter{
				token: tc.token,
				err:   tc.tokenErr,
			}
			rc := NewResourceClient("abc.com", tc.tenantID, c, getToken)
			ri, err := rc.Get("/mock/self/link")
			if tc.hasError == false {
				assert.Nil(t, err)
				assert.NotNil(t, ri)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestResourceClientCreate(t *testing.T) {
	tests := []struct {
		name      string
		code      int
		bts       []byte
		hasError  bool
		token     string
		tokenErr  error
		clientErr error
		tenantID  string
	}{
		{
			name:      "should call create and return a resource",
			code:      201,
			hasError:  false,
			bts:       []byte(`{ "name": "name", "title": "title" }`),
			token:     "Bearer token",
			tokenErr:  nil,
			clientErr: nil,
			tenantID:  "123",
		},
		{
			name:      "should return an error when the status code is not 201",
			code:      400,
			hasError:  true,
			bts:       []byte(`{ "name": "name", "title": "title" }`),
			token:     "Bearer token",
			tokenErr:  nil,
			clientErr: nil,
			tenantID:  "123",
		},
		{
			name:      "should return an error unable to retrieve a token",
			code:      0,
			hasError:  true,
			bts:       []byte(`{ "name": "name", "title": "title" }`),
			token:     "Bearer token",
			tokenErr:  fmt.Errorf("err"),
			clientErr: nil,
			tenantID:  "123",
		},
		{
			name:      "should return an error when Send returns an error",
			code:      0,
			hasError:  true,
			bts:       []byte(`{ "name": "name", "title": "title" }`),
			token:     "Bearer token",
			tokenErr:  nil,
			clientErr: fmt.Errorf("err"),
			tenantID:  "123",
		},
		{
			name:      "should return an error when unmarshalling fails",
			code:      201,
			hasError:  true,
			bts:       []byte(`nope`),
			token:     "Bearer token",
			tokenErr:  nil,
			clientErr: nil,
			tenantID:  "123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &mockClient{
				t:    t,
				code: tc.code,
				err:  tc.clientErr,
				body: tc.bts,
			}
			getToken := &mockTokenGetter{
				token: tc.token,
				err:   tc.tokenErr,
			}
			rc := NewResourceClient("abc.com", tc.tenantID, c, getToken)
			ri, err := rc.Create("/resource", tc.bts)
			if tc.hasError == false {
				assert.Nil(t, err)
				assert.NotNil(t, ri)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

type mockClient struct {
	t    *testing.T
	code int
	err  error
	body []byte
}

func (c mockClient) Send(req api.Request) (*api.Response, error) {
	assert.NotEmpty(c.t, req.Headers["Authorization"])
	assert.NotEmpty(c.t, req.Headers["X-Axway-Tenant-Id"])
	return &api.Response{
		Code: c.code,
		Body: c.body,
	}, c.err
}
