package watchmanager

import (
	"context"
)

type rpcCredential interface {
	GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error)
	RequireTransportSecurity() bool
}

type rpcAuth struct {
	tenantID    string
	tokenGetter TokenGetter
}

// newRPCAuth - Create a new RPC authenticator which uses the token getter to fetch token
func newRPCAuth(tenantID string, tokenGetter TokenGetter) rpcCredential {
	auth := rpcAuth{
		tenantID:    tenantID,
		tokenGetter: tokenGetter,
	}
	return auth
}

// GetRequestMetadata - sets the http header prior to transport
func (t rpcAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	token, err := t.tokenGetter()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"x-axway-tenant-id": t.tenantID,
		"authorization":     "Bearer " + token,
	}, nil
}

// RequireTransportSecurity - leave as false for now
func (rpcAuth) RequireTransportSecurity() bool {
	return false
}
