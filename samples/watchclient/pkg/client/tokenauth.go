package client

import (
	"github.com/Axway/agent-sdk/pkg/apic/auth"
)

// tokenAuth implements the PerRPCCredentials interface
type tokenAuth struct {
	tenantID       string
	tokenRequester auth.PlatformTokenGetter
}

// newTokenAuth Create a new auth token requester
func newTokenAuth(ac AuthConfig, tenantID string) *tokenAuth {
	instance := &tokenAuth{tenantID: tenantID}
	tokenURL := ac.URL + "/realms/Broker/protocol/openid-connect/token"
	aud := ac.URL + "/realms/Broker"
	instance.tokenRequester = auth.NewPlatformTokenGetter(ac.PrivateKey,
		ac.PublicKey,
		ac.KeyPassword,
		tokenURL,
		aud,
		ac.ClientID,
		ac.Timeout,
	)
	return instance
}

func (t tokenAuth) GetToken() (string, error) {
	return t.tokenRequester.GetToken()
}
