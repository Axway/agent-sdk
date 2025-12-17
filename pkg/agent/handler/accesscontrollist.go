package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type aclHandler struct {
	agentCacheManager agentcache.Manager
}

// NewACLHandler creates a Handler for Access Requests
func NewACLHandler(agentCacheManager agentcache.Manager) Handler {
	return &aclHandler{
		agentCacheManager: agentCacheManager,
	}
}

// Handle processes grpc events triggered for AccessRequests
func (h *aclHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.AccessControlListGVK().Kind {
		return nil
	}

	if action != proto.Event_DELETED {
		h.agentCacheManager.SetAccessControlList(resource)
		return nil
	}

	return h.agentCacheManager.DeleteAccessControlList(resource.Name)
}
