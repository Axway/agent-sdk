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

const (
	provision     = "Provision"
	deprovision   = "Deprovision"
	statusErr     = "Error"
	statusSuccess = "Success"
	statusPending = "Pending"
)

type arProvisioner interface {
	AccessRequestProvision(accessRequest prov.AccessRequest) (status prov.RequestStatus)
	AccessRequestDeprovision(accessRequest prov.AccessRequest) (status prov.RequestStatus)
}

type accessRequestHandler struct {
	prov   arProvisioner
	cache  agentcache.Manager
	client client
}

// NewAccessRequestHandler creates a Handler for Access Requests
func NewAccessRequestHandler(prov arProvisioner, cache agentcache.Manager, client client) Handler {
	return &accessRequestHandler{
		prov:   prov,
		cache:  cache,
		client: client,
	}
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

	if ar.Status.Level == statusErr || ar.Status.Level == statusSuccess {
		return nil
	}

	app, err := h.getManagedApp(ar)
	if err != nil {
		return err
	}

	req, err := h.newReq(ar, util.GetAgentDetails(app))
	if err != nil {
		return err
	}

	if ar.Status == nil || ar.Status.Level == "" {
		return fmt.Errorf("unable to provision AccessRequest %s. Status not found", ar.Name)
	}

	if action == proto.Event_DELETED {
		h.prov.AccessRequestDeprovision(req)
		return nil
	}

	var status prov.RequestStatus

	if ar.Status.Level == statusPending && ar.State.Name == provision {
		status = h.prov.AccessRequestProvision(req)
	}

	if ar.Status.Level == statusPending && ar.State.Name == deprovision {
		status = h.prov.AccessRequestDeprovision(req)
	}

	s := prov.NewStatusReason(status)
	ar.Status = &s

	details := util.MergeMapStringInterface(util.GetAgentDetails(ar), status.GetProperties())
	util.SetAgentDetails(ar, details)

	bts, err := json.Marshal(ar)
	if err != nil {
		return err
	}

	_, err = h.client.UpdateResource(ar.Metadata.SelfLink, bts)
	if err != nil {
		return err
	}

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

func (h *accessRequestHandler) newReq(ar *mv1.AccessRequest, appDetails map[string]interface{}) (*arReq, error) {
	instID := ""
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

	return &arReq{
		apiID:         apiID,
		accessDetails: util.GetAgentDetails(ar),
		appDetails:    appDetails,
		managedApp:    ar.Spec.ManagedApplication,
	}, nil
}

type arReq struct {
	apiID         string
	appDetails    map[string]interface{}
	accessDetails map[string]interface{}
	managedApp    string
}

// GetApplicationName gets the application name the access request is linked too.
func (r arReq) GetApplicationName() string {
	return r.managedApp
}

// GetAPIID gets the api service instance id that the access request is linked too.
func (r arReq) GetAPIID() string {
	return r.apiID
}

// GetApplicationDetails returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (r arReq) GetApplicationDetails(key string) interface{} {
	if r.appDetails == nil {
		return nil
	}
	return r.appDetails[key]
}

// GetAccessRequestDetails returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
func (r arReq) GetAccessRequestDetails(key string) interface{} {
	if r.appDetails == nil {
		return nil
	}
	return r.accessDetails[key]
}
