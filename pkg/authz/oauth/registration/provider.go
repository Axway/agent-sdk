package registration

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/config"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

// ProviderType - type of provider
type ProviderType int

// Provider types
const (
	Generic ProviderType = iota + 1
	Okta
	KeyCloak
)

// Provider - interface for external IdP provider
type Provider interface {
	RegisterClient(clientMetadata Client) (Client, error)
}

type provider struct {
	providerType       ProviderType
	accessToken        string
	metadataURL        string
	extraProperties    map[string]string
	apiClient          coreapi.Client
	authServerMetadata *AuthorizationServerMetadata
}

// NewProvider - create a new IdP provider
func NewProvider(idp config.IDPConfig, tlsCfg corecfg.TLSConfig, proxyURL string, clientTimeout time.Duration) (Provider, error) {
	apiClient := coreapi.NewClientWithTimeout(tlsCfg, proxyURL, clientTimeout)

	providerType := Generic
	switch idp.GetIDPType() {
	case "keycloak":
		providerType = KeyCloak
	case "okta":
		providerType = Okta
	}

	p := &provider{
		providerType:    providerType,
		metadataURL:     idp.GetMetadataURL(),
		accessToken:     idp.GetAccessToken(),
		extraProperties: idp.GetExtraProperties(),
		apiClient:       apiClient,
	}
	metadata, err := p.fetchMetadata()
	if err != nil {
		return nil, err
	}

	p.authServerMetadata = metadata

	return p, nil
}

func (p *provider) fetchMetadata() (*AuthorizationServerMetadata, error) {
	request := api.Request{
		Method: api.GET,
		URL:    p.metadataURL,
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code < 400 {
		authSrvMetadata := &AuthorizationServerMetadata{}
		err = json.Unmarshal(response.Body, authSrvMetadata)
		return authSrvMetadata, err
	}
	return nil, fmt.Errorf("error status code: %d, body: %s", response.Code, string(response.Body))

}

func (p *provider) getAuthorizationHeaderPrefix() string {
	switch p.providerType {
	case Okta:
		return "SSWS"
	default:
		return "Bearer"
	}
}

func (p *provider) RegisterClient(clientReq Client) (Client, error) {
	authPrefix := p.getAuthorizationHeaderPrefix()
	clientRequest, ok := clientReq.(*client)
	if !ok {
		return nil, fmt.Errorf("unrecognized client request metadata")
	}
	clientRequest.JwksURI = p.authServerMetadata.JwksURI
	clientRequest.extraProperties = p.extraProperties
	clientBuffer, err := json.Marshal(clientRequest)
	if err != nil {
		return nil, err
	}

	header := map[string]string{
		"Authorization": authPrefix + " " + p.accessToken,
		"Content-Type":  "application/json",
	}

	request := api.Request{
		Method:  api.POST,
		URL:     p.authServerMetadata.RegistrationEndpoint,
		Headers: header,
		Body:    clientBuffer,
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code < 400 {
		clientRes := &client{}
		err = json.Unmarshal(response.Body, clientRes)
		return clientRes, err
	}
	return nil, fmt.Errorf("error status code: %d, body: %s", response.Code, string(response.Body))
}
