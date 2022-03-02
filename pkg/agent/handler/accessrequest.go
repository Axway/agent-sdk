package handler

import (
	"encoding/json"
	"fmt"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const accessRequest = "AccessRequest"

type accessRequestProvision interface {
	AccessRequestProvision(accessRequest prov.AccessRequest) (status prov.RequestStatus)
	AccessRequestDeprovision(accessRequest prov.AccessRequest) (status prov.RequestStatus)
}

type client interface {
	GetResource(url string) (*v1.ResourceInstance, error)
	CreateResource(url string, bts []byte) (*v1.ResourceInstance, error)
	UpdateResource(url string, bts []byte) (*v1.ResourceInstance, error)
	CreateSubResourceScoped(scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{}) error
}

type accessRequestHandler struct {
	prov   accessRequestProvision
	cache  agentcache.Manager
	client client
}

// NewAccessRequestHandler creates a Handler for Access Requests
func NewAccessRequestHandler() Handler {
	return &accessRequestHandler{}
}

func (h *accessRequestHandler) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.AccessRequestGVK().Kind {
		return nil
	}

	ar := &mv1.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		return err
	}

	app, err := h.getManagedApp(ar)
	if err != nil {
		return err
	}

	req, err := h.newReq(ar, util.GetAgentDetails(app))
	if err != nil {
		return err
	}

	var status prov.RequestStatus

	// provision when status == pending, and state == provision
	if ar.State.Name == "provision" {
		status = h.prov.AccessRequestProvision(req)
	}

	// deprovision when status == pending, and state == deprovision

	if ar.State.Name == "deprovision" || action == proto.Event_DELETED {
		status = h.prov.AccessRequestDeprovision(req)
	}

	if action == proto.Event_DELETED {
		return nil
	}

	// TODO: update access request status.
	//  ar.Status = prov.NewStatusReason(status)

	// TODO: merge AccessRequest 'x-agent-details' with status.properties
	// 	details = util.MergeMapStringInterface(util.GetAgentDetails(ar), status.properties)
	// 	util.SetAgentDetails(ar, details)

	fmt.Println("Status: %+v", status)

	// TODO: update the AccessRequest with changes to x-agent-details and Status
	bts, err := json.Marshal(ar)
	if err != nil {
		return err
	}

	_, err = h.client.UpdateResource(ar.Metadata.SelfLink, bts)

	err = h.client.CreateSubResourceScoped(
		mv1.EnvironmentResourceName,
		ar.Metadata.Scope.Name,
		ar.PluralName(),
		ar.Name,
		ar.Group,
		ar.APIVersion,
		map[string]interface{}{
			defs.XAgentDetails: util.GetAgentDetails(ar),
		},
	)

	return err
}

func (h *accessRequestHandler) getManagedApp(ar *mv1.AccessRequest) (*v1.ResourceInstance, error) {
	url := fmt.Sprintf(
		"/management/v1alpha1/environments/%s/managedapplications/%s",
		ar.Metadata.Scope.Name,
		ar.Spec.ManagedApplication,
	)
	app, err := h.client.GetResource(url)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (h *accessRequestHandler) newReq(ar *mv1.AccessRequest, appDetails map[string]interface{}) (*req, error) {
	instID := ""
	managedAppName := "" // ar.Spec.ManagedApplication
	for _, ref := range ar.Metadata.References {
		if ref.Name == ar.Spec.ApiServiceInstance {
			instID = ref.ID
			break
		}
	}

	instance, err := h.cache.GetAPIServiceInstanceByID(instID)
	if err != nil {
		return nil, err
	}

	apiID, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)

	return &req{
		apiID:         apiID,
		accessDetails: util.GetAgentDetails(ar),
		appDetails:    appDetails,
		managedApp:    managedAppName,
	}, nil
}

type req struct {
	apiID         string
	appDetails    map[string]interface{}
	accessDetails map[string]interface{}
	managedApp    string
}

// GetApplicationName gets the application name the access request is linked too.
func (r req) GetApplicationName() string {
	return r.managedApp
}

// GetAPIID gets the api service instance id that the access request is linked too.
func (r req) GetAPIID() string {
	return r.apiID
}

// GetApplicationDetails returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (r req) GetApplicationDetails(key string) interface{} {
	if r.appDetails == nil {
		return nil
	}
	return r.appDetails[key]
}

// GetAccessRequestDetails returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
func (r req) GetAccessRequestDetails(key string) interface{} {
	if r.appDetails == nil {
		return nil
	}
	return r.accessDetails[key]
}
