package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
)

type mockIDPResourceSupplier struct {
	idpResult      *management.IdentityProvider
	idpErr         error
	metadataResult *management.IdentityProviderMetadata
	metadataErr    error
}

func (m *mockIDPResourceSupplier) GetIdentityProvider(_ config.IDPConfig) (*management.IdentityProvider, error) {
	return m.idpResult, m.idpErr
}

func (m *mockIDPResourceSupplier) GetIdentityProviderMetadata(_ config.IDPConfig, _ *oauth.AuthorizationServerMetadata) (*management.IdentityProviderMetadata, error) {
	return m.metadataResult, m.metadataErr
}

func TestSetGetIDPResourceSupplier(t *testing.T) {
	tests := map[string]struct {
		supplier IDPResourceSupplier
	}{
		"nil supplier":      {supplier: nil},
		"non-nil supplier":  {supplier: &mockIDPResourceSupplier{}},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			agent.idpResourceSupplier = nil
			SetIDPResourceSupplier(tc.supplier)
			assert.Equal(t, tc.supplier, GetIDPResourceSupplier())
		})
	}
}

func TestNewIdentityProviderMetadataFromServerMetadata(t *testing.T) {
	tests := map[string]struct {
		name      string
		scopeName string
		metadata  *oauth.AuthorizationServerMetadata
		wantSpec  management.IdentityProviderMetadataSpec
	}{
		"nil metadata returns empty spec": {
			name:      "my-idp",
			scopeName: "my-idp",
			metadata:  nil,
			wantSpec:  management.IdentityProviderMetadataSpec{},
		},
		"populated metadata maps all fields": {
			name:      "my-idp",
			scopeName: "my-idp",
			metadata: &oauth.AuthorizationServerMetadata{
				Issuer:                "https://idp.example.com",
				AuthorizationEndpoint: "https://idp.example.com/auth",
				TokenEndpoint:         "https://idp.example.com/token",
				IntrospectionEndpoint: "https://idp.example.com/introspect",
				JwksURI:               "https://idp.example.com/jwks",
			},
			wantSpec: management.IdentityProviderMetadataSpec{
				Issuer:                "https://idp.example.com",
				AuthorizationEndpoint: "https://idp.example.com/auth",
				TokenEndpoint:         "https://idp.example.com/token",
				IntrospectionEndpoint: "https://idp.example.com/introspect",
				JwksUri:               "https://idp.example.com/jwks",
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			res := NewIdentityProviderMetadataFromServerMetadata(tc.name, tc.scopeName, tc.metadata)
			assert.NotNil(t, res)
			assert.Equal(t, tc.name, res.Name)
			assert.Equal(t, tc.scopeName, res.Metadata.Scope.Name)
			assert.Equal(t, tc.wantSpec, res.Spec)
		})
	}
}
