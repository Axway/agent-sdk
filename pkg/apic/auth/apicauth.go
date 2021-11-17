// Package auth implements the apic service account token management.
// Contributed by Xenon team
package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"

	"net/http"
	"net/url"

	"encoding/json"
	"encoding/pem"

	"math/big"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"

	jwt "github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

func closeHelper(closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Warnf("Failed to close: %v", closer)
	}
}

// PlatformTokenGetter - Interface for token getter
type PlatformTokenGetter interface {
	tokenGetterCloser
}

// ApicAuth provides authentication methods for calls agains APIC Cloud services.
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
func NewWithFlow(tenantID, privKey, publicKey, password, url, aud, clientID string, timeout time.Duration) *ApicAuth {
	return &ApicAuth{
		tenantID,
		tokenGetterWithChannel(NewPlatformTokenGetter(privKey, publicKey, password, url, aud, clientID, timeout)),
	}
}

// NewPlatformTokenGetter returns a token getter for axway ID
func NewPlatformTokenGetter(privKey, publicKey, password, url, aud, clientID string, timeout time.Duration) PlatformTokenGetter {
	return &platformTokenGetter{
		aud,
		clientID,
		&platformTokenGenerator{
			url:     url,
			timeout: timeout,
		},
		&keyReader{
			privKey:   privKey,
			publicKey: publicKey,
			password:  password,
		},
		&tokenHolder{},
		&sync.Mutex{},
	}
}

// NewPlatformTokenGetterWithCentralConfig returns a token getter for axway ID
func NewPlatformTokenGetterWithCentralConfig(centralCfg config.CentralConfig) PlatformTokenGetter {
	return &platformTokenGetter{
		centralCfg.GetAuthConfig().GetAudience(),
		centralCfg.GetAuthConfig().GetClientID(),
		&platformTokenGenerator{
			url:       centralCfg.GetAuthConfig().GetTokenURL(),
			timeout:   centralCfg.GetAuthConfig().GetTimeout(),
			tlsConfig: centralCfg.GetTLSConfig(),
			proxyURL:  centralCfg.GetProxyURL(),
		},
		&keyReader{
			privKey:   centralCfg.GetAuthConfig().GetPrivateKey(),
			publicKey: centralCfg.GetAuthConfig().GetPublicKey(),
			password:  centralCfg.GetAuthConfig().GetKeyPassword(),
		},
		&tokenHolder{},
		&sync.Mutex{},
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

type keyReader struct {
	privKey   string // path to rsa encoded private key, used to sign platform tokens
	publicKey string // path to the rsa encoded public key
	password  string // path to password for private key
}

// parseRSAPrivateKeyFromPEMWithBytePassword tries to parse an rsa private key using password as bytes
// inspired from jwt.ParseRSAPrivateKeyFromPEMWithPassword
func parseRSAPrivateKeyFromPEMWithBytePassword(key []byte, password []byte) (*rsa.PrivateKey, error) {
	var err error

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(key); block == nil {
		return nil, fmt.Errorf("[apicauth] key must be pem encoded")
	}

	var parsedKey interface{}

	var blockDecrypted []byte
	if blockDecrypted, err = x509.DecryptPEMBlock(block, password); err != nil {
		return nil, err
	}

	if parsedKey, err = x509.ParsePKCS1PrivateKey(blockDecrypted); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(blockDecrypted); err != nil {
			return nil, err
		}
	}

	var pkey *rsa.PrivateKey
	var ok bool
	if pkey, ok = parsedKey.(*rsa.PrivateKey); !ok {
		return nil, fmt.Errorf("[apicauth] not a private key")
	}

	return pkey, nil
}

func (kr *keyReader) getPrivateKey() (*rsa.PrivateKey, error) {
	keyBytes, err := ioutil.ReadFile(kr.privKey)
	if err != nil {
		return nil, err
	}
	// cleanup private key read bytes
	defer func() {
		for i := range keyBytes {
			keyBytes[i] = 0
		}
	}()

	if kr.password != "" {
		pwdBytes, err := kr.getPassword()
		if err != nil {
			return nil, err
		}
		// cleanup password bytes
		defer func() {
			for i := range pwdBytes {
				pwdBytes[i] = 0
			}
		}()

		if len(pwdBytes) > 0 {
			key, err := parseRSAPrivateKeyFromPEMWithBytePassword(keyBytes, pwdBytes)
			if err != nil {
				return nil, err
			}

			return key, nil

		}
		log.Debug("password file empty, assuming unencrypted key")
		return jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	}
	log.Debug("no password, assuming unencrypted key")
	return jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
}

// getPublicKey from the path provided
func (kr *keyReader) getPublicKey() ([]byte, error) {
	keyBytes, err := ioutil.ReadFile(kr.publicKey)
	if err != nil {
		return nil, err
	}
	return keyBytes, nil
}

func parseDER(publicKey []byte) ([]byte, error) {
	if b64key, err := base64.StdEncoding.DecodeString(string(publicKey)); err == nil {
		return b64key, nil
	}

	_, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		pemBlock, _ := pem.Decode(publicKey)
		if pemBlock == nil {
			return nil, errors.New("data in key was not valid")
		}
		if pemBlock.Type != "PUBLIC KEY" {
			return nil, errors.New("unsupported key type")
		}
		return pemBlock.Bytes, nil
	}
	return publicKey, nil
}

func computeKIDFromDER(publicKey []byte) (kid string, err error) {
	b64key, err := parseDER(publicKey)
	if err != nil {
		return "", err
	}
	h := sha256.New() // create new hash with sha256 checksum
	/* #nosec G104 */
	if _, err := h.Write(b64key); err != nil { // add b64key to hash
		return "", err
	}
	e := base64.StdEncoding.EncodeToString(h.Sum(nil)) // return string of base64 encoded hash
	kid = strings.Split(e, "=")[0]
	kid = strings.Replace(kid, "+", "-", -1)
	kid = strings.Replace(kid, "/", "_", -1)
	return
}

