package oauth

import (
	"encoding/json"
	"errors"
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
	GetTitle() string
	GetIssuer() string
	GetTokenEndpoint() string
	GetMTLSTokenEndpoint() string
	GetAuthorizationEndpoint() string
	GetSupportedScopes() []string
	GetSupportedGrantTypes() []string
	GetSupportedTokenAuthMethods() []string
	GetSupportedResponseMethod() []string
	RegisterClient(clientMetadata ClientMetadata) (ClientMetadata, map[string]string, error)
	UnregisterClient(clientID, accessToken, registrationClientURI string, agentDetails map[string]string) error
	Validate() error
	GetConfig() corecfg.IDPConfig
	GetMetadata() *AuthorizationServerMetadata
}

type provider struct {
	logger             log.FieldLogger
	cfg                corecfg.IDPConfig
	metadataURL        string
	extraProperties    map[string]interface{}
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
	validateExtraProperties(extraProps map[string]interface{}) error
	postProcessClientRegistration(clientRes ClientMetadata, idp corecfg.IDPConfig, apiClient coreapi.Client) (map[string]string, error)
	postProcessClientUnregister(clientID string, agentDetails map[string]string, idp corecfg.IDPConfig, apiClient coreapi.Client) error
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

	extraProps := idp.GetExtraProperties()
	if extraProps == nil {
		extraProps = make(map[string]interface{})
	}

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
		extraProperties:    extraProps,
		requestHeaders:     idp.GetRequestHeaders(),
		queryParameters:    idp.GetQueryParams(),
		apiClient:          apiClient,
		idpType:            idpType,
		authServerMetadata: pOpts.authServerMetadata,
	}

	if p.authServerMetadata == nil {
		metadata, err := p.fetchMetadata()
		if err != nil {
			return nil, fmt.Errorf("unable to fetch OAuth authorization server metadata for provider %q: %w", p.cfg.GetIDPName(), err)
		}

		p.authServerMetadata = metadata
	}

	// Fail-fast Okta validation: if Okta group/policy is configured, verify the resources exist.
	if idp.GetIDPType() == TypeOkta {
		if err := validateOktaConfiguredResources(idp, apiClient); err != nil {
			return nil, fmt.Errorf("failed to validate Okta configuration for provider %q: %w", p.cfg.GetIDPName(), err)
		}
	}

	// No OAuth client is needed to request token for access token based authentication to IdP
	if p.cfg.GetAuthConfig() != nil && p.cfg.GetAuthConfig().GetType() != corecfg.AccessToken {
		authClient, err := p.createAuthClient()
		if err != nil {
			return nil, err
		}
		p.authClient = authClient
	}

	// Validate provider-specific extra properties
	if err := idpType.validateExtraProperties(p.extraProperties); err != nil {
		return nil, fmt.Errorf("invalid extra properties for %s provider: %w", idp.GetIDPType(), err)
	}

	return p, nil
}

