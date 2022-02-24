package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const accessRequest = "AccessRequest"

type accessRequestHandler struct{}

// NewAccessRequestHandler creates a Handler for Access Requests
func NewAccessRequestHandler() Handler {
	return &accessRequestHandler{}
}

func (h *accessRequestHandler) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != accessRequest {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		log.Info("Access Request Created or Updated")
	}

	if action == proto.Event_DELETED {
		log.Info("Access Request Deleted")
	}

	return nil
}
