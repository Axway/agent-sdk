package oauth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func createIDPConfig(name, metadataURL string) *config.IDPConfiguration {
	return &config.IDPConfiguration{
		Name:        name,
		MetadataURL: metadataURL,
	}
}

func TestProviderRegistry(t *testing.T) {
	idpServer := NewMockIDPServer()
	defer idpServer.Close()
	providerReg := NewProviderRegistry()
	idpRegistry := NewIdpRegistry(WithProviderRegistry(providerReg))
	idCfg := createIDPConfig("test", idpServer.GetMetadataURL())
	idpServer.SetMetadataResponseCode(http.StatusBadRequest)
	err := idpRegistry.RegisterProvider(context.Background(), idCfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.NotNil(t, err)

	idCfg = createIDPConfig("test", idpServer.GetMetadataURL())
	err = idpRegistry.RegisterProvider(context.Background(), idCfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.Nil(t, err)

	p, err := idpRegistry.GetProviderByName(context.Background(), "test")
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = idpRegistry.GetProviderByName(context.Background(), "test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = idpRegistry.GetProviderByIssuer(context.Background(), idpServer.GetIssuer())
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = idpRegistry.GetProviderByIssuer(context.Background(), "invalid-issuer")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = idpRegistry.GetProviderByTokenEndpoint(context.Background(), idpServer.GetTokenURL())
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = idpRegistry.GetProviderByTokenEndpoint(context.Background(), "invalid-token-url")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = idpRegistry.GetProviderByAuthorizationEndpoint(context.Background(), idpServer.GetAuthEndpoint())
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = idpRegistry.GetProviderByAuthorizationEndpoint(context.Background(), "invalid-auth-url")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = idpRegistry.GetProviderByMetadataURL(context.Background(), idpServer.GetMetadataURL())
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = idpRegistry.GetProviderByMetadataURL(context.Background(), "invalid-auth-url")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	pr, _ := providerReg.(*providerRegistry)
	pr.providerMap.Set("test1", "")
	pr.providerMap.SetSecondaryKey("test1", "issuer:test1")
	pr.providerMap.SetSecondaryKey("test1", "tokenEp:test1")
	pr.providerMap.SetSecondaryKey("test1", "authEp:test1")

	p, err = idpRegistry.GetProviderByName(context.Background(), "test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = idpRegistry.GetProviderByIssuer(context.Background(), "test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = idpRegistry.GetProviderByTokenEndpoint(context.Background(), "test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = idpRegistry.GetProviderByAuthorizationEndpoint(context.Background(), "test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)
}

func TestIDPResourceName(t *testing.T) {
	const (
		metadataURL  = "https://idp.example.com/.well-known/openid-configuration"
		resourceName = "my-idp-resource"
	)

	tests := []struct {
		name          string
		lookupURL     string
		preSet        bool
		expectedName  string
		expectedFound bool
	}{
		{
			name:          "not found before set",
			lookupURL:     metadataURL,
			preSet:        false,
			expectedName:  "",
			expectedFound: false,
		},
		{
			name:          "found after set",
			lookupURL:     metadataURL,
			preSet:        true,
			expectedName:  resourceName,
			expectedFound: true,
		},
		{
			name:          "different URL not found",
			lookupURL:     "https://other.example.com/",
			preSet:        true,
			expectedName:  "",
			expectedFound: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			providerReg := NewProviderRegistry()
			if tc.preSet {
				providerReg.SetIDPResourceName(metadataURL, resourceName)
			}

			name, ok := providerReg.GetIDPResourceName(tc.lookupURL)
			assert.Equal(t, tc.expectedFound, ok)
			assert.Equal(t, tc.expectedName, name)
		})
	}
}
