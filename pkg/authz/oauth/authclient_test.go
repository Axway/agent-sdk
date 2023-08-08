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
		WithClientSecretAuth("invalid_client", "invalid-secrt", ""))
	assert.Nil(t, err)
	assert.NotNil(t, ac)

	_, err = ac.GetToken()
	assert.NotNil(t, err)

	privateKey, _ := util.ReadPrivateKeyFile("testdata/private_key.pem", "")
	publicKey, _ := util.ReadPublicKeyBytes("testdata/publickey")
	s.SetTokenResponse("", 0, http.StatusBadRequest)
	ac, err = NewAuthClient(s.GetTokenURL(), apiClient,
		WithServerName("testServer"),
		WithKeyPairAuth("invalid_client", "", "", privateKey, publicKey, ""))
	assert.Nil(t, err)
	assert.NotNil(t, ac)

	_, err = ac.GetToken()
	assert.NotNil(t, err)

	s.SetTokenResponse("token", 3*time.Second, http.StatusOK)
	ac, err = NewAuthClient(s.GetTokenURL(), apiClient,
		WithServerName("testServer"),
		WithKeyPairAuth("invalid_client", "", "", privateKey, publicKey, ""))
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
		WithClientSecretAuth("invalid_client", "invalid-secrt", ""))

	assert.Nil(t, err)
	assert.NotNil(t, ac)

	_, err = ac.GetToken()
	assert.NotNil(t, err)
}
