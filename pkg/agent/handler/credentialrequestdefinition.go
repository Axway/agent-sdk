package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type crdHandler struct {
	agentCacheManager agentcache.Manager
}

// NewCRDHandler creates a Handler for Credential Request Definitions
func NewCRDHandler(agentCacheManager agentcache.Manager) Handler {
	return &crdHandler{
		agentCacheManager: agentCacheManager,
	}
}

// Handle processes grpc events triggered for Credentials
func (h *crdHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.CredentialRequestDefinitionGVK().Kind {
		return nil
	}

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddCredentialRequestDefinition(resource)
		return nil
	}

	return h.agentCacheManager.DeleteCredentialRequestDefinition(resource.Metadata.ID)
}
