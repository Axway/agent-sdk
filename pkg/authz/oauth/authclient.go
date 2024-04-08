package oauth

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// AuthClient - Interface representing the auth Client
type AuthClient interface {
	GetToken() (string, error)
	FetchToken(useCachedToken bool) (string, error)
}

// AuthClientOption - configures auth client.
type AuthClientOption func(*authClientOptions)

type authClientOptions struct {
	serverName    string
	authenticator authenticator
}

// authClient -
type authClient struct {
	tokenURL          string
	logger            log.FieldLogger
	apiClient         api.Client
	cachedToken       *tokenResponse
	getTokenMutex     *sync.Mutex
	options           *authClientOptions
	cachedTokenExpiry time.Time
}

type authenticator interface {
	prepareRequest() (url.Values, map[string]string, error)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// NewAuthClient - create a new auth client with client options
func NewAuthClient(tokenURL string, apiClient api.Client, opts ...AuthClientOption) (AuthClient, error) {
	logger := log.NewFieldLogger().
		WithComponent("authclient").
		WithPackage("sdk.agent.authz.oauth")
	client := &authClient{
		tokenURL:      tokenURL,
		apiClient:     apiClient,
		getTokenMutex: &sync.Mutex{},
		options:       &authClientOptions{},
		logger:        logger,
	}
	for _, o := range opts {
		o(client.options)
	}

	if client.options.serverName == "" {
		client.options.serverName = defaultServerName
	}
	if client.options.authenticator == nil {
		return nil, errors.New("unable to create client, no authenticator configured")
	}
	return client, nil
}

// WithServerName - sets up the server name in auth client
func WithServerName(serverName string) AuthClientOption {
	return func(opt *authClientOptions) {
		opt.serverName = serverName
	}
}

// WithClientSecretBasicAuth - sets up to use client secret basic authenticator
func WithClientSecretBasicAuth(clientID, clientSecret, scope string) AuthClientOption {
	return func(opt *authClientOptions) {
		opt.authenticator = &clientSecretBasicAuthenticator{
			clientID,
			clientSecret,
			scope,
		}
	}
}

// WithClientSecretPostAuth - sets up to use client secret authenticator
func WithClientSecretPostAuth(clientID, clientSecret, scope string) AuthClientOption {
	return func(opt *authClientOptions) {
		opt.authenticator = &clientSecretPostAuthenticator{
			clientID,
			clientSecret,
			scope,
		}
	}
}

// WithClientSecretJwtAuth - sets up to use client secret authenticator
func WithClientSecretJwtAuth(clientID, clientSecret, scope, issuer, aud, signingMethod string) AuthClientOption {
	return func(opt *authClientOptions) {
		opt.authenticator = &clientSecretJwtAuthenticator{
			clientID,
			clientSecret,
			scope,
			issuer,
			aud,
			signingMethod,
		}
	}
}

// WithKeyPairAuth - sets up to use public/private key pair authenticator
func WithKeyPairAuth(clientID, issuer, audience string, privKey *rsa.PrivateKey, publicKey []byte, scope, signingMethod string) AuthClientOption {
	return func(opt *authClientOptions) {
		opt.authenticator = &keyPairAuthenticator{
			clientID,
			issuer,
			audience,
			privKey,
			publicKey,
			scope,
			signingMethod,
		}
	}
}

// WithTLSClientAuth - sets up to use tls_client_auth and self_signed_tls_client_auth authenticator
func WithTLSClientAuth(clientID, scope string) AuthClientOption {
	return func(opt *authClientOptions) {
		opt.authenticator = &tlsClientAuthenticator{
			clientID: clientID,
			scope:    scope,
		}
	}
}

func (c *authClient) getCachedToken() string {
	if time.Now().After(c.cachedTokenExpiry) {
		c.cachedToken = nil
	}
	if c.cachedToken != nil {
		return c.cachedToken.AccessToken
	}
	return ""
}

// GetToken returns a token from cache if not expired or fetches a new token
func (c *authClient) GetToken() (string, error) {
	return c.FetchToken(true)
}

// GetToken returns a token from cache if not expired or fetches a new token
func (c *authClient) FetchToken(useCachedToken bool) (string, error) {
	// only one GetToken should execute at a time
	c.getTokenMutex.Lock()
	defer c.getTokenMutex.Unlock()
	token := c.getCachedToken()
	if useCachedToken && token != "" {
		return token, nil
	}

	// try fetching a new token
	return c.fetchNewToken()
}

// fetchNewToken fetches a new token from the platform and updates the token cache.
func (c *authClient) fetchNewToken() (string, error) {
	tokenResponse, err := c.getOAuthTokens()
	if err != nil {
		return "", err
	}

	almostExpires := (tokenResponse.ExpiresIn * 4) / 5

	c.cachedToken = tokenResponse
	c.cachedTokenExpiry = time.Now().Add(time.Duration(almostExpires) * time.Second)
	return c.cachedToken.AccessToken, nil
}

func (c *authClient) getOAuthTokens() (*tokenResponse, error) {
	req, headers, err := c.options.authenticator.prepareRequest()
	if err != nil {
		return nil, err
	}

	resp, err := c.postAuthForm(req, headers)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		err := fmt.Errorf("bad response from %s: %d %s", c.options.serverName, resp.Code, http.StatusText(resp.Code))
		c.logger.
			WithField("server", c.options.serverName).
			WithField("url", c.tokenURL).
			WithField("status", resp.Code).
			WithField("body", string(resp.Body)).
			WithError(err).
			Debug(err.Error())
		return nil, err
	}

	tokens := tokenResponse{}
	if err := json.Unmarshal(resp.Body, &tokens); err != nil {
		return nil, fmt.Errorf("unable to unmarshal token: %v", err)
	}

	return &tokens, nil
}

func (c *authClient) postAuthForm(data url.Values, headers map[string]string) (resp *api.Response, err error) {
	reqHeaders := map[string]string{
		hdrContentType: mimeApplicationFormURLEncoded,
	}
	for name, value := range headers {
		reqHeaders[name] = value
	}
	req := api.Request{
		Method:  api.POST,
		URL:     c.tokenURL,
		Body:    []byte(data.Encode()),
		Headers: reqHeaders,
	}
	return c.apiClient.Send(req)
}
