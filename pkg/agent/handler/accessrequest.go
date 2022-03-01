package handler

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const accessRequest = "AccessRequest"

type accessRequestProvision interface {
	AccessRequestProvision(accessRequest prov.AccessRequest) (status prov.RequestStatus)
	AccessRequestDeprovision(accessRequest prov.AccessRequest) (status prov.RequestStatus)
}

type accessRequestHandler struct {
	prov  accessRequestProvision
	cache agentcache.Manager
}

// NewAccessRequestHandler creates a Handler for Access Requests
func NewAccessRequestHandler() Handler {
	return &accessRequestHandler{}
}

func (h *accessRequestHandler) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != accessRequest {
		return nil
	}

	ar := &mv1.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		return err
	}

	instID := ""
	managedAppName := ""
	for _, ref := range ar.Metadata.References {
		if ref.Name == ar.Spec.ApiServiceInstance {
			instID = ref.ID
		}
		if ref.Kind == managedAppKind {
			managedAppName = ref.Name
		}
	}

	r := &req{
		apiID:   instID,
		appName: managedAppName,
		data:    ar.Spec.Data,
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.prov.AccessRequestProvision(r)
	}

	if action == proto.Event_DELETED {
		h.prov.AccessRequestDeprovision(r)
	}

	return nil
}

type req struct {
	appName string
	apiID   string
	data    map[string]interface{}
}

func (r req) GetApplicationName() string {
	return "app name"
}

func (r req) GetAPIID() string {
	return "123"
}

func (r req) GetProperty(key string) interface{} {
	return r.data[key]
}
