package oauth

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientMetadataSerialization(t *testing.T) {
	c, err := NewClientMetadataBuilder().
		SetClientName("test").
		SetGrantTypes([]string{"client_credentials", "authorization_code"}).
		SetTokenEndpointAuthMethod("client_secret_jwt").
		SetResponseType([]string{"token"}).
		SetRedirectURIs([]string{"http://localhost"}).
		SetScopes([]string{"scope1", "scope2"}).
		SetLogoURI("http://localhost").
		SetJWKSURI("http://localhost").
		SetExtraProperties(map[string]string{"key": "value"}).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, c)

	cm := c.(*clientMetadata)
	cm.ClientID = "test"
	cm.ClientSecret = "test"
	var now Time
	cm.ClientIDIssuedAt = &now
	cm.ClientSecretExpiresAt = &now

	buf, err := json.Marshal(c)
	assert.Nil(t, err)
	assert.NotNil(t, buf)

	scm := &clientMetadata{}
	err = json.Unmarshal(buf, scm)
	assert.Nil(t, err)

	assert.Equal(t, c.GetClientName(), scm.GetClientName())
	assert.Equal(t, c.GetClientID(), scm.GetClientID())
	assert.Equal(t, c.GetClientSecret(), scm.GetClientSecret())
	assert.Equal(t, c.GetClientIDIssuedAt().Unix(), scm.GetClientIDIssuedAt().Unix())
	assert.Equal(t, c.GetClientSecretExpiresAt().Unix(), scm.GetClientSecretExpiresAt().Unix())
	assert.Equal(t, strings.Join(c.GetGrantTypes(), ","), strings.Join(scm.GetGrantTypes(), ","))
	assert.Equal(t, c.GetTokenEndpointAuthMethod(), scm.GetTokenEndpointAuthMethod())
	assert.Equal(t, strings.Join(c.GetResponseTypes(), ","), strings.Join(scm.GetResponseTypes(), ","))
	assert.Equal(t, strings.Join(c.GetRedirectURIs(), ","), strings.Join(scm.GetRedirectURIs(), ","))
	assert.Equal(t, strings.Join(c.GetScopes(), ","), strings.Join(scm.GetScopes(), ","))
	assert.Equal(t, c.GetLogoURI(), scm.GetLogoURI())
	assert.Equal(t, c.GetJwksURI(), scm.GetJwksURI())
	assert.Equal(t, len(c.GetExtraProperties()), len(scm.GetExtraProperties()))

}
