package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	coreapi "github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// ProviderType - type of provider
type ProviderType int

// Provider - interface for external IdP provider
type Provider interface {
	GetName() string
	GetTitle() string
	GetIssuer() string
	GetTokenEndpoint() string
	GetMTLSTokenEndpoint() string
	GetAuthorizationEndpoint() string
	GetSupportedScopes() []string
	GetSupportedGrantTypes() []string
	GetSupportedTokenAuthMethods() []string
	GetSupportedResponseMethod() []string
	RegisterClient(clientMetadata ClientMetadata) (ClientMetadata, error)
	UnregisterClient(clientID, accessToken string) error
	Validate() error
	GetConfig() corecfg.IDPConfig
	GetMetadata() *AuthorizationServerMetadata
}

type provider struct {
	logger             log.FieldLogger
	cfg                corecfg.IDPConfig
	metadataURL        string
	extraProperties    map[string]string
	requestHeaders     map[string]string
	queryParameters    map[string]string
	apiClient          coreapi.Client
	authServerMetadata *AuthorizationServerMetadata
	authClient         AuthClient
	idpType            typedIDP
}

type typedIDP interface {
	getAuthorizationHeaderPrefix() string
	preProcessClientRequest(clientRequest *clientMetadata)
}

type providerOptions struct {
	authServerMetadata *AuthorizationServerMetadata
}

func WithAuthServerMetadata(metadata *AuthorizationServerMetadata) func(*providerOptions) {
	return func(p *providerOptions) {
		p.authServerMetadata = metadata
	}
}

// NewProvider - create a new IdP provider
func NewProvider(idp corecfg.IDPConfig, tlsCfg corecfg.TLSConfig, proxyURL string, clientTimeout time.Duration, opts ...func(*providerOptions)) (Provider, error) {
	logger := log.NewFieldLogger().
		WithComponent("provider").
		WithPackage("sdk.agent.authz.oauth")

	pOpts := &providerOptions{}
	for _, opt := range opts {
		opt(pOpts)
	}

	apiClient := coreapi.NewClient(tlsCfg, proxyURL, coreapi.WithTimeout(clientTimeout))
	var idpType typedIDP
	switch idp.GetIDPType() {
	case TypeOkta:
		idpType = &okta{}
	default: // keycloak, generic
		idpType = &genericIDP{}
	}

	p := &provider{
		logger:             logger,
		metadataURL:        idp.GetMetadataURL(),
		cfg:                idp,
		extraProperties:    idp.GetExtraProperties(),
		requestHeaders:     idp.GetRequestHeaders(),
		queryParameters:    idp.GetQueryParams(),
		apiClient:          apiClient,
		idpType:            idpType,
		authServerMetadata: pOpts.authServerMetadata,
	}

	if p.authServerMetadata == nil {
		metadata, err := p.fetchMetadata()
		if err != nil {
			p.logger.
				WithField("provider", p.cfg.GetIDPName()).
				WithField("type", p.cfg.GetIDPType()).
				WithField("metadata-url", p.metadataURL).
				WithError(err).
				Error("unable to fetch OAuth authorization server metadata")
			return nil, err
		}

		p.authServerMetadata = metadata
	}

	// No OAuth client is needed to request token for access token based authentication to IdP
	if p.cfg.GetAuthConfig() != nil && p.cfg.GetAuthConfig().GetType() != corecfg.AccessToken {
		authClient, err := p.createAuthClient()
		if err != nil {
			return nil, err
		}
		p.authClient = authClient
	}
	return p, nil
}

