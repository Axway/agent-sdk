package handler

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type ardHandler struct {
	cache agentcache.Manager
}

// NewARDHandler creates a Handler for Access Requests
func NewARDHandler(cache agentcache.Manager) Handler {
	return &ardHandler{
		cache: cache,
	}
}

// Handle processes grpc events triggered for AccessRequests
func (h *ardHandler) Handle(action proto.Event_Type, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	return nil
}
