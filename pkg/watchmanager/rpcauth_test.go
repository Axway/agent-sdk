package watchmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_rpcAuth(t *testing.T) {
	cred := newRPCAuth("123", getToken)

	headers, err := cred.GetRequestMetadata(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, headers["x-axway-tenant-id"], "123")
	assert.Equal(t, headers["authorization"], "Bearer abc")
}

func getToken() (string, error) {
	return "abc", nil
}
