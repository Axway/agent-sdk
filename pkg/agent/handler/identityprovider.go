package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type idpHandler struct {
	agentCacheManager agentcache.Manager
	credentialConfig  config.CredentialConfig
}

// NewIDPHandler creates a Handler for Identity Providers.
func NewIDPHandler(agentCacheManager agentcache.Manager, credentialConfig config.CredentialConfig) Handler {
	return &idpHandler{
		agentCacheManager: agentCacheManager,
		credentialConfig:  credentialConfig,
	}
}

func (c *idpHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource == nil || resource.Kind != management.IdentityProviderMetadataGVK().Kind {
		return nil
	}

	action := GetActionFromContext(ctx)
	if action == proto.Event_DELETED {
		c.removeIDPMetadata(resource)
		return nil
	}
	c.updateIDPMetadata(resource)

	return nil
}

func (c *idpHandler) updateIDPMetadata(resource *v1.ResourceInstance) {
	if resource == nil {
		return
	}
	meta := &management.IdentityProviderMetadata{}
	if err := meta.FromInstance(resource); err != nil {
		return
	}
	if meta.Spec.TokenEndpoint == "" || meta.Metadata.Scope.Name == "" {
		return
	}
	c.agentCacheManager.AddIdentityProviderMetadata(resource)
}

func (c *idpHandler) removeIDPMetadata(resource *v1.ResourceInstance) {
	if resource == nil {
		return
	}
	c.agentCacheManager.DeleteIdentityProviderMetadata(resource.Metadata.ID)
}
