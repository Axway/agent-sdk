package oauth

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestProvider(t *testing.T) {

	cases := []struct {
		name                       string
		idpType                    string
		clientRequest              *clientMetadata
		expectedClient             *clientMetadata
		metadataResponseCode       int
		registrationResponseCode   int
		unRegistrationResponseCode int
		expectMetadataErr          bool
		expectRegistrationErr      bool
		expectUnRegistrationErr    bool
	}{
		{
			name:                 "IDP metadata bad request",
			idpType:              "generic",
			metadataResponseCode: http.StatusBadRequest,
			expectMetadataErr:    true,
		},
		{
			name:    "registration bad request",
			idpType: "generic",
			clientRequest: &clientMetadata{
				ClientName: "test",
			},
			metadataResponseCode:     http.StatusOK,
			registrationResponseCode: http.StatusBadRequest,
			expectRegistrationErr:    true,
		},
		{
			name:    "unregistration bad request",
			idpType: "okta",
			clientRequest: &clientMetadata{
				ClientName:   "test",
				RedirectURIs: []string{"http://localhost"},
				JwksURI:      "http://jwks",
				GrantTypes:   []string{GrantTypeAuthorizationCode},
			},
			expectedClient: &clientMetadata{
				ClientName:              "test",
				RedirectURIs:            []string{"http://localhost"},
				JwksURI:                 "http://jwks",
				GrantTypes:              []string{GrantTypeAuthorizationCode},
				TokenEndpointAuthMethod: config.ClientSecretBasic,
				ResponseTypes:           []string{AuthResponseCode},
				Scope:                   []string{"read", "write"},
				extraProperties: map[string]string{
					"key":               "value",
					oktaApplicationType: oktaAppTypeWeb,
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusBadRequest,
			expectUnRegistrationErr:    true,
		},
		{
			name:    "successful create and delete client",
			idpType: "generic",
			clientRequest: &clientMetadata{
				ClientName:   "test",
				RedirectURIs: []string{"http://localhost"},
				JwksURI:      "http://jwks",
				GrantTypes:   []string{GrantTypeImplicit},
			},
			expectedClient: &clientMetadata{
				ClientName:              "test",
				RedirectURIs:            []string{"http://localhost"},
				JwksURI:                 "http://jwks",
				GrantTypes:              []string{GrantTypeImplicit},
				TokenEndpointAuthMethod: config.ClientSecretBasic,
				ResponseTypes:           []string{AuthResponseToken},
				Scope:                   []string{"read", "write"},
				extraProperties: map[string]string{
					"key": "value",
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
		{
			name:    "successful client_credential",
			idpType: "generic",
			clientRequest: &clientMetadata{
				ClientName:   "test",
				RedirectURIs: []string{"http://localhost"},
				JwksURI:      "http://jwks",
				GrantTypes:   []string{GrantTypeClientCredentials},
			},
			expectedClient: &clientMetadata{
				ClientName:              "test",
				RedirectURIs:            []string{"http://localhost"},
				JwksURI:                 "http://jwks",
				GrantTypes:              []string{GrantTypeClientCredentials},
				TokenEndpointAuthMethod: config.ClientSecretBasic,
				ResponseTypes:           []string{},
				Scope:                   []string{"read", "write"},
				extraProperties: map[string]string{
					"key": "value",
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewMockIDPServer()
			defer s.Close()
			idpCfg := &config.IDPConfiguration{
				Name: "test",
				Type: tc.idpType,
				AuthConfig: &config.IDPAuthConfiguration{
					Type:         config.Client,
					ClientID:     "test",
					ClientSecret: "test",
				},
				GrantType:       GrantTypeClientCredentials,
				ClientScopes:    "read,write",
				AuthMethod:      config.ClientSecretBasic,
				MetadataURL:     s.GetMetadataURL(),
				ExtraProperties: config.ExtraProperties{"key": "value"},
			}

			s.SetMetadataResponseCode(tc.metadataResponseCode)
			p, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 30*time.Second)
			if tc.expectMetadataErr {
				assert.NotNil(t, err)
				assert.Nil(t, p)
				return
			}

			assert.Nil(t, err)
			assert.NotNil(t, p)

			s.SetRegistrationResponseCode(tc.registrationResponseCode)
			cr, err := p.RegisterClient(tc.clientRequest)
			if tc.expectRegistrationErr {
				assert.NotNil(t, err)
				assert.Nil(t, cr)
				return
			}

			assert.Nil(t, err)
			assert.NotNil(t, cr)

			assert.Equal(t, tc.expectedClient.GetClientName(), cr.GetClientName())
			assert.NotEmpty(t, cr.GetClientID())
			assert.NotEmpty(t, cr.GetClientSecret())
			assert.Equal(t, strings.Join(tc.expectedClient.GetGrantTypes(), ","), strings.Join(cr.GetGrantTypes(), ","))
			assert.Equal(t, tc.expectedClient.GetTokenEndpointAuthMethod(), cr.GetTokenEndpointAuthMethod())
			assert.Equal(t, strings.Join(tc.expectedClient.GetResponseTypes(), ","), strings.Join(cr.GetResponseTypes(), ","))
			assert.Equal(t, strings.Join(tc.expectedClient.GetRedirectURIs(), ","), strings.Join(cr.GetRedirectURIs(), ","))
			assert.Equal(t, strings.Join(tc.expectedClient.GetScopes(), ","), strings.Join(cr.GetScopes(), ","))
			assert.Equal(t, tc.expectedClient.GetJwksURI(), cr.GetJwksURI())
			assert.Equal(t, len(tc.expectedClient.GetExtraProperties()), len(cr.GetExtraProperties()))

			s.SetRegistrationResponseCode(tc.unRegistrationResponseCode)
			err = p.UnregisterClient(cr.GetClientID(), cr.GetRegistrationAccessToken())
			if tc.expectUnRegistrationErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}

}