func FetchMetadata(apiClient coreapi.Client, metadataURL string) (*AuthorizationServerMetadata, error) {
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
func (p *provider) RegisterClient(clientReq ClientMetadata) (ClientMetadata, map[string]string, error) {
	authPrefix := p.idpType.getAuthorizationHeaderPrefix()
	err := p.enrichClientReq(clientReq)
	if err != nil {
		return nil, nil, err
	}

	clientBuffer, err := json.Marshal(clientReq)
	if err != nil {
		return nil, nil, err
	}

	token, err := p.getClientToken()
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	if response.Code == http.StatusCreated || response.Code == http.StatusOK {
		clientRes := &clientMetadata{}
		err = json.Unmarshal(response.Body, clientRes)
		if !p.cfg.GetAuthConfig().UseRegistrationAccessToken() {
			clientRes.RegistrationAccessToken = ""
		}

		// Okta post-registration hook
		var createdAgentDetails map[string]string
		if p.cfg.GetIDPType() == TypeOkta {
			var hookErr error
			createdAgentDetails, hookErr = p.idpType.postProcessClientRegistration(clientRes, p.cfg, p.apiClient)
			if hookErr != nil {
				// If post-registration processing fails, attempt to delete
				// the newly created OAuth client to avoid leaving orphaned registrations behind.
				rollbackErr := p.rollbackRegisteredClient(clientRes, authPrefix, token)
				if rollbackErr != nil {
					return nil, nil, fmt.Errorf(
						"failed to complete Okta client setup for client %q. Manual cleanup in Okta may be required. setup error: %v; rollback error: %w",
						clientRes.GetClientID(),
						hookErr,
						rollbackErr,
					)
				}
				return nil, nil, fmt.Errorf("failed to complete Okta client setup: %w", hookErr)
			}
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
		return clientRes, createdAgentDetails, err
	}

	return nil, nil, fmt.Errorf("failed to register OAuth client for provider %q: status code %d, body: %s", p.cfg.GetIDPName(), response.Code, string(response.Body))
}

func (p *provider) rollbackRegisteredClient(clientRes ClientMetadata, authPrefix, fallbackToken string) error {
	if clientRes == nil {
		return nil
	}
	clientID := clientRes.GetClientID()
	if clientID == "" {
		return fmt.Errorf("registered client response missing client id")
	}

	rollbackToken := clientRes.GetRegistrationAccessToken()
	if rollbackToken == "" {
		rollbackToken = fallbackToken
	}

	logger := p.logger.
		WithField("provider", p.cfg.GetIDPName()).
		WithField("client-id", clientID)

	return p.attemptUnregisterAll(
		logger,
		clientID,
		clientRes.GetRegistrationClientURI(),
		authPrefix,
		rollbackToken,
	)
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
func (p *provider) UnregisterClient(clientID, accessToken, registrationClientURI string, agentDetails map[string]string) error {
	logger := p.logger.
		WithField("provider", p.cfg.GetIDPName()).
		WithField("client-id", clientID)

	logger.Debug("starting client unregistration")

	authPrefix := p.idpType.getAuthorizationHeaderPrefix()
	if accessToken == "" {
		token, err := p.getClientToken()
		if err != nil {
			return err
		}
		accessToken = token
	}

	// Run provider-specific cleanup first (for example, Okta group/policy unassignment), but always continue to OAuth client deletion so transient cleanup failures do not
	// leave active clients behind. Return whichever error(s) occurred after both steps.
	cleanupErr := p.runPostUnregisterHook(clientID, agentDetails)
	unregisterErr := p.attemptUnregisterAll(logger, clientID, registrationClientURI, authPrefix, accessToken)

	if unregisterErr == nil {
		logger.Info("unregistered client")
	}

	switch {
	case cleanupErr != nil && unregisterErr != nil:
		return fmt.Errorf("failed to fully remove the Okta client. Provider cleanup failed and OAuth client deletion failed. cleanup error: %v; delete error: %w", cleanupErr, unregisterErr)
	case unregisterErr != nil:
		return fmt.Errorf("failed to delete the OAuth client from the identity provider: %w", unregisterErr)
	case cleanupErr != nil:
		return fmt.Errorf("failed to complete provider cleanup after client unregistration. Manual cleanup in Okta may be required: %w", cleanupErr)
	default:
		return nil
	}
}

// runPostUnregisterHook calls provider-specific unregister hook when applicable.
func (p *provider) runPostUnregisterHook(clientID string, agentDetails map[string]string) error {
	if p.cfg.GetIDPType() != TypeOkta {
		return nil
	}
	return p.idpType.postProcessClientUnregister(clientID, agentDetails, p.cfg, p.apiClient)
}

// attemptUnregisterAll tries unregistering with the registration URI, the standard
// registration endpoint (base + /clientID) and finally as a query-parameter.

func (p *provider) attemptUnregisterAll(logger log.FieldLogger, clientID, registrationClientURI, authPrefix, accessToken string) error {
	var err error

	// Try with registration client URI if not empty
	if registrationClientURI != "" {
		err = p.tryUnregister(registrationClientURI, "", authPrefix, accessToken)
		if err == nil {
			return nil
		}
		logger.
			WithError(err).
			WithField("registration-uri", registrationClientURI).
			Trace("failed to unregister with registration client URI")
	}

	// Try with base url + clientID in path
	standardURL := p.getClientRegistrationEndpoint() + "/" + clientID
	if standardURL != registrationClientURI {
		err = p.tryUnregister(standardURL, "", authPrefix, accessToken)
		if err == nil {
			return nil
		}
		logger.
			WithError(err).
			WithField("unregister-url", standardURL).
			Trace("failed to unregister with standard URL")
	}

	// Try with clientID as query parameter
	baseURL := p.getClientRegistrationEndpoint()
	if baseURL != registrationClientURI {
		if strings.Contains(standardURL, "/"+clientID) {
			baseURL = strings.Replace(standardURL, "/"+clientID, "", 1)
		}
		err = p.tryUnregister(baseURL, clientID, authPrefix, accessToken)
		if err == nil {
			return nil
		}
		logger.
			WithError(err).
			WithField("unregister-url", baseURL).
			Trace("failed to unregister with clientID as query parameter")
	}

	return err
}

// tryUnregister attempts to unregister using the provided parameters
func (p *provider) tryUnregister(unregisterURL, clientID, authPrefix, accessToken string) error {
	queryParams := make(map[string]string)
	if clientID != "" {
		for k, v := range p.queryParameters {
			queryParams[k] = v
		}
		queryParams["client_id"] = clientID
	} else {
		queryParams = p.queryParameters
	}

	request := coreapi.Request{
		Method:      coreapi.DELETE,
		URL:         unregisterURL,
		QueryParams: queryParams,
		Headers:     p.prepareHeaders(authPrefix, accessToken),
	}

	response, err := p.apiClient.Send(request)
	if err != nil {
		return err
	}

	if response.Code == http.StatusNoContent {
		return nil
	}

	return fmt.Errorf("unregister failed with status code: %d, body: %s", response.Code, string(response.Body))
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
