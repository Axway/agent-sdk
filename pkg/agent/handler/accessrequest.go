package handler

import (
	"fmt"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	provision     = "provision"
	deprovision   = "deprovision"
	statusErr     = "Error"
	statusSuccess = "Success"
	statusPending = "Pending"
	arFinalizer   = "agent.accessrequest.provisioned"
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

	if ok := isStatusFound(ar.Status); !ok {
		return nil
	}

	if ok := shouldProcessPending(ar.Status.Level, ar.Metadata.State); ok {
		log.Tracef("access request handler - processing resource in pending status")
		return h.onPending(ar)
	}

	if ok := shouldProcessDeleting(ar.Status.Level, ar.Metadata.State, len(ar.Finalizers)); ok {
		log.Tracef("access request handler - processing resource in deleting state")
		return h.onDeleting(ar)
	}

	return nil
}

func (h *accessRequestHandler) onPending(ar *mv1.AccessRequest) error {
	app, err := h.getManagedApp(ar)
	if err != nil {
		return err
	}

	req, err := h.newReq(ar, util.GetAgentDetails(app))
	if err != nil {
		return err
	}

	status := h.prov.AccessRequestProvision(req)
	ar.Status = prov.NewStatusReason(status)

	details := util.MergeMapStringString(util.GetAgentDetailStrings(ar), status.GetProperties())
	util.SetAgentDetails(ar, util.MapStringStringToMapStringInterface(details))

	// add a finalizer
	ri, _ := ar.AsInstance()
	h.client.UpdateResourceFinalizer(ri, arFinalizer, "", true)

	return h.client.CreateSubResourceScoped(
		ar.ResourceMeta,
		map[string]interface{}{
			defs.XAgentDetails: util.GetAgentDetails(ar),
			"status":           ar.Status,
		},
	)
}

func (h *accessRequestHandler) onDeleting(ar *mv1.AccessRequest) error {
	req, err := h.newReq(ar, map[string]interface{}{})
	if err != nil {
		return err
	}
	status := h.prov.AccessRequestDeprovision(req)

	ri, _ := ar.AsInstance()
	if status.GetStatus() == prov.Success {
		h.client.UpdateResourceFinalizer(ri, arFinalizer, "", false)
	}

	return nil
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
func (r provAccReq) GetApplicationDetailsValue(key string) string {
	if r.appDetails == nil {
		return ""
	}

	return util.ToString(r.appDetails[key])
}

// GetAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
func (r provAccReq) GetAccessRequestDetailsValue(key string) string {
	if r.appDetails == nil {
		return ""
	}

	return util.ToString(r.accessDetails[key])
}

func (r provAccReq) GetStage() string {
	return r.stage
}
