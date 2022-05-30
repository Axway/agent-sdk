package registration

import (
	"fmt"
	"time"

	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

var providerMap map[string]Provider
var tokenEndpointProviderMap map[string]Provider

func init() {
	providerMap = make(map[string]Provider)
	tokenEndpointProviderMap = make(map[string]Provider)
}

// RegisterProvider - registers the IdP provider using the config
func RegisterProvider(idp corecfg.IDPConfig, tlsCfg corecfg.TLSConfig, proxyURL string, clientTimeout time.Duration) error {
	p, err := NewProvider(idp, tlsCfg, proxyURL, clientTimeout)
	if err != nil {
		return err
	}

	providerMap[idp.GetIDPName()] = p
	tokenEndpointProviderMap[p.GetTokenEndpoint()] = p
	return nil
}

// RegisterClient - registers the client using the registered provider
func RegisterClient(name string, client Client) (Client, error) {
	p, ok := providerMap[name]
	if ok {
		return p.RegisterClient(client)
	}
	return nil, fmt.Errorf("unrecognized credential provider with name %s", name)
}

// GetProviderByName - returns the provider based on lookup by nme
func GetProviderByName(name string) (Provider, error) {
	p, ok := providerMap[name]
	if ok {
		return p, nil
	}
	return nil, fmt.Errorf("unrecognized credential provider with name %s", name)
}

// GetProviderByTokenEndpoint - returns the provider based on lookup by nme
func GetProviderByTokenEndpoint(tokenEP string) (Provider, error) {
	p, ok := tokenEndpointProviderMap[tokenEP]
	if ok {
		return p, nil
	}
	return nil, fmt.Errorf("unrecognized credential provider with token endpoint %s", tokenEP)
}

// UnregisterClient - removes the client using the registered provider
func UnregisterClient(providerName string, clientID string) error {
	p, ok := providerMap[providerName]
	if ok {
		return p.UnregisterClient(clientID)
	}
	return fmt.Errorf("unrecognized credential provider with name %s", providerName)
}