func (kr *keyReader) getPassword() ([]byte, error) {
	return ioutil.ReadFile(kr.password)
}

type platformTokenGenerator struct {
	url       string           // url for access token generation
	timeout   time.Duration    // timeout for the http request
	tlsConfig config.TLSConfig // TLS Config
	proxyURL  string           // ProxyURL
}

// prepareInitialToken prepares a token for an access request
func prepareInitialToken(privateKey interface{}, kid, clientID, aud string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
		Issuer:    fmt.Sprintf("urn:ietf:params:oauth:client-assertion-type:jwt-bearer:%s", clientID),
		Subject:   clientID,
		Audience:  aud,
		ExpiresAt: now.Add(60*time.Second).UnixNano() / 1e9,
		IssuedAt:  now.UnixNano() / 1e9,
		Id:        uuid.New().String(),
	})

	token.Header["kid"] = kid

	requestToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return requestToken, nil
}
func (ptg *platformTokenGenerator) getHTTPClient() http.Client {
	client := http.Client{Timeout: ptg.timeout}
	httpTransport := &http.Transport{}

	if ptg.tlsConfig != nil {
		httpTransport.TLSClientConfig = ptg.tlsConfig.BuildTLSConfig()
	}

	if ptg.proxyURL != "" {
		url, err := url.Parse(ptg.proxyURL)
		if err != nil {
			log.Errorf("Error parsing proxyURL from config; creating a non-proxy client: %s", err.Error())
		}
		httpTransport.Proxy = util.GetProxyURL(url)
	}
	client.Transport = httpTransport
	return client
}

func (ptg *platformTokenGenerator) getPlatformTokens(requestToken string) (*axwayTokenResponse, error) {
	startTime := time.Now()
	client := ptg.getHTTPClient()
	resp, err := client.PostForm(ptg.url, url.Values{
		"grant_type":            []string{"client_credentials"},
		"client_assertion_type": []string{"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      []string{requestToken},
	})

	duration := time.Now().Sub(startTime)
	if err != nil {
		log.Tracef("%s [%dms] - ERR - %s - %s", "POST", duration.Milliseconds(), ptg.url, err.Error())
	} else {
		log.Tracef("%s [%dms] - %d - %s", "POST", duration.Milliseconds(), resp.StatusCode, ptg.url)
	}

	if err != nil {
		return nil, err
	}

	defer closeHelper(resp.Body)
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Debugf("bad response from AxwayID: %s: %s, request time : %s", resp.Status, body, startTime.String())
		log.Debug("possibly a clock syncing issue. Please check NTP daemon, if being used, that is up and running correctly.")
		return nil, fmt.Errorf("bad response from AxwayId: %s", resp.Status)
	}

	tokens := axwayTokenResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	return &tokens, nil
}

type tokenHolder struct {
	tokens *axwayTokenResponse
	expiry *time.Timer
}

func (th *tokenHolder) getCachedToken() string {
	if th.tokens != nil {
		select {
		case <-th.expiry.C:
			// cleanup the token on expiry
			th.tokens = nil
			return ""
		default:
			return th.tokens.AccessToken
		}
	}
	return ""
}

// platformTokenGetter can get an access token from apic platform.
type platformTokenGetter struct {
	aud      string // audience for the token
	clientID string // id of the account
	*platformTokenGenerator
	*keyReader
	*tokenHolder
	getTokenMutex *sync.Mutex
}

// Close a PlatformTokenGetter
func (ptp *platformTokenGetter) Close() error {
	return nil
}

// fetchNewToken fetches a new token from the platform and updates the token cache.
func (ptp *platformTokenGetter) fetchNewToken() (string, error) {
	log.Trace("Get cached token is empty.  Try and fetch a new token")
	privateKey, err := ptp.getPrivateKey()
	if err != nil {
		return "", err
	}
	// cleanup memory used by decoded privatekey in a (futile) attempt to prevent heartbleed like attaks
	defer func() {
		for i := range privateKey.Primes {
			*(privateKey.Primes[i]) = big.Int{}
		}
		*(privateKey.D) = big.Int{}
		*privateKey.Precomputed.Dp = big.Int{}
		*privateKey.Precomputed.Dq = big.Int{}
		*privateKey.Precomputed.Qinv = big.Int{}
	}()

	publicKey, err := ptp.getPublicKey()
	if err != nil {
		return "", err
	}

	kid, err := computeKIDFromDER(publicKey)
	if err != nil {
		return "", err
	}

	requestToken, err := prepareInitialToken(privateKey, kid, ptp.clientID, ptp.aud)
	if err != nil {
		return "", err
	}

	tokens, err := ptp.getPlatformTokens(requestToken)
	if err != nil {
		return "", err
	}

	almostExpires := (tokens.ExpiresIn * 4) / 5

	ptp.tokenHolder = &tokenHolder{
		tokens,
		time.NewTimer(time.Duration(almostExpires) * time.Second),
	}

	return tokens.AccessToken, nil
}

// GetToken returns a token from cache if not expired or fetches a new token
func (ptp *platformTokenGetter) GetToken() (string, error) {
	// only one GetToken should execute at a time
	ptp.getTokenMutex.Lock()
	defer ptp.getTokenMutex.Unlock()

	if token := ptp.getCachedToken(); token != "" {
		return token, nil
	}

	// try fetching a new token
	return ptp.fetchNewToken()
}

type axwayTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
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
			break // if input channel is closed,s top
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
