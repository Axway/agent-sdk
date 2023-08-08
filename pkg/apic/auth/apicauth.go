// Package auth implements the apic service account token management.
// Contributed by Xenon team
package auth

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func closeHelper(closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Warnf("Failed to close: %v", err)
	}
}

// PlatformTokenGetter - Interface for token getter
type PlatformTokenGetter interface {
	tokenGetterCloser
}

// ApicAuth provides authentication methods for calls against APIC Cloud services.
type ApicAuth struct {
	tenantID string
	tokenGetterCloser
}

// Authenticate applies the authentication headers
func (aa *ApicAuth) Authenticate(hs HeaderSetter) error {
	token, err := aa.GetToken()
	if err != nil {
		return err
	}

	hs.SetHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	hs.SetHeader("X-Axway-Tenant-Id", aa.tenantID)

	return nil
}

// AuthenticateNet applies the authentication headers
func (aa *ApicAuth) AuthenticateNet(req *http.Request) error {
	return aa.Authenticate(NetHeaderSetter{req})
}

// NewWithStatic returns an ApicAuth that uses a fixed token
func NewWithStatic(tenantID, token string) *ApicAuth {
	return &ApicAuth{
		tenantID,
		staticTokenGetter(token),
	}
}

// NewWithFlow returns an ApicAuth that uses the axway authentication flow
func NewWithFlow(tenantID, privKey, publicKey, password, url, aud, clientID string, singleURL string, timeout time.Duration) *ApicAuth {
	return &ApicAuth{
		tenantID,
		tokenGetterWithChannel(NewPlatformTokenGetter(privKey, publicKey, password, url, aud, clientID, singleURL, timeout)),
	}
}

// NewPlatformTokenGetter returns a token getter for axway ID
func NewPlatformTokenGetter(privKey, publicKey, password, url, aud, clientID string, singleURL string, timeout time.Duration) PlatformTokenGetter {
	cfg := config.NewCentralConfig(config.GenericService)
	centralCfg, _ := cfg.(*config.CentralConfiguration)
	centralCfg.SingleURL = singleURL

	acfg := cfg.GetAuthConfig()
	authCfg, _ := acfg.(*config.AuthConfiguration)
	authCfg.ClientID = clientID
	authCfg.PrivateKey = privKey
	authCfg.PublicKey = publicKey
	authCfg.KeyPwd = password
	authCfg.URL = url
	authCfg.Timeout = timeout

	return NewPlatformTokenGetterWithCentralConfig(cfg)
}

// NewPlatformTokenGetterWithCentralConfig returns a token getter for axway ID
func NewPlatformTokenGetterWithCentralConfig(centralCfg config.CentralConfig) PlatformTokenGetter {
	return &platformTokenGetter{
		cfg: centralCfg,
		keyReader: oauth.NewKeyReader(
			centralCfg.GetAuthConfig().GetPrivateKey(),
			centralCfg.GetAuthConfig().GetPublicKey(),
			centralCfg.GetAuthConfig().GetKeyPassword()),
	}
}

type funcTokenGetter func() (string, error)

// GetToken returns the fixed token.
func (f funcTokenGetter) GetToken() (string, error) {
	return f()
}

func (f funcTokenGetter) Close() error {
	return nil
}

// staticTokenGetter returns a token getter with a fixed token
func staticTokenGetter(token string) funcTokenGetter {
	return funcTokenGetter(func() (string, error) { return token, nil })
}

// platformTokenGetter can get an access token from apic platform.
type platformTokenGetter struct {
	cfg           config.CentralConfig
	keyReader     oauth.KeyReader
	axwayIDClient oauth.AuthClient
}

// Close a PlatformTokenGetter
func (ptp *platformTokenGetter) Close() error {
	return nil
}

func (ptp *platformTokenGetter) initAxwayIDPClient() error {
	privateKey, err := ptp.keyReader.GetPrivateKey()
	if err != nil {
		return err
	}

	publicKey, err := ptp.keyReader.GetPublicKey()
	if err != nil {
		return err
	}

	apiClient := api.NewClient(
		ptp.cfg.GetTLSConfig(),
		ptp.cfg.GetProxyURL(),
		api.WithTimeout(ptp.cfg.GetAuthConfig().GetTimeout()),
		api.WithSingleURL())

	ptp.axwayIDClient, err = oauth.NewAuthClient(ptp.cfg.GetAuthConfig().GetTokenURL(), apiClient,
		oauth.WithServerName("AxwayId"),
		oauth.WithKeyPairAuth(
			ptp.cfg.GetAuthConfig().GetClientID(),
			"",
			ptp.cfg.GetAuthConfig().GetAudience(),
			privateKey,
			publicKey, ""))

	return err
}

