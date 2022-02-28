package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const managedAppKind = "ManagedApplication"

type managedApplication struct{}

// NewManagedApplicationHandler creates a Handler for Access Requests
func NewManagedApplicationHandler() Handler {
	return &managedApplication{}
}

func (h *managedApplication) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != managedAppKind {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		log.Info("Managed Application Created or Updated")
	}

	if action == proto.Event_DELETED {
		log.Info("Managed Application Deleted")
	}

	return nil
}
