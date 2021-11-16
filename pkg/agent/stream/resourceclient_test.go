package stream

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestNewResourceClient(t *testing.T) {
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
			}
			getToken := &mockTokenGetter{
				token: tc.token,
				err:   tc.tokenErr,
			}
			rc := newResourceClient("abc.com", tc.tenantID, c, getToken)
			ri, err := rc.get("/mock/self/link")
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
}

func (c mockClient) Send(req api.Request) (*api.Response, error) {
	assert.NotEmpty(c.t, req.Headers["authorization"])
	assert.NotEmpty(c.t, req.Headers["x-axway-tenant-id"])
	return &api.Response{
		Code: c.code,
		Body: []byte("{}"),
	}, c.err
}
