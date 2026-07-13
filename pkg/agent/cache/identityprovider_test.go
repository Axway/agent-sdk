package cache

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func createIdentityProviderMetadataRI(id, name, scopeName, tokenEndpoint string) *v1.ResourceInstance {
	idpMeta := management.NewIdentityProviderMetadata(name, scopeName)
	idpMeta.Metadata.ID = id
	idpMeta.Spec.TokenEndpoint = tokenEndpoint
	ri, _ := idpMeta.AsInstance()
	return ri
}

func TestAddIdentityProviderMetadata(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	// nil resource should not panic
	m.AddIdentityProviderMetadata(nil)

	ri := createIdentityProviderMetadataRI("idp-meta-1", "my-idp-metadata", "my-idp", "https://example.com/token")
	m.AddIdentityProviderMetadata(ri)

	// get by token URL
	cached := m.GetIdentityProviderMetadataByTokenUrl("https://example.com/token")
	assert.NotNil(t, cached)
	assert.Equal(t, ri, cached)

	// get by invalid token URL
	cached = m.GetIdentityProviderMetadataByTokenUrl("https://invalid.com/token")
	assert.Nil(t, cached)
}

func TestAddMultipleIdentityProviderMetadata(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	ri1 := createIdentityProviderMetadataRI("meta-1", "metadata-one", "idp-one", "https://idp1.com/token")
	ri2 := createIdentityProviderMetadataRI("meta-2", "metadata-two", "idp-two", "https://idp2.com/token")

	m.AddIdentityProviderMetadata(ri1)
	m.AddIdentityProviderMetadata(ri2)

	cached := m.GetIdentityProviderMetadataByTokenUrl("https://idp1.com/token")
	assert.Equal(t, ri1, cached)

	cached = m.GetIdentityProviderMetadataByTokenUrl("https://idp2.com/token")
	assert.Equal(t, ri2, cached)
}

func TestDeleteIdentityProviderMetadata(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	ri := createIdentityProviderMetadataRI("meta-1", "my-metadata", "my-idp", "https://example.com/token")
	m.AddIdentityProviderMetadata(ri)

	cached := m.GetIdentityProviderMetadataByTokenUrl("https://example.com/token")
	assert.NotNil(t, cached)

	err := m.DeleteIdentityProviderMetadata("meta-1")
	assert.Nil(t, err)

	cached = m.GetIdentityProviderMetadataByTokenUrl("https://example.com/token")
	assert.Nil(t, cached)

	// delete non-existent should return error
	err = m.DeleteIdentityProviderMetadata("meta-1")
	assert.NotNil(t, err)
}

func TestAddIdentityProviderMetadataNilResource(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	// nil resource should not panic or store anything
	m.AddIdentityProviderMetadata(nil)

	cached := m.GetIdentityProviderMetadataByTokenUrl("https://example.com/token")
	assert.Nil(t, cached)
}
