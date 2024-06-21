package oauth

import (
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
	idCfg := createIDPConfig("test", idpServer.GetMetadataURL())
	idpServer.SetMetadataResponseCode(http.StatusBadRequest)
	err := providerReg.RegisterProvider(idCfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.NotNil(t, err)

	idCfg = createIDPConfig("test", idpServer.GetMetadataURL())
	err = providerReg.RegisterProvider(idCfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.Nil(t, err)

	p, err := providerReg.GetProviderByName("test")
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = providerReg.GetProviderByName("test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = providerReg.GetProviderByIssuer(idpServer.GetIssuer())
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = providerReg.GetProviderByIssuer("invalid-issuer")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = providerReg.GetProviderByTokenEndpoint(idpServer.GetTokenURL())
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = providerReg.GetProviderByTokenEndpoint("invalid-token-url")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = providerReg.GetProviderByAuthorizationEndpoint(idpServer.GetAuthEndpoint())
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = providerReg.GetProviderByAuthorizationEndpoint("invalid-auth-url")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = providerReg.GetProviderByMetadataURL(idpServer.GetMetadataURL())
	assert.Nil(t, err)
	assert.NotNil(t, p)

	p, err = providerReg.GetProviderByMetadataURL("invalid-auth-url")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	pr, _ := providerReg.(*providerRegistry)
	pr.providerMap.Set("test1", "")
	pr.providerMap.SetSecondaryKey("test1", "issuer:test1")
	pr.providerMap.SetSecondaryKey("test1", "tokenEp:test1")
	pr.providerMap.SetSecondaryKey("test1", "authEp:test1")

	p, err = providerReg.GetProviderByName("test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = providerReg.GetProviderByIssuer("test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = providerReg.GetProviderByTokenEndpoint("test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)

	p, err = providerReg.GetProviderByAuthorizationEndpoint("test1")
	assert.NotNil(t, err)
	assert.Nil(t, p)
}
