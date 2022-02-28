package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const credentialKind = "Credential"

type credentials struct{}

// NewcredentialHandler creates a Handler for Access Requests
func NewcredentialHandler() Handler {
	return &credentials{}
}

func (h *credentials) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != credentialKind {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		log.Info("Credentials Created or Updated")
	}

	if action == proto.Event_DELETED {
		log.Info("Credentials Deleted")
	}

	return nil
}
