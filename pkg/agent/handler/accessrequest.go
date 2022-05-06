package handler

import (
	"context"
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
	provision   = "provision"
	deprovision = "deprovision"
	arFinalizer = "agent.accessrequest.provisioned"
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
func (h *accessRequestHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := getActionFromContext(ctx)
	if resource.Kind != mv1.AccessRequestGVK().Kind || h.prov == nil || isNotStatusSubResourceUpdate(action, meta) {
		return nil
	}

	log := getLoggerFromContext(ctx).WithComponent("accessRequestHandler")
	ctx = setLoggerInContext(ctx, log)

	ar := &mv1.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		log.WithError(err).Error("could not handle access request")
		return nil
	}

	if ok := isStatusFound(ar.Status); !ok {
		log.Debug("could not handle access request as it did not have a status subresource")
		return nil
	}

	if ok := shouldProcessPending(ar.Status.Level, ar.Metadata.State); ok {
		log.Trace("processing resource in pending status")
		ar := h.onPending(ctx, ar)
		err := h.client.CreateSubResourceScoped(ar.ResourceMeta, ar.SubResources)
		if err != nil {
			log.WithError(err).Errorf("error creating subresources")
			return err
		}
		err = h.client.CreateSubResourceScoped(ar.ResourceMeta, map[string]interface{}{"status": ar.Status})
		if err != nil {
			log.WithError(err).Errorf("error creating status subresources")
			return err
		}
	}

	if ok := shouldProcessDeleting(ar.Status.Level, ar.Metadata.State, len(ar.Finalizers)); ok {
		log.Trace("processing resource in deleting state")
		h.onDeleting(ctx, ar)
	}

	log.Trace("finished processing request")
	return nil
}

func (h *accessRequestHandler) onPending(ctx context.Context, ar *mv1.AccessRequest) *mv1.AccessRequest {
	log := getLoggerFromContext(ctx)
	app, err := h.getManagedApp(ctx, ar)
	if err != nil {
		log.WithError(err).Error("error getting managed app")
		h.onError(ctx, ar, err)
		return ar
	}

	// check the application status
	if app.Status.Level != prov.Success.String() {
		err = fmt.Errorf("error can't handle access request when application is not yet successful")
		h.onError(ctx, ar, err)
		return ar
	}

	req, err := h.newReq(ctx, ar, util.GetAgentDetails(app))
	if err != nil {
		log.WithError(err).Error("error getting resource details")
		h.onError(ctx, ar, err)
		return ar
	}

	status := h.prov.AccessRequestProvision(req)
	ar.Status = prov.NewStatusReason(status)

	details := util.MergeMapStringString(util.GetAgentDetailStrings(ar), status.GetProperties())
	util.SetAgentDetails(ar, util.MapStringStringToMapStringInterface(details))

	ri, _ := ar.AsInstance()
	if ar.Status.Level == prov.Success.String() {
		// only add finalizer on success
		h.client.UpdateResourceFinalizer(ri, arFinalizer, "", true)
	}

	ar.SubResources = map[string]interface{}{
		defs.XAgentDetails: util.GetAgentDetails(ar),
	}

	return ar
}

// onError updates the AccessRequest with an error status
func (h *accessRequestHandler) onError(_ context.Context, ar *mv1.AccessRequest, err error) {
	ps := prov.NewRequestStatusBuilder()
	status := ps.SetMessage(err.Error()).Failed()
	ar.Status = prov.NewStatusReason(status)
	ar.SubResources = map[string]interface{}{
		"status": ar.Status,
	}
}

// onDeleting deprovisions an access request and removes the finalizer
func (h *accessRequestHandler) onDeleting(ctx context.Context, ar *mv1.AccessRequest) {
	log := getLoggerFromContext(ctx)
	req, err := h.newReq(ctx, ar, map[string]interface{}{})
	if err != nil {
		log.WithError(err).Error("error getting deprovision request details: %s")
		h.onError(ctx, ar, err)
		h.client.CreateSubResourceScoped(ar.ResourceMeta, ar.SubResources)
		return
	}

	status := h.prov.AccessRequestDeprovision(req)

	ri, _ := ar.AsInstance()
	if status.GetStatus() == prov.Success {
		h.client.UpdateResourceFinalizer(ri, arFinalizer, "", false)
	} else {
		log.Debugf("request status was not Success, skipping")
		h.onError(ctx, ar, fmt.Errorf(status.GetMessage()))
		h.client.CreateSubResourceScoped(ar.ResourceMeta, ar.SubResources)
	}
}

func (h *accessRequestHandler) getManagedApp(_ context.Context, ar *mv1.AccessRequest) (*mv1.ManagedApplication, error) {
	app := mv1.NewManagedApplication(ar.Spec.ManagedApplication, ar.Metadata.Scope.Name)
	ri, err := h.client.GetResource(app.GetSelfLink())
	if err != nil {
		return nil, err
	}

	app = &mv1.ManagedApplication{}
	err = app.FromInstance(ri)
	return app, err
}

func (h *accessRequestHandler) newReq(_ context.Context, ar *mv1.AccessRequest, appDetails map[string]interface{}) (*provAccReq, error) {
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

	return &provAccReq{
		appDetails:      appDetails,
		accessData:      ar.Spec.Data,
		accessDetails:   util.GetAgentDetails(ar),
		instanceDetails: util.GetAgentDetails(instance),
		managedApp:      ar.Spec.ManagedApplication,
	}, nil
}

type provAccReq struct {
	appDetails      map[string]interface{}
	accessDetails   map[string]interface{}
	accessData      map[string]interface{}
	instanceDetails map[string]interface{}
	managedApp      string
}

// GetApplicationName gets the application name the access request is linked too.
func (r provAccReq) GetApplicationName() string {
	return r.managedApp
}

// GetAccessRequestData gets the data of the access request
func (r provAccReq) GetAccessRequestData() map[string]interface{} {
	return r.accessData
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

// GetInstanceDetails returns the 'x-agent-details' sub resource of the API Service Instance
func (r provAccReq) GetInstanceDetails() map[string]interface{} {
	if r.instanceDetails == nil {
		return map[string]interface{}{}
	}

	return r.instanceDetails
}
