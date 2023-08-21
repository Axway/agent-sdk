package oauth

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestRegistration(t *testing.T) {
	s := NewMockIDPServer()
	defer s.Close()

	cfg := &config.IDPConfiguration{
		Name:        "test",
		Type:        "okta",
		MetadataURL: s.GetMetadataURL(),
		AuthConfig: &config.IDPAuthConfiguration{
			Type:         config.Client,
			ClientID:     "test",
			ClientSecret: "test",
		},
		GrantType:        GrantTypeClientCredentials,
		ClientScopes:     "read,write",
		AuthMethod:       config.ClientSecretBasic,
		AuthResponseType: "",
		ExtraProperties:  config.ExtraProperties{"key": "value"},
	}

	s.SetMetadataResponseCode(http.StatusBadRequest)
	p, err := NewProvider(cfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = NewProvider(cfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, p)

	c, err := NewClientMetadataBuilder().
		SetClientName("test").
		SetRedirectURIs([]string{"http://localhost"}).
		SetJWKSURI("http://localhost").
		SetGrantTypes([]string{"authorization_code"}).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, c)

	s.SetRegistrationResponseCode(http.StatusBadRequest)
	cr, err := p.RegisterClient(c)
	assert.NotNil(t, err)
	assert.Nil(t, cr)

	cr, err = p.RegisterClient(c)
	assert.Nil(t, err)
	assert.NotNil(t, cr)

	assert.Equal(t, c.GetClientName(), cr.GetClientName())
	assert.NotEmpty(t, cr.GetClientID())
	assert.NotEmpty(t, cr.GetClientSecret())
	assert.Equal(t, strings.Join(c.GetGrantTypes(), ","), strings.Join(cr.GetGrantTypes(), ","))
	assert.Equal(t, c.GetTokenEndpointAuthMethod(), cr.GetTokenEndpointAuthMethod())
	assert.Equal(t, strings.Join(c.GetResponseTypes(), ","), strings.Join(cr.GetResponseTypes(), ","))
	assert.Equal(t, strings.Join(c.GetRedirectURIs(), ","), strings.Join(cr.GetRedirectURIs(), ","))
	assert.Equal(t, strings.Join(c.GetScopes(), ","), strings.Join(cr.GetScopes(), ","))
	assert.Equal(t, c.GetJwksURI(), cr.GetJwksURI())
	assert.Equal(t, len(c.GetExtraProperties()), len(cr.GetExtraProperties()))

	s.SetRegistrationResponseCode(http.StatusBadRequest)
	err = p.UnregisterClient(cr.GetClientID())
	assert.NotNil(t, err)

	s.SetRegistrationResponseCode(http.StatusNoContent)
	err = p.UnregisterClient(cr.GetClientID())
	assert.Nil(t, err)
}
