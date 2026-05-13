package agent

import (
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// NewIdentityProviderMetadataFromServerMetadata creates an *IdentityProviderMetadata populated from fetched IdP server metadata.
func NewIdentityProviderMetadataFromServerMetadata(name, scopeName string, m *oauth.AuthorizationServerMetadata) *management.IdentityProviderMetadata {
	res := management.NewIdentityProviderMetadata(name, scopeName)
	if m != nil {
		res.Spec = management.IdentityProviderMetadataSpec{
			Issuer:                m.Issuer,
			AuthorizationEndpoint: m.AuthorizationEndpoint,
			TokenEndpoint:         m.TokenEndpoint,
			IntrospectionEndpoint: m.IntrospectionEndpoint,
			JwksUri:               m.JwksURI,
		}
	}
	return res
}

// IDPResourceSupplier - optional interface for agent implementations to provide custom IdP resources
type IDPResourceSupplier interface {
	// GetIdentityProvider - returns a custom IdentityProvider resource for the given IdP config
	GetIdentityProvider(cfg config.IDPConfig) (*management.IdentityProvider, error)
	// GetIdentityProviderMetadata - returns a custom IdentityProviderMetadata resource for the given IdP config and fetched metadata
	GetIdentityProviderMetadata(cfg config.IDPConfig, metadata *oauth.AuthorizationServerMetadata) (*management.IdentityProviderMetadata, error)
}

// SetIDPResourceSupplier - registers a custom supplier for IdP and IdP Metadata resources
func SetIDPResourceSupplier(s IDPResourceSupplier) {
	agent.idpResourceSupplier = s
}

// GetIDPResourceSupplier - returns the registered custom supplier for IdP and IdP Metadata resources
func GetIDPResourceSupplier() IDPResourceSupplier {
	return agent.idpResourceSupplier
}

// ManageIDPResourceWithMetadata creates or reuses an IdentityProvider resource in Engage using
// agent-supplied metadata, bypassing any outbound HTTP fetch to a discovery URL.
// Agents that resolve OAuth metadata independently (e.g. from an internal registry) call this
// instead of relying on the config-driven startup path.
// Returns the Engage IdentityProvider resource name, or empty string on failure.
func ManageIDPResourceWithMetadata(idp config.IDPConfig, metadata *oauth.AuthorizationServerMetadata) string {
	idpLogger := log.NewFieldLogger().
		WithComponent("idplifecycle").
		WithPackage("sdk.agent").
		WithField("name", idp.GetIDPName()).
		WithField("type", idp.GetIDPType())
	return manageIDPResourceWithMetadata(idpLogger, idp, metadata)
}
