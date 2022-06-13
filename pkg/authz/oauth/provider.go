package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// ProviderType - type of provider
type ProviderType int

// Provider - interface for external IdP provider
type Provider interface {
	GetName() string
	GetIssuer() string
	GetTokenEndpoint() string
	GetAuthorizationEndpoint() string
	GetSupportedScopes() []string
	GetSupportedGrantTypes() []string
	GetSupportedTokenAuthMethods() []string
	GetSupportedResponseMethod() []string
	RegisterClient(clientMetadata ClientMetadata) (ClientMetadata, error)
	UnregisterClient(clientID string) error
}

type provider struct {
	logger             log.FieldLogger
	cfg                corecfg.IDPConfig
	metadataURL        string
	extraProperties    map[string]string
	apiClient          coreapi.Client
	authServerMetadata *AuthorizationServerMetadata
	authClient         AuthClient
	idpType            typedIDP
}

type typedIDP interface {
	getAuthorizationHeaderPrefix() string
	preProcessClientRequest(clientRequest *clientMetadata)
}

// NewProvider - create a new IdP provider
func NewProvider(idp corecfg.IDPConfig, tlsCfg corecfg.TLSConfig, proxyURL string, clientTimeout time.Duration) (Provider, error) {
	logger := log.NewFieldLogger().
		WithComponent("provider").
		WithPackage("sdk.agent.authz.oauth")

	apiClient := coreapi.NewClientWithTimeout(tlsCfg, proxyURL, clientTimeout)
	var idpType typedIDP
	switch idp.GetIDPType() {
	case TypeOkta:
		idpType = &okta{}
	default: // keycloak, generic
		idpType = &genericIDP{}
	}

	p := &provider{
		logger:          logger,
		metadataURL:     idp.GetMetadataURL(),
		cfg:             idp,
		extraProperties: idp.GetExtraProperties(),
		apiClient:       apiClient,
		idpType:         idpType,
	}

	metadata, err := p.fetchMetadata()
	if err != nil {
		p.logger.
			WithField("name", p.cfg.GetIDPName()).
			WithField("type", p.cfg.GetIDPType()).
			WithField("metadataUrl", p.metadataURL).
			WithError(err).
			Error("unable to fetch OAuth authorization server metadata")
		return nil, err
	}

	p.authServerMetadata = metadata
	if p.cfg.GetAuthConfig() != nil && p.cfg.GetAuthConfig().GetType() == IDPAuthTypeClient {
		p.authClient, err = NewAuthClient(p.authServerMetadata.TokenEndpoint, apiClient,
			WithServerName(idp.GetIDPName()),
			WithClientSecretAuth(p.cfg.GetAuthConfig().GetClientID(), p.cfg.GetAuthConfig().GetClientSecret()))
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *provider) fetchMetadata() (*AuthorizationServerMetadata, error) {
	request := coreapi.Request{
		Method: coreapi.GET,
		URL:    p.metadataURL,
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code == http.StatusOK {
		authSrvMetadata := &AuthorizationServerMetadata{}
		err = json.Unmarshal(response.Body, authSrvMetadata)
		return authSrvMetadata, err
	}
	return nil, fmt.Errorf("error fetching metadata status code: %d, body: %s", response.Code, string(response.Body))

}

// GetName - returns the name of the provider
func (p *provider) GetName() string {
	return p.cfg.GetIDPName()
}

// GetIssuer - returns the issuer for the provider
func (p *provider) GetIssuer() string {
	if p.authServerMetadata != nil {
		return p.authServerMetadata.Issuer
	}
	return ""
}

// GetTokenEndpoint - return the token endpoint URL
func (p *provider) GetTokenEndpoint() string {
	if p.authServerMetadata != nil {
		return p.authServerMetadata.TokenEndpoint
	}
	return ""
}

// GetAuthorizationEndpoint - return authorization endpoint
func (p *provider) GetAuthorizationEndpoint() string {
	if p.authServerMetadata != nil {
		return p.authServerMetadata.AuthorizationEndpoint
	}
	return ""
}

// GetSupportedScopes - returns the global scopes supported by provider
func (p *provider) GetSupportedScopes() []string {
	if p.authServerMetadata != nil {
		return p.authServerMetadata.ScopesSupported
	}
	return []string{""}
}

// GetSupportedGrantTypes - returns the grant type supported by provider
func (p *provider) GetSupportedGrantTypes() []string {
	if p.authServerMetadata != nil {
		return p.authServerMetadata.GrantTypesSupported
	}
	return []string{""}
}

// GetSupportedTokenAuthMethods - returns the token auth method supported by provider
func (p *provider) GetSupportedTokenAuthMethods() []string {
	if p.authServerMetadata != nil {
		return p.authServerMetadata.TokenEndpointAuthMethodSupported
	}
	return []string{""}

}

// GetSupportedResponseMethod - returns the token response method supported by provider
func (p *provider) GetSupportedResponseMethod() []string {
	if p.authServerMetadata != nil {
		return p.authServerMetadata.ResponseTypesSupported
	}
	return []string{""}
}

// RegisterClient - register the OAuth client with IDP
func (p *provider) RegisterClient(clientReq ClientMetadata) (ClientMetadata, error) {
	authPrefix := p.idpType.getAuthorizationHeaderPrefix()
	err := p.enrichClientReq(clientReq)
	if err != nil {
		return nil, err
	}

	clientBuffer, err := json.Marshal(clientReq)
	if err != nil {
		return nil, err
	}

	token, err := p.getClientToken()
	if err != nil {
		return nil, err
	}

	header := map[string]string{
		hdrAuthorization: authPrefix + " " + token,
		hdrContentType:   mimeApplicationJSON,
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     p.authServerMetadata.RegistrationEndpoint,
		Headers: header,
		Body:    clientBuffer,
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code == http.StatusCreated || response.Code == http.StatusOK {
		clientRes := &clientMetadata{}
		err = json.Unmarshal(response.Body, clientRes)
		p.logger.
			WithField("provider", p.cfg.GetIDPName()).
			WithField("clientName", clientReq.GetClientName()).
			WithField("grantType", clientReq.GetGrantTypes()).
			WithField("tokenAuthMethod", clientReq.GetTokenEndpointAuthMethod()).
			WithField("responseType", clientReq.GetResponseTypes()).
			WithField("redirectURL", clientReq.GetRedirectURIs()).
			Info("registered client")
		return clientRes, err
	}

	err = fmt.Errorf("error status code: %d, body: %s", response.Code, string(response.Body))
	p.logger.
		WithField("clientName", clientReq.GetClientName()).
		WithField("grantType", clientReq.GetGrantTypes()).
		WithField("tokenAuthMethod", clientReq.GetTokenEndpointAuthMethod()).
		WithField("responseType", clientReq.GetResponseTypes()).
		WithField("redirectURL", clientReq.GetRedirectURIs()).
		Error(err.Error())

	return nil, err
}

func (p *provider) enrichClientReq(clientReq ClientMetadata) error {
	clientRequest, ok := clientReq.(*clientMetadata)
	if !ok {
		return fmt.Errorf("unrecognized client request metadata")
	}

	p.applyClientDefaults(clientRequest)

	clientRequest.extraProperties = p.extraProperties

	p.idpType.preProcessClientRequest(clientRequest)

	return nil
}

func (p *provider) applyClientDefaults(clientRequest *clientMetadata) {
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
}

// UnregisterClient - removes the OAuth client from IDP
func (p *provider) UnregisterClient(clientID string) error {
	authPrefix := p.idpType.getAuthorizationHeaderPrefix()
	token, err := p.getClientToken()
	if err != nil {
		return err
	}
	header := map[string]string{
		hdrAuthorization: authPrefix + " " + token,
		hdrContentType:   mimeApplicationJSON,
	}

	request := coreapi.Request{
		Method:  coreapi.DELETE,
		URL:     p.authServerMetadata.RegistrationEndpoint + "/" + clientID,
		Headers: header,
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return err
	}

	if response.Code != http.StatusNoContent {
		err := fmt.Errorf("error status code: %d, body: %s", response.Code, string(response.Body))
		p.logger.
			WithField("provider", p.cfg.GetIDPName()).
			WithField("clientName", clientID).
			Error(err.Error())
		return err
	}

	p.logger.
		WithField("provider", p.cfg.GetIDPName()).
		WithField("clientName", clientID).
		Info("unregistered client")
	return nil
}

func (p *provider) getClientToken() (string, error) {
	if p.authClient != nil {
		return p.authClient.GetToken()
	}
	return p.cfg.GetAuthConfig().GetAccessToken(), nil
}
