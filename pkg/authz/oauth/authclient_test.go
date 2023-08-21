package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestResponseDecode(t *testing.T) {
	fixture := `
{
  "access_token": "some_value",
  "expires_in": 1800,
  "refresh_expires_in": 21600,
  "refresh_token": "some_value",
  "token_type": "bearer",
  "not-before-policy": 1510148785,
  "session_state": "f4b0fe58-a6f7-4452-9010-3945a7ecd493"
}`

	tokens := &tokenResponse{}
	json.Unmarshal([]byte(fixture), tokens)
	if tokens.AccessToken != "some_value" {
		t.Error("unexpected access token value")
	}
	if tokens.ExpiresIn != 1800 {
		t.Error("unexpected expires in token")
	}
}

func TestEmptyTokenHolder(t *testing.T) {
	ac := &authClient{}

	if token := ac.getCachedToken(); token != "" {
		t.Error("unexpected token from cache")
	}
}

func TestExpiredTokenHolder(t *testing.T) {
	ac := &authClient{
		cachedToken: &tokenResponse{
			AccessToken: "some_token",
		},
		cachedTokenExpiry: time.NewTimer(0),
	}

	time.Sleep(time.Millisecond)

	if token := ac.getCachedToken(); token != "" {
		t.Error("unexpected token from cache")
	}
}

func TestGetPlatformTokensHttpError(t *testing.T) {
	s := NewMockIDPServer()
	defer s.Close()

	apiClient := api.NewClient(config.NewTLSConfig(), "")
	s.SetTokenResponse("", 0, http.StatusBadRequest)
	ac, err := NewAuthClient(s.GetTokenURL(), apiClient,
		WithServerName("testServer"),
		WithClientSecretPostAuth("invalid_client", "invalid-secrt", ""))
	assert.Nil(t, err)
	assert.NotNil(t, ac)

	_, err = ac.GetToken()
	assert.NotNil(t, err)

	privateKey, _ := util.ReadPrivateKeyFile("testdata/private_key.pem", "")
	publicKey, _ := util.ReadPublicKeyBytes("testdata/publickey")
	s.SetTokenResponse("", 0, http.StatusBadRequest)
	ac, err = NewAuthClient(s.GetTokenURL(), apiClient,
		WithServerName("testServer"),
		WithKeyPairAuth("invalid_client", "", "", privateKey, publicKey, "", ""))
	assert.Nil(t, err)
	assert.NotNil(t, ac)

	_, err = ac.GetToken()
	assert.NotNil(t, err)

	s.SetTokenResponse("token", 3*time.Second, http.StatusOK)
	ac, err = NewAuthClient(s.GetTokenURL(), apiClient,
		WithServerName("testServer"),
		WithKeyPairAuth("invalid_client", "", "", privateKey, publicKey, "", ""))
	assert.Nil(t, err)
	assert.NotNil(t, ac)

	token, err := ac.GetToken()
	assert.Nil(t, err)
	assert.Equal(t, "token", token)
}

func TestGetPlatformTokensTimeout(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer s.Close()

	apiClient := api.NewClientWithTimeout(config.NewTLSConfig(), "", time.Second)
	ac, err := NewAuthClient(s.URL, apiClient,
		WithServerName("testServer"),
		WithClientSecretPostAuth("invalid_client", "invalid-secrt", ""))

	assert.Nil(t, err)
	assert.NotNil(t, ac)

	_, err = ac.GetToken()
	assert.NotNil(t, err)
}

func TestAuthClientTypes(t *testing.T) {
	s := NewMockIDPServer()
	defer s.Close()
	keyReader := NewKeyReader(
		"testdata/private_key.pem",
		"testdata/publickey",
		"",
	)
	privateKey, keyErr := keyReader.GetPrivateKey()
	assert.Nil(t, keyErr)

	publicKey, keyErr := keyReader.GetPublicKey()
	assert.Nil(t, keyErr)

	cases := []struct {
		name                             string
		tokenReqWithAuthorization        bool
		typedAuthOpt                     AuthClientOption
		expectedTokenReqClientID         string
		expectedTokenReqClientSecret     string
		expectedTokenReqClientAssertType string
		expectedTokenReqScope            string
	}{
		{
			name:                      "test",
			typedAuthOpt:              WithClientSecretBasicAuth("test-id", "test-secret", "test-scope"),
			tokenReqWithAuthorization: true,
			expectedTokenReqScope:     "test-scope",
		},
		{
			name:                         "test",
			typedAuthOpt:                 WithClientSecretPostAuth("test-id", "test-secret", "test-scope"),
			expectedTokenReqClientID:     "test-id",
			expectedTokenReqClientSecret: "test-secret",
			expectedTokenReqScope:        "test-scope",
		},
		{
			name:                             "test",
			typedAuthOpt:                     WithClientSecretJwtAuth("test-id", "test-secret", "test-scope", "", "aud", ""),
			expectedTokenReqClientID:         "test-id",
			expectedTokenReqScope:            "test-scope",
			expectedTokenReqClientAssertType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
		},
		{
			name:                             "test",
			typedAuthOpt:                     WithKeyPairAuth("test-id", "", "aud", privateKey, publicKey, "test-scope", ""),
			expectedTokenReqScope:            "test-scope",
			expectedTokenReqClientAssertType: "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
		},
		{
			name:                     "test",
			typedAuthOpt:             WithTLSClientAuth("test-id", "test-scope"),
			expectedTokenReqClientID: "test-id",
			expectedTokenReqScope:    "test-scope",
		},
	}
	for _, tc := range cases {
		s.SetTokenResponse("token", 3*time.Second, http.StatusOK)
		apiClient := api.NewClientWithTimeout(config.NewTLSConfig(), "", time.Second)
		opts := []AuthClientOption{WithServerName("testServer"), tc.typedAuthOpt}
		ac, err := NewAuthClient(s.GetTokenURL(), apiClient, opts...)

		assert.Nil(t, err)
		assert.NotNil(t, ac)

		token, err := ac.GetToken()
		assert.Nil(t, err)
		assert.Equal(t, "token", token)
		if tc.tokenReqWithAuthorization {
			headers := s.GetTokenRequestHeaders()
			authHeaderVal := headers.Get("Authorization")
			assert.NotEmpty(t, authHeaderVal)
		}
		tokenReqValues := s.GetTokenRequestValues()

		grantType := tokenReqValues.Get("grant_type")
		assert.Equal(t, "client_credentials", grantType)

		clientID := tokenReqValues.Get("client_id")
		assert.Equal(t, tc.expectedTokenReqClientID, clientID)

		clientSecret := tokenReqValues.Get("client_secret")
		assert.Equal(t, tc.expectedTokenReqClientSecret, clientSecret)

		clientAssertType := tokenReqValues.Get("client_assertion_type")
		assert.Equal(t, tc.expectedTokenReqClientAssertType, clientAssertType)

		tokenScope := tokenReqValues.Get("scope")
		assert.Equal(t, tc.expectedTokenReqScope, tokenScope)

	}
}
