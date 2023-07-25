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
	apiClient         api.Client
	cachedToken       *tokenResponse
	cachedTokenExpiry *time.Timer
	getTokenMutex     *sync.Mutex
	options           *authClientOptions
	logger            log.FieldLogger
}

type authenticator interface {
	prepareRequest() (url.Values, error)
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

// WithClientSecretAuth - sets up to use client secret authenticator
func WithClientSecretAuth(clientID, clientSecret, scope string) AuthClientOption {
	return func(opt *authClientOptions) {
		opt.authenticator = &clientSecretAuthenticator{
			clientID,
			clientSecret,
			scope,
		}
	}
}

// WithKeyPairAuth - sets up to use public/private key pair authenticator
func WithKeyPairAuth(clientID, audience string, privKey *rsa.PrivateKey, publicKey []byte) AuthClientOption {
	return func(opt *authClientOptions) {
		opt.authenticator = &keyPairAuthenticator{
			clientID,
			audience,
			privKey,
			publicKey,
		}
	}
}

func (c *authClient) getCachedToken() string {
	if c.cachedToken != nil {
		select {
		case <-c.cachedTokenExpiry.C:
			// cleanup the token on expiry
			c.cachedToken = nil
			return ""
		default:
			return c.cachedToken.AccessToken
		}
	}
	return ""
}

// GetToken returns a token from cache if not expired or fetches a new token
func (c *authClient) GetToken() (string, error) {
	// only one GetToken should execute at a time
	c.getTokenMutex.Lock()
	defer c.getTokenMutex.Unlock()

	if token := c.getCachedToken(); token != "" {
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
	c.cachedTokenExpiry = time.NewTimer(time.Duration(almostExpires) * time.Second)
	return c.cachedToken.AccessToken, nil
}

func (c *authClient) getOAuthTokens() (*tokenResponse, error) {
	req, err := c.options.authenticator.prepareRequest()
	if err != nil {
		return nil, err
	}

	resp, err := c.postAuthForm(req)
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

func (c *authClient) postAuthForm(data url.Values) (resp *api.Response, err error) {
	req := api.Request{
		Method: api.POST,
		URL:    c.tokenURL,
		Body:   []byte(data.Encode()),
		Headers: map[string]string{
			hdrContentType: mimeApplicationFormURLEncoded,
		},
	}
	return c.apiClient.Send(req)
}
