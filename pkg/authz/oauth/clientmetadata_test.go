package oauth

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestClientMetadataSerialization(t *testing.T) {
	c, err := NewClientMetadataBuilder().
		SetClientName("test").
		SetGrantTypes([]string{GrantTypeClientCredentials, GrantTypeAuthorizationCode}).
		SetTokenEndpointAuthMethod(config.ClientSecretJWT).
		SetResponseType([]string{AuthResponseToken}).
		SetRedirectURIs([]string{"http://localhost"}).
		SetScopes([]string{"scope1", "scope2"}).
		SetLogoURI("http://localhost").
		SetJWKSURI("http://localhost").
		SetExtraProperties(map[string]interface{}{"key": "value", "boolKey": true}).
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

	// Validate extra properties
	assert.NotNil(t, c.GetExtraProperties())
	assert.NotNil(t, scm.GetExtraProperties())
	assert.Equal(t, len(c.GetExtraProperties()), len(scm.GetExtraProperties()))
	assert.Equal(t, 2, len(scm.GetExtraProperties()))

	// Validate string value in extra properties
	assert.Contains(t, scm.GetExtraProperties(), "key")
	assert.Equal(t, "value", scm.GetExtraProperties()["key"])

	// Validate boolean value in extra properties
	assert.Contains(t, scm.GetExtraProperties(), "boolKey")
	assert.Equal(t, true, scm.GetExtraProperties()["boolKey"])

}
