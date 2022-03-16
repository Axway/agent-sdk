package handler

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type crdHandler struct {
	cache agentcache.Manager
}

// NewCRDHandler creates a Handler for Credential Request Definitions
func NewCRDHandler(cache agentcache.Manager) Handler {
	return &crdHandler{
		cache: cache,
	}
}

// Handle processes grpc events triggered for Credentials
func (h *crdHandler) Handle(action proto.Event_Type, meta *proto.EventMeta, resource *v1.ResourceInstance) error {

	return nil
}
