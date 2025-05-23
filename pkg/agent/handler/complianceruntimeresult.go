package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type crrHandler struct {
	agentCacheManager agentcache.Manager
}

// NewCRRHandler creates a Handler for Compliance Runtime Results
func NewCRRHandler(agentCacheManager agentcache.Manager) Handler {
	return &crrHandler{
		agentCacheManager: agentCacheManager,
	}
}

// Handle processes grpc events triggered for Compliance Runtime Results
func (h *crrHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.ComplianceRuntimeResultGVK().Kind {
		return nil
	}

	if action != proto.Event_DELETED {
		h.agentCacheManager.AddComplianceRuntimeResult(resource)
		return nil
	}

	return h.agentCacheManager.DeleteComplianceRuntimeResult(resource.Metadata.ID)
}