func FetchMetadata(apiClient api.Client, metadataURL string) (*AuthorizationServerMetadata, error) {
	if apiClient == nil || metadataURL == "" {
		return nil, errors.New("unexpected arguments")
	}
	request := coreapi.Request{
		Method: coreapi.GET,
		URL:    metadataURL,
	}

	response, err := apiClient.Send(request)
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

func (p *provider) fetchMetadata() (*AuthorizationServerMetadata, error) {
	return FetchMetadata(p.apiClient, p.metadataURL)
}

func (p *provider) createAuthClient() (AuthClient, error) {
	switch p.cfg.GetAuthConfig().GetType() {
	case corecfg.Client:
		fallthrough
	case corecfg.ClientSecretPost:
		return p.createClientSecretPostAuthClient()
	case corecfg.ClientSecretBasic:
		return p.createClientSecretBasicAuthClient()
	case corecfg.ClientSecretJWT:
		return p.createClientSecretJWTAuthClient()
	case corecfg.PrivateKeyJWT:
		return p.createPrivateKeyJWTAuthClient()
	case corecfg.TLSClientAuth:
		fallthrough
	case corecfg.SelfSignedTLSClientAuth:
		return p.createTLSAuthClient()
	default:
		return nil, fmt.Errorf("%s", "unknown IdP auth type")
	}
}

func (p *provider) createClientSecretPostAuthClient() (AuthClient, error) {
	return NewAuthClient(p.GetTokenEndpoint(), p.apiClient,
		WithServerName(p.cfg.GetIDPName()),
		WithRequestHeaders(p.cfg.GetAuthConfig().GetRequestHeaders()),
		WithQueryParams(p.cfg.GetAuthConfig().GetQueryParams()),
		WithClientSecretPostAuth(p.cfg.GetAuthConfig().GetClientID(), p.cfg.GetAuthConfig().GetClientSecret(), p.cfg.GetAuthConfig().GetClientScope()))
}

func (p *provider) createClientSecretBasicAuthClient() (AuthClient, error) {
	return NewAuthClient(p.GetTokenEndpoint(), p.apiClient,
		WithServerName(p.cfg.GetIDPName()),
		WithRequestHeaders(p.cfg.GetAuthConfig().GetRequestHeaders()),
		WithQueryParams(p.cfg.GetAuthConfig().GetQueryParams()),
		WithClientSecretBasicAuth(p.cfg.GetAuthConfig().GetClientID(), p.cfg.GetAuthConfig().GetClientSecret(), p.cfg.GetAuthConfig().GetClientScope()))
}

func (p *provider) createClientSecretJWTAuthClient() (AuthClient, error) {
	return NewAuthClient(p.GetTokenEndpoint(), p.apiClient,
		WithServerName(p.cfg.GetIDPName()),
		WithRequestHeaders(p.cfg.GetAuthConfig().GetRequestHeaders()),
		WithQueryParams(p.cfg.GetAuthConfig().GetQueryParams()),
		WithClientSecretJwtAuth(
			p.cfg.GetAuthConfig().GetClientID(),
			p.cfg.GetAuthConfig().GetClientSecret(),
			p.cfg.GetAuthConfig().GetClientScope(),
			p.cfg.GetAuthConfig().GetClientID(),
			p.authServerMetadata.Issuer,
			p.cfg.GetAuthConfig().GetTokenSigningMethod(),
		))
}

func (p *provider) createPrivateKeyJWTAuthClient() (AuthClient, error) {
	keyReader := NewKeyReader(
		p.cfg.GetAuthConfig().GetPrivateKey(),
		p.cfg.GetAuthConfig().GetPublicKey(),
		p.cfg.GetAuthConfig().GetKeyPassword(),
	)
	privateKey, keyErr := keyReader.GetPrivateKey()
	if keyErr != nil {
		return nil, keyErr
	}

	publicKey, keyErr := keyReader.GetPublicKey()
	if keyErr != nil {
		return nil, keyErr
	}
	return NewAuthClient(p.GetTokenEndpoint(), p.apiClient,
		WithServerName(p.cfg.GetIDPName()),
		WithRequestHeaders(p.cfg.GetAuthConfig().GetRequestHeaders()),
		WithQueryParams(p.cfg.GetAuthConfig().GetQueryParams()),
		WithKeyPairAuth(
			p.cfg.GetAuthConfig().GetClientID(),
			p.cfg.GetAuthConfig().GetClientID(),
			p.authServerMetadata.Issuer,
			privateKey,
			publicKey,
			p.cfg.GetAuthConfig().GetClientScope(),
			p.cfg.GetAuthConfig().GetTokenSigningMethod(),
		),
	)
}

func (p *provider) createTLSAuthClient() (AuthClient, error) {
	return NewAuthClient(p.GetMTLSTokenEndpoint(), p.apiClient,
		WithServerName(p.cfg.GetIDPName()),
		WithRequestHeaders(p.cfg.GetAuthConfig().GetRequestHeaders()),
		WithQueryParams(p.cfg.GetAuthConfig().GetQueryParams()),
		WithTLSClientAuth(p.cfg.GetAuthConfig().GetClientID(), p.cfg.GetAuthConfig().GetClientScope()))
}

// GetName - returns the name of the provider
func (p *provider) GetName() string {
	return p.cfg.GetIDPName()
}

// GetTitle - returns the friendly name of the provider
func (p *provider) GetTitle() string {
	return p.cfg.GetIDPTitle()
}

// GetIssuer - returns the issuer for the provider
func (p *provider) GetIssuer() string {
	if p.authServerMetadata != nil {
		return p.authServerMetadata.Issuer
	}
	return ""
}

func (p *provider) useTLSAuth() bool {
	if p.cfg.GetAuthConfig() == nil {
		return false
	}
	return p.cfg.GetAuthConfig().GetType() == corecfg.TLSClientAuth || p.cfg.GetAuthConfig().GetType() == corecfg.SelfSignedTLSClientAuth
}

// GetTokenEndpoint - return the token endpoint URL
func (p *provider) GetTokenEndpoint() string {
	return p.authServerMetadata.TokenEndpoint
}

func (p *provider) GetMTLSTokenEndpoint() string {
	if p.authServerMetadata != nil {
		if p.authServerMetadata.MTLSEndPointAlias != nil && p.authServerMetadata.MTLSEndPointAlias.TokenEndpoint != "" {
			return p.authServerMetadata.MTLSEndPointAlias.TokenEndpoint
		}
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

func (p *provider) getClientRegistrationEndpoint() string {
	registrationEndpoint := p.authServerMetadata.RegistrationEndpoint
	if p.useTLSAuth() &&
		p.authServerMetadata.MTLSEndPointAlias != nil && p.authServerMetadata.MTLSEndPointAlias.RegistrationEndpoint != "" {
		registrationEndpoint = p.authServerMetadata.MTLSEndPointAlias.RegistrationEndpoint
	}
	return registrationEndpoint
}

func (p *provider) prepareHeaders(authPrefix, token string) map[string]string {
	headers := make(map[string]string)
	for key, value := range p.requestHeaders {
		headers[key] = value
	}
	headers[hdrAuthorization] = authPrefix + " " + token
	headers[hdrContentType] = mimeApplicationJSON
	return headers
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

	request := coreapi.Request{
		Method:      coreapi.POST,
		URL:         p.getClientRegistrationEndpoint(),
		QueryParams: p.queryParameters,
		Headers:     p.prepareHeaders(authPrefix, token),
		Body:        clientBuffer,
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return nil, err
	}

	if response.Code == http.StatusCreated || response.Code == http.StatusOK {
		clientRes := &clientMetadata{}
		err = json.Unmarshal(response.Body, clientRes)
		if !p.cfg.GetAuthConfig().UseRegistrationAccessToken() {
			clientRes.RegistrationAccessToken = ""
		}

		p.logger.
			WithField("provider", p.cfg.GetIDPName()).
			WithField("client-name", clientReq.GetClientName()).
			WithField("client-id", clientReq.GetClientName()).
			WithField("grant-type", clientReq.GetGrantTypes()).
			WithField("token-auth-method", clientReq.GetTokenEndpointAuthMethod()).
			WithField("response-type", clientReq.GetResponseTypes()).
			WithField("redirect-url", clientReq.GetRedirectURIs()).
			Info("registered client")
		return clientRes, err
	}

	err = fmt.Errorf("error status code: %d, body: %s", response.Code, string(response.Body))
	p.logger.
		WithField("provider", p.cfg.GetIDPName()).
		WithField("client-name", clientReq.GetClientName()).
		WithField("grant-type", clientReq.GetGrantTypes()).
		WithField("token-auth-method", clientReq.GetTokenEndpointAuthMethod()).
		WithField("response-type", clientReq.GetResponseTypes()).
		WithField("redirect-url", clientReq.GetRedirectURIs()).
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
	p.preProcessResponseType(clientRequest)
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
}

func (p *provider) preProcessResponseType(clientRequest *clientMetadata) {
	for _, grantTypes := range clientRequest.GrantTypes {
		switch grantTypes {
		case GrantTypeAuthorizationCode:
			if !hasResponseType(clientRequest, AuthResponseCode) {
				addResponseType(clientRequest, AuthResponseCode)
			}
		case GrantTypeImplicit:
			if !hasResponseType(clientRequest, AuthResponseToken) {
				addResponseType(clientRequest, AuthResponseToken)
			}
		}
	}
}

func hasResponseType(clientRequest *clientMetadata, responseType string) bool {
	for _, clientResponseType := range clientRequest.ResponseTypes {
		if clientResponseType == responseType {
			return true
		}
	}
	return false
}

func addResponseType(clientRequest *clientMetadata, responseType string) {
	if clientRequest.ResponseTypes == nil {
		clientRequest.ResponseTypes = make([]string, 0)
	}
	clientRequest.ResponseTypes = append(clientRequest.ResponseTypes, responseType)
}

// UnregisterClient - removes the OAuth client from IDP
func (p *provider) UnregisterClient(clientID, accessToken string) error {
	authPrefix := p.idpType.getAuthorizationHeaderPrefix()
	if accessToken == "" {
		token, err := p.getClientToken()
		if err != nil {
			return err
		}
		accessToken = token
	}

	request := coreapi.Request{
		Method:      coreapi.DELETE,
		URL:         p.getClientRegistrationEndpoint() + "/" + clientID,
		QueryParams: p.queryParameters,
		Headers:     p.prepareHeaders(authPrefix, accessToken),
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return err
	}

	if response.Code != http.StatusNoContent {
		err := fmt.Errorf("error status code: %d, body: %s", response.Code, string(response.Body))
		p.logger.
			WithField("provider", p.cfg.GetIDPName()).
			WithField("client-id", clientID).
			Error(err.Error())
		return err
	}

	p.logger.
		WithField("provider", p.cfg.GetIDPName()).
		WithField("client-id", clientID).
		Info("unregistered client")
	return nil
}

func (p *provider) getClientToken() (string, error) {
	if p.authClient != nil {
		useTokenCache := p.cfg.GetAuthConfig().UseTokenCache()
		return p.authClient.FetchToken(useTokenCache)
	}
	return p.cfg.GetAuthConfig().GetAccessToken(), nil
}

func (p *provider) GetConfig() corecfg.IDPConfig {
	return p.cfg
}

func (p *provider) GetMetadata() *AuthorizationServerMetadata {
	return p.authServerMetadata
}

func (p *provider) Validate() error {
	// Validate fetching token using client id/secret with oauth flow
	// how to validate accessToken
	// validate if the auth used has authorization?
	_, err := p.getClientToken()
	if err != nil {
		return err
	}
	return nil
}
