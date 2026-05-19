package cache

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func createIdentityProviderRI(id, name string) *v1.ResourceInstance {
	idp := management.NewIdentityProvider(name)
	idp.Metadata.ID = id
	ri, _ := idp.AsInstance()
	return ri
}

func createIdentityProviderMetadataRI(id, name, scopeName, tokenEndpoint string) *v1.ResourceInstance {
	idpMeta := management.NewIdentityProviderMetadata(name, scopeName)
	idpMeta.Metadata.ID = id
	idpMeta.Spec.TokenEndpoint = tokenEndpoint
	ri, _ := idpMeta.AsInstance()
	return ri
}

func TestAddIdentityProvider(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	// nil resource should not panic
	m.AddIdentityProvider(nil)

	ri := createIdentityProviderRI("idp-1", "my-idp")
	m.AddIdentityProvider(ri)

	// get by ID
	cached := m.GetIdentityProviderByID("idp-1")
	assert.NotNil(t, cached)
	assert.Equal(t, ri, cached)

	// get by name
	cached = m.GetIdentityProviderByName("my-idp")
	assert.NotNil(t, cached)
	assert.Equal(t, ri, cached)

	// get by invalid ID
	cached = m.GetIdentityProviderByID("non-existent")
	assert.Nil(t, cached)

	// get by invalid name
	cached = m.GetIdentityProviderByName("non-existent")
	assert.Nil(t, cached)
}

func TestAddMultipleIdentityProviders(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	ri1 := createIdentityProviderRI("idp-1", "idp-one")
	ri2 := createIdentityProviderRI("idp-2", "idp-two")

	m.AddIdentityProvider(ri1)
	m.AddIdentityProvider(ri2)

	cached := m.GetIdentityProviderByID("idp-1")
	assert.Equal(t, ri1, cached)

	cached = m.GetIdentityProviderByID("idp-2")
	assert.Equal(t, ri2, cached)

	cached = m.GetIdentityProviderByName("idp-one")
	assert.Equal(t, ri1, cached)

	cached = m.GetIdentityProviderByName("idp-two")
	assert.Equal(t, ri2, cached)
}

func TestDeleteIdentityProvider(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	ri := createIdentityProviderRI("idp-1", "my-idp")
	m.AddIdentityProvider(ri)

	cached := m.GetIdentityProviderByID("idp-1")
	assert.NotNil(t, cached)

	err := m.DeleteIdentityProvider("idp-1")
	assert.Nil(t, err)

	cached = m.GetIdentityProviderByID("idp-1")
	assert.Nil(t, cached)

	cached = m.GetIdentityProviderByName("my-idp")
	assert.Nil(t, cached)

	// delete non-existent should return error
	err = m.DeleteIdentityProvider("idp-1")
	assert.NotNil(t, err)
}

func TestAddIdentityProviderNilResource(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	// nil resource should not panic or store anything
	m.AddIdentityProvider(nil)

	cached := m.GetIdentityProviderByID("idp-1")
	assert.Nil(t, cached)
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
