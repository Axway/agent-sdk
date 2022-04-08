package handler

import (
	"log"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type aclHandler struct {
	agentCacheManager agentcache.Manager
}

// NewACLHandler creates a Handler for Access Requests
func NewACLHandler(agentCacheManager agentcache.Manager) Handler {
	log.Print()
	return &aclHandler{
		agentCacheManager: agentCacheManager,
	}
}

// Handle processes grpc events triggered for AccessRequests
func (h *aclHandler) Handle(action proto.Event_Type, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.AccessControlListGVK().Kind {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.agentCacheManager.SetAccessControlList(resource)
	}

	if action == proto.Event_DELETED {
		return h.agentCacheManager.DeleteAccessControlList()
	}

	return nil
}
