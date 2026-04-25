package agent

import (
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
)

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
