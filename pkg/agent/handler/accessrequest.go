package handler

import (
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
	provision     = "provision"
	deprovision   = "deprovision"
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

// Handle processes grpc events triggered for AccessRequests
func (h *accessRequestHandler) Handle(action proto.Event_Type, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.AccessRequestGVK().Kind || h.prov == nil || isNotStatusSubResourceUpdate(action, meta) {
		return nil
	}

	ar := &mv1.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		return err
	}

	ok := isStatusFound(ar.Status)
	if !ok {
		return nil
	}

	if ar.Status.Level != statusPending {
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

	if action == proto.Event_DELETED {
		h.prov.AccessRequestDeprovision(req)
		return nil
	}

	var status prov.RequestStatus

	if ar.Status.Level == statusPending {
		status = h.prov.AccessRequestProvision(req)

		ar.Status = prov.NewStatusReason(status)

		details := util.MergeMapStringInterface(util.GetAgentDetails(ar), status.GetProperties())
		util.SetAgentDetails(ar, details)

		//TODO add finalizer

		err = h.client.CreateSubResourceScoped(
			mv1.EnvironmentResourceName,
			ar.Metadata.Scope.Name,
			ar.PluralName(),
			ar.Name,
			ar.Group,
			ar.APIVersion,
			map[string]interface{}{
				defs.XAgentDetails: util.GetAgentDetails(ar),
				"status":           ar.Status,
			},
		)
	}

	// check for deleting state on success status
	if ar.Status.Level == statusSuccess && ar.Metadata.State == "DELETING" {
		status = h.prov.AccessRequestDeprovision(req)

		// TODO remove finalizer
		_ = status
	}

	return err
}

func (h *accessRequestHandler) getManagedApp(ar *mv1.AccessRequest) (*v1.ResourceInstance, error) {
	url := fmt.Sprintf(
		"/management/v1alpha1/environments/%s/managedapplications/%s",
		ar.Metadata.Scope.Name,
		ar.Spec.ManagedApplication,
	)
	return h.client.GetResource(url)
}

func (h *accessRequestHandler) newReq(ar *mv1.AccessRequest, appDetails map[string]interface{}) (*provAccReq, error) {
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
	stage, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIStage)

	return &provAccReq{
		apiID:         apiID,
		appDetails:    appDetails,
		stage:         stage,
		accessDetails: util.GetAgentDetails(ar),
		managedApp:    ar.Spec.ManagedApplication,
	}, nil
}

type provAccReq struct {
	apiID         string
	appDetails    map[string]interface{}
	accessDetails map[string]interface{}
	managedApp    string
	stage         string
}

// GetApplicationName gets the application name the access request is linked too.
func (r provAccReq) GetApplicationName() string {
	return r.managedApp
}

// GetAPIID gets the api service instance id that the access request is linked too.
func (r provAccReq) GetAPIID() string {
	return r.apiID
}

// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (r provAccReq) GetApplicationDetailsValue(key string) interface{} {
	if r.appDetails == nil {
		return nil
	}
	return r.appDetails[key]
}

// GetAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
func (r provAccReq) GetAccessRequestDetailsValue(key string) interface{} {
	if r.appDetails == nil {
		return nil
	}
	return r.accessDetails[key]
}

func (r provAccReq) GetStage() string {
	return r.stage
}