package registration

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
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
	UnregisterClient(clientId string) error
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type provider struct {
	providerType       ProviderType
	cfg                config.IDPConfig
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
		cfg:             idp,
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

	// Default the values from config if not set on the request
	if len(clientRequest.GetScopes()) == 0 {
		clientRequest.Scope = strings.Split(p.cfg.GetClientScopes(), " ")
	}

	if len(clientRequest.GetGrantTypes()) == 0 {
		clientRequest.GrantTypes = []string{p.cfg.GetGrantType()}
	}

	if clientRequest.TokenEndpointAuthMethod == "" {
		clientRequest.TokenEndpointAuthMethod = p.cfg.GetAuthMethod()
	}

	if len(clientRequest.ResponseTypes) == 0 {
		clientRequest.ResponseTypes = []string{p.cfg.GetAuthResponseType()}
	}

	clientRequest.JwksURI = p.authServerMetadata.JwksURI

	clientRequest.extraProperties = p.extraProperties
	if p.providerType == Okta {
		if len(clientRequest.extraProperties) == 0 {
			clientRequest.extraProperties = make(map[string]string)
		}
		_, ok := clientRequest.extraProperties["application_tye"]
		if !ok {
			clientRequest.extraProperties["application_tye"] = "service"
		}
	}

	clientBuffer, err := json.Marshal(clientRequest)
	if err != nil {
		return nil, err
	}

	token, err := p.getClientToken()
	if err != nil {
		return nil, err
	}

	header := map[string]string{
		"Authorization": authPrefix + " " + token,
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

func (p *provider) UnregisterClient(clientID string) error {
	authPrefix := p.getAuthorizationHeaderPrefix()
	token, err := p.getClientToken()
	if err != nil {
		return err
	}
	header := map[string]string{
		"Authorization": authPrefix + " " + token,
		"Content-Type":  "application/json",
	}

	request := api.Request{
		Method:  api.DELETE,
		URL:     p.authServerMetadata.RegistrationEndpoint + "/" + clientID,
		Headers: header,
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return err
	}

	if response.Code != 204 {
		return fmt.Errorf("error status code: %d, body: %s", response.Code, string(response.Body))
	}
	return nil
}

func (p *provider) getClientToken() (string, error) {
	token := p.cfg.GetAuthConfig().GetAccessToken()
	if p.cfg.GetAuthConfig().GetType() == "client" {
		tokenURL := p.authServerMetadata.TokenEndpoint

		data := url.Values{
			"client_id":     []string{p.cfg.GetAuthConfig().GetClientID()},
			"client_secret": []string{p.cfg.GetAuthConfig().GetClientSecret()},
			"grant_type":    []string{"client_credentials"},
			// "scope":         []string{"client-manage"},
		}
		bufBody := data.Encode()
		fmt.Println(bufBody)

		req := api.Request{
			Method: api.POST,
			URL:    tokenURL,
			Body:   []byte(bufBody),
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
		}

		res, err := p.apiClient.Send(req)
		if err != nil {
			return "", err
		}
		fmt.Println(string(res.Body))
		tok := tokenResponse{}
		if err := json.Unmarshal(res.Body, &tok); err != nil {
			return "", err
		}
		token = tok.AccessToken
	}
	return token, nil
}
