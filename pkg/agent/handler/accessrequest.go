package handler

import (
	"fmt"

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

	req := newReq(ar)

	var status prov.RequestStatus

	// provision when status == pending, and state == provision
	if ar.State.Name == "provision" {
		status = h.prov.AccessRequestProvision(req)
		// TODO: update AccessRequest status
	}

	// deprovision when status == pending, and state == deprovision

	if ar.State.Name == "deprovision" {
		status = h.prov.AccessRequestDeprovision(req)
		// TODO: update AccessRequest status
	}

	fmt.Println("Status: %+v", status)

	// TODO: Delete event probably isn't necessary. Remove it from the watch topic.

	// TODO: add all StatusRequest fields to x-agent-details
	// TODO: update the AccessRequest with changes to x-agent-details

	return nil
}

func newReq(ar *mv1.AccessRequest) req {
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

	return req{
		apiID:   instID,
		appName: managedAppName,
		data:    ar.Spec.Data,
	}
}

type req struct {
	appName string
	apiID   string
	data    map[string]interface{}
}

// GetApplicationName gets the application name the access request is linked too.
func (r req) GetApplicationName() string {
	return r.appName
}

// GetAPIID gets the api service instance id that the access request is linked too.
func (r req) GetAPIID() string {
	return r.apiID
}

// GetProperty gets a property off of the access request data map.
func (r req) GetProperty(key string) interface{} {
	return r.data[key]
}
