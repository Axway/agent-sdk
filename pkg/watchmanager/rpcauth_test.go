package watchmanager

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_rpcAuth(t *testing.T) {
	testCases := []struct {
		name        string
		tenantID    string
		tokenFunc   func() (string, error)
		expectError bool
	}{
		{
			name:        "valid token",
			tenantID:    "123",
			tokenFunc:   getToken,
			expectError: false,
		},
		{
			name:        "token function error",
			tenantID:    "123",
			tokenFunc:   func() (string, error) { return "", fmt.Errorf("token error") },
			expectError: true,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cred := newRPCAuth(test.tenantID, test.tokenFunc)
			headers, err := cred.GetRequestMetadata(context.Background())
			if test.expectError {
				assert.NotNil(t, err)
				assert.Nil(t, headers)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, "Bearer abc", headers["authorization"])
				assert.Equal(t, "123", headers["x-axway-tenant-id"])
			}
			assert.False(t, cred.RequireTransportSecurity())
		})
	}
}

func getToken() (string, error) {
	return "abc", nil
}
