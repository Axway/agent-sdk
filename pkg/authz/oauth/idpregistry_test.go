package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/config"
)

func validMetadata() *AuthorizationServerMetadata {
	return &AuthorizationServerMetadata{
		Issuer:                "https://idp.example.com",
		AuthorizationEndpoint: "https://idp.example.com/auth",
		TokenEndpoint:         "https://idp.example.com/token",
		IntrospectionEndpoint: "https://idp.example.com/introspect",
		JwksURI:               "https://idp.example.com/jwks",
	}
}

func TestRegisterProviderWithMetadata(t *testing.T) {
	tests := map[string]struct {
		metadata *AuthorizationServerMetadata
		wantErr  bool
	}{
		"valid metadata registers successfully": {
			metadata: validMetadata(),
			wantErr:  false,
		},
		"nil metadata returns validation error": {
			metadata: nil,
			wantErr:  true,
		},
		"missing issuer returns validation error": {
			metadata: &AuthorizationServerMetadata{AuthorizationEndpoint: "a", TokenEndpoint: "b", IntrospectionEndpoint: "c", JwksURI: "d"},
			wantErr:  true,
		},
		"missing authorizationEndpoint returns validation error": {
			metadata: &AuthorizationServerMetadata{Issuer: "a", TokenEndpoint: "b", IntrospectionEndpoint: "c", JwksURI: "d"},
			wantErr:  true,
		},
		"missing tokenEndpoint returns validation error": {
			metadata: &AuthorizationServerMetadata{Issuer: "a", AuthorizationEndpoint: "b", IntrospectionEndpoint: "c", JwksURI: "d"},
			wantErr:  true,
		},
		"missing introspectionEndpoint returns validation error": {
			metadata: &AuthorizationServerMetadata{Issuer: "a", AuthorizationEndpoint: "b", TokenEndpoint: "c", JwksURI: "d"},
			wantErr:  true,
		},
		"missing jwksUri returns validation error": {
			metadata: &AuthorizationServerMetadata{Issuer: "a", AuthorizationEndpoint: "b", TokenEndpoint: "c", IntrospectionEndpoint: "d"},
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			idpServer := NewMockIDPServer()
			defer idpServer.Close()

			reg := NewIdpRegistry()
			idpCfg := &config.IDPConfiguration{
				Name:        "test-idp",
				MetadataURL: idpServer.GetMetadataURL(),
				AuthConfig:  &config.IDPAuthConfiguration{Type: "client", ClientID: "id", ClientSecret: "secret"},
				GrantType:   GrantTypeClientCredentials,
				AuthMethod:  config.ClientSecretBasic,
			}

			err := reg.RegisterProviderWithMetadata(context.Background(), idpCfg, tc.metadata, config.NewTLSConfig(), "", 30*time.Second)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			p, err := reg.GetProviderByName(context.Background(), "test-idp")
			assert.NoError(t, err)
			assert.NotNil(t, p)
			assert.Equal(t, tc.metadata.Issuer, p.GetMetadata().Issuer)
		})
	}
}

func TestIdPRegistryIDPResourceName(t *testing.T) {
	const (
		metadataURL  = "https://idp.example.com/.well-known/openid-configuration"
		resourceName = "my-idp-resource"
	)

	tests := map[string]struct {
		lookupURL     string
		preSet        bool
		expectedName  string
		expectedFound bool
	}{
		"not found before set": {
			lookupURL:     metadataURL,
			preSet:        false,
			expectedName:  "",
			expectedFound: false,
		},
		"found after set": {
			lookupURL:     metadataURL,
			preSet:        true,
			expectedName:  resourceName,
			expectedFound: true,
		},
		"different URL not found": {
			lookupURL:     "https://other.example.com/",
			preSet:        true,
			expectedName:  "",
			expectedFound: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			provReg := NewProviderRegistry()
			reg := NewIdpRegistry(WithProviderRegistry(provReg))
			if tc.preSet {
				provReg.SetIDPResourceName(metadataURL, resourceName)
			}

			got, ok := reg.GetIDPResourceName(tc.lookupURL)
			assert.Equal(t, tc.expectedFound, ok)
			assert.Equal(t, tc.expectedName, got)
		})
	}
}
