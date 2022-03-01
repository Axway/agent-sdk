package handler

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const managedAppKind = "ManagedApplication"

type managedAppProvision interface {
	ApplicationRequestProvision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus)
	ApplicationRequestDeprovision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus)
}

type managedApplication struct {
	prov managedAppProvision
}

// NewManagedApplicationHandler creates a Handler for Access Requests
func NewManagedApplicationHandler() Handler {
	return &managedApplication{}
}

func (h *managedApplication) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != managedAppKind {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.prov.ApplicationRequestProvision(&managedApp{})
	}

	if action == proto.Event_DELETED {
		h.prov.ApplicationRequestDeprovision(&managedApp{})
	}

	return nil
}

type managedApp struct {
}

func (a managedApp) GetManagedApplicationName() string {
	return "app name"
}

func (a managedApp) GetApplicationName() string {
	return "app name"
}

func (a managedApp) GetProperty(key string) (string, error) {
	return "value", nil
}