// GetToken returns a token from cache if not expired or fetches a new token
func (ptp *platformTokenGetter) GetToken() (string, error) {
	if ptp.axwayIDClient == nil {
		err := ptp.initAxwayIDPClient()
		if err != nil {
			return "", err
		}
	}

	token, err := ptp.axwayIDClient.GetToken()
	if err != nil && strings.Contains(err.Error(), "bad response") {
		log.Debug("please check the value for CENTRAL_AUTH_URL: The Amplify login URL.  Otherwise, possibly a clock syncing issue. Please check NTP daemon, if being used, that is up and running correctly.")
	}
	return token, err
}

// TokenGetter provides a bearer token to be used in api calls. Might block
type TokenGetter interface {
	GetToken() (string, error)
}

// TokenGetterCloser can get a token and clean up resources if needed.
type tokenGetterCloser interface {
	TokenGetter
	Close() error
}

// NetHeaderSetter sets headers an a net/http request
type NetHeaderSetter struct {
	*http.Request
}

// SetHeader sets a header on a net/http request
func (nhs NetHeaderSetter) SetHeader(key, value string) {
	nhs.Header.Set(key, value)
}

// HeaderSetter sets a header for a request
type HeaderSetter interface {
	// SetHeader sets a header on a http request
	SetHeader(key, value string)
}

// channelTokenGetter uses a channel to ensure synchronized access to the wrapped token getter
type channelTokenGetter struct {
	tokenGetter tokenGetterCloser
	responses   chan struct {
		string
		error
	}
	requests chan struct{}
}

// tokenGetterWithChannel wraps a token getter in a channelTokenGetter
func tokenGetterWithChannel(tokenGetter tokenGetterCloser) *channelTokenGetter {
	requests := make(chan struct{})
	responses := make(chan struct {
		string
		error
	})

	ctg := &channelTokenGetter{tokenGetter, responses, requests}

	go ctg.loop()

	return ctg
}

// loop reads requests and responds with token from the embedded token getter
func (ctg *channelTokenGetter) loop() {
	defer close(ctg.responses)
	defer closeHelper(ctg.tokenGetter)
	for {
		if _, ok := <-ctg.requests; !ok { // wait for a request
			break // if input channel is closed, stop
		}

		t, err := ctg.tokenGetter.GetToken()
		ctg.responses <- struct { // send back a response
			string
			error
		}{t, err}

	}
}

func (ctg *channelTokenGetter) GetToken() (string, error) {
	ctg.requests <- struct{}{}
	resp, ok := <-ctg.responses
	if !ok {
		return "", fmt.Errorf("[apicauth] channelTokenGetter closed")
	}
	return resp.string, resp.error

}

func (ctg *channelTokenGetter) Close() error {
	close(ctg.requests)
	return nil
}

// tokenAuth -
type tokenAuth struct {
	tenantID       string
	tokenRequester TokenGetter
}

// Config the auth config
type Config struct {
	PrivateKey  string        `mapstructure:"private_key"`
	PublicKey   string        `mapstructure:"public_key"`
	KeyPassword string        `mapstructure:"key_password"`
	URL         string        `mapstructure:"url"`
	Audience    string        `mapstructure:"audience"`
	ClientID    string        `mapstructure:"client_id"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// NewTokenAuth Create a new auth token requester
func NewTokenAuth(ac Config, tenantID string) TokenGetter {
	instance := &tokenAuth{tenantID: tenantID}
	tokenURL := ac.URL + "/realms/Broker/protocol/openid-connect/token"
	aud := ac.URL + "/realms/Broker"

	cfg := &config.CentralConfiguration{}
	singleURL := cfg.GetSingleURL()

	instance.tokenRequester = NewPlatformTokenGetter(
		ac.PrivateKey,
		ac.PublicKey,
		ac.KeyPassword,
		tokenURL,
		aud,
		ac.ClientID,
		singleURL,
		ac.Timeout,
	)
	return instance
}

// GetToken gets a token
func (t tokenAuth) GetToken() (string, error) {
	return t.tokenRequester.GetToken()
}
