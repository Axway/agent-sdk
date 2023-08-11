package oauth

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestClientBuilder(t *testing.T) {
	cm, err := NewClientMetadataBuilder().Build()
	assert.Nil(t, err)
	assert.NotNil(t, cm)
	assert.Equal(t, "", cm.GetClientName())
	assert.Equal(t, 0, len(cm.GetGrantTypes()))          // Provider will set default
	assert.Equal(t, "", cm.GetTokenEndpointAuthMethod()) // Provider will set default
	assert.Equal(t, 0, len(cm.GetRedirectURIs()))
	assert.Equal(t, 0, len(cm.GetScopes()))
	assert.Equal(t, "", cm.GetLogoURI())
	assert.Nil(t, cm.GetJwks())

	cm, err = NewClientMetadataBuilder().
		SetClientName("test").
		SetGrantTypes([]string{GrantTypeClientCredentials, GrantTypeAuthorizationCode}).
		SetTokenEndpointAuthMethod(config.ClientSecretPost).
		SetResponseType([]string{"token"}).
		SetRedirectURIs([]string{"http://localhost"}).
		SetScopes([]string{"scope1", "scope2"}).
		SetLogoURI("http://localhost").
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, cm)
	assert.Equal(t, "test", cm.GetClientName())
	assert.Equal(t, 2, len(cm.GetGrantTypes()))                            // Provider will set default
	assert.Equal(t, "client_secret_post", cm.GetTokenEndpointAuthMethod()) // Provider will set default
	assert.Equal(t, 1, len(cm.GetResponseTypes()))
	assert.Equal(t, 1, len(cm.GetRedirectURIs()))
	assert.Equal(t, 2, len(cm.GetScopes()))
	assert.Equal(t, "http://localhost", cm.GetLogoURI())
	assert.Nil(t, cm.GetJwks())

	publicKey, err := util.ReadPublicKeyBytes("testdata/publickey")
	assert.Nil(t, err)

	cm, err = NewClientMetadataBuilder().
		SetClientName("test").
		SetGrantTypes([]string{GrantTypeClientCredentials}).
		SetTokenEndpointAuthMethod(config.PrivateKeyJWT).
		SetJWKS(publicKey).
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, cm)
	assert.NotNil(t, cm.GetJwks())
}

func TestBuildValidations(t *testing.T) {
	client, err := NewClientMetadataBuilder().
		SetClientName("test").
		SetGrantTypes([]string{GrantTypeClientCredentials}).
		SetTokenEndpointAuthMethod(config.PrivateKeyJWT).
		SetJWKS([]byte("invalid-public-key")).
		Build()

	assert.NotNil(t, err)
	assert.Nil(t, client)

	publicKey, err := util.ReadPublicKeyBytes("testdata/publickey")
	assert.Nil(t, err)

	client, err = NewClientMetadataBuilder().
		SetClientName("test").
		SetGrantTypes([]string{GrantTypeClientCredentials}).
		SetTokenEndpointAuthMethod(config.PrivateKeyJWT).
		SetJWKS(publicKey).
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, client)
}
