package registration

import (
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

var providerMap map[string]Provider

func init() {
	providerMap = make(map[string]Provider)
}

// RegisterProvider - registers the IdP provider using the config
func RegisterProvider(idp config.IDPConfig, tlsCfg corecfg.TLSConfig, proxyURL string, clientTimeout time.Duration) error {
	p, err := NewProvider(idp, tlsCfg, proxyURL, clientTimeout)
	if err != nil {
		return err
	}

	providerMap[idp.GetIDPName()] = p
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

// UnregisterClient - removes the client using the registered provider
func UnregisterClient(providerName string, clientId string) error {
	p, ok := providerMap[providerName]
	if ok {
		return p.UnregisterClient(clientId)
	}
	return fmt.Errorf("unrecognized credential provider with name %s", providerName)
}
