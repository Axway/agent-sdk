package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	provision   = "provision"
	deprovision = "deprovision"
	arFinalizer = "agent.accessrequest.provisioned"
)

type customUnitHandler interface {
	HandleQuotaEnforcement(*management.AccessRequest, *management.ManagedApplication) error
}

type accessRequestHandler struct {
	marketplaceHandler
	prov              prov.AccessProvisioner
	cache             agentcache.Manager
	client            client
	encryptSchema     encryptSchemaFunc
	customUnitHandler customUnitHandler
	retryCount        int
}

func WithAccessRequestRetryCount(rc int) func(c *accessRequestHandler) {
	return func(c *accessRequestHandler) {
		c.retryCount = rc
	}
}

// NewAccessRequestHandler creates a Handler for Access Requests
func NewAccessRequestHandler(prov prov.AccessProvisioner, cache agentcache.Manager, client client, customUnitHandler customUnitHandler, opts ...func(c *accessRequestHandler)) Handler {
	arh := &accessRequestHandler{
		prov:              prov,
		cache:             cache,
		client:            client,
		encryptSchema:     encryptSchema,
		customUnitHandler: customUnitHandler,
	}
	for _, o := range opts {
		o(arh)
	}
	return arh
}

// Handle processes grpc events triggered for AccessRequests
func (h *accessRequestHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.AccessRequestGVK().Kind || h.prov == nil || h.shouldIgnoreSubResourceUpdate(action, meta) {
		return nil
	}

	log := getLoggerFromContext(ctx).WithComponent("accessRequestHandler")
	defer log.Trace("finished processing request")
	ctx = setLoggerInContext(ctx, log)

	ar := &management.AccessRequest{}
	err := ar.FromInstance(resource)
	if err != nil {
		log.WithError(err).Error("could not handle access request")
		return nil
	}

	// add or update the cache with the access request
	// migrated access request is not added to cache until processed for Pending
	if (action == proto.Event_CREATED || action == proto.Event_UPDATED) && ar.Spec.AccessRequest == "" {
		h.cache.AddAccessRequest(resource)
	}

	if ok := isStatusFound(ar.Status); !ok {
		log.Debug("could not handle access request as it did not have a status subresource")
		return nil
	}

	if h.shouldSkipAccessRequest(log, ar) {
		return nil
	}

	if ok := h.shouldProcessPending(ar.Status, ar.Metadata.State); ok {
		mar := h.getMigratingAccessRequest(ar)

		log.Trace("processing resource in pending status")
		ar := h.onPending(ctx, ar, mar)

		ri, _ := ar.AsInstance()
		defer h.cache.AddAccessRequest(ri)

		err := h.client.CreateSubResource(ar.ResourceMeta, ar.SubResources)
		if err != nil {
			log.WithError(err).Error("error creating subresources")
		}

		// update the status regardless of errors updating the other subresources
		statusErr := h.client.CreateSubResource(ar.ResourceMeta, map[string]interface{}{"status": ar.Status})
		if statusErr != nil {
			log.WithError(statusErr).Error("error creating status subresources")
			return statusErr
		}

		if mar != nil && ar.Status.Level == prov.Success.String() {
			h.client.UpdateResourceFinalizer(mar, arFinalizer, "", false)
			err := h.client.DeleteResourceInstance(mar)
			if err != nil {
				log.WithError(err).Error("failed to delete migrating access request")
			}
			h.cache.DeleteAccessRequest(ri.Metadata.ID)
		}
		return err
	}

	if ok := h.shouldProcessDeleting(ar.Status, ar.Metadata.State, ar.Finalizers); ok {
		log.Trace("processing resource in deleting state")
		h.onDeleting(ctx, ar)
	}

	return nil
}

func (h *accessRequestHandler) onPending(ctx context.Context, ar *management.AccessRequest, mar *apiv1.ResourceInstance) *management.AccessRequest {
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

	ard, err := h.getARD(ctx, ar)
	if err != nil {
		log.WithError(err).Errorf("error getting access request definition")
		h.onError(ctx, ar, err)
		return ar
	}

	req, err := h.newReq(ctx, ar, mar, util.GetAgentDetails(app))
	if err != nil {
		log.WithError(err).Error("error getting resource details")
		h.onError(ctx, ar, err)
		return ar
	}

	updateDataFromEnumMap(ar.Spec.Data, ard.Spec.Schema)

	data := map[string]interface{}{}
	status, accessData := h.provision(req)

	if status.GetStatus() == prov.Success {
		err := h.customUnitHandler.HandleQuotaEnforcement(ar, app)

		if err != nil {
			// h.onError(ctx, ar, err)
			status = prov.NewRequestStatusBuilder().
				SetMessage(err.Error()).
				SetCurrentStatusReasons(ar.Status.Reasons).
				Failed()
		}
	}

	if status.GetStatus() == prov.Success && accessData != nil {
		sec := app.Spec.Security
		d := accessData.GetData()
		if ard.Spec.Provision == nil {
			data = d // no provision schema found, return the data
		} else if d != nil {
			data, err = h.encryptSchema(
				ard.Spec.Provision.Schema,
				d,
				sec.EncryptionKey, sec.EncryptionAlgorithm, sec.EncryptionHash,
			)
		}

		if err != nil {
			status = prov.NewRequestStatusBuilder().
				SetMessage(fmt.Sprintf("error encrypting access data: %s", err.Error())).
				SetCurrentStatusReasons(ar.Status.Reasons).
				Failed()
		}
	}

	ar.Data = data
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
		"data":             ar.Data,
	}

	return ar
}

func (h *accessRequestHandler) provision(par *provAccReq) (prov.RequestStatus, prov.AccessData) {
	status, accessData := h.prov.AccessRequestProvision(par)
	if status.GetStatus() == prov.Success {
		return status, accessData
	}

	for i := range h.retryCount {
		// Exponential backoff: 15s, 30s, 45s
		if util.IsNotTest() {
			time.Sleep(time.Duration(15*(i+1)) * time.Second)
		}

		status, accessData = h.prov.AccessRequestProvision(par)
		if status.GetStatus() == prov.Success {
			return status, accessData
		}
	}
	return status, accessData
}

// onError updates the AccessRequest with an error status
func (h *accessRequestHandler) onError(_ context.Context, ar *management.AccessRequest, err error) {
	ps := prov.NewRequestStatusBuilder()
	status := ps.SetMessage(err.Error()).SetCurrentStatusReasons(ar.Status.Reasons).Failed()
	ar.Status = prov.NewStatusReason(status)
	ar.SubResources = map[string]interface{}{
		"status": ar.Status,
	}
}

// onDeleting deprovisions an access request and removes the finalizer
func (h *accessRequestHandler) onDeleting(ctx context.Context, ar *management.AccessRequest) {
	log := getLoggerFromContext(ctx)

	app, err := h.getManagedApp(ctx, ar)
	if err != nil {
		log.WithError(err).Error("error getting managed app")
		h.onError(ctx, ar, err)
		return
	}

	ri, _ := ar.AsInstance()

	req, err := h.newReq(ctx, ar, nil, util.GetAgentDetails(app))
	if err != nil {
		log.WithError(err).Debug("removing finalizers on the access request")
		h.client.UpdateResourceFinalizer(ri, arFinalizer, "", false)
		h.cache.DeleteAccessRequest(ri.Metadata.ID)
		return
	}

	status := h.prov.AccessRequestDeprovision(req)

	if status.GetStatus() == prov.Success {
		h.client.UpdateResourceFinalizer(ri, arFinalizer, "", false)
		h.cache.DeleteAccessRequest(ri.Metadata.ID)
	} else {
		err := errors.New(status.GetMessage())
		log.WithError(err).Error("request status was not Success, skipping")
		h.onError(ctx, ar, err)
		h.client.CreateSubResource(ar.ResourceMeta, ar.SubResources)
	}
}

func (h *accessRequestHandler) getManagedApp(_ context.Context, ar *management.AccessRequest) (*management.ManagedApplication, error) {
	app := management.NewManagedApplication(ar.Spec.ManagedApplication, ar.Metadata.Scope.Name)
	ri, err := h.client.GetResource(app.GetSelfLink())
	if err != nil {
		return nil, err
	}

	app = &management.ManagedApplication{}
	err = app.FromInstance(ri)
	return app, err
}

func (h *accessRequestHandler) getARD(ctx context.Context, ar *management.AccessRequest) (*management.AccessRequestDefinition, error) {
	// get the instance from the cache
	instance, err := h.getServiceInstance(ctx, ar)
	if err != nil {
		return nil, err
	}
	svcInst := management.NewAPIServiceInstance(instance.Name, instance.Metadata.Scope.Name)

	err = svcInst.FromInstance(instance)
	if err != nil {
		return nil, err
	}
	// verify that the service instance has an access request definition
	if svcInst.Spec.AccessRequestDefinition == "" {
		return nil, fmt.Errorf("failed to provision access for service instance %s. Please contact your system administrator for further assistance", svcInst.Name)
	}

	// now get the access request definition from the instance
	ard := management.NewAccessRequestDefinition(svcInst.Spec.AccessRequestDefinition, ar.Metadata.Scope.Name)

	ri, err := h.client.GetResource(ard.GetSelfLink())
	if err != nil {
		return nil, err
	}

	ard = &management.AccessRequestDefinition{}
	err = ard.FromInstance(ri)
	return ard, err
}

func (h *accessRequestHandler) getServiceInstance(_ context.Context, ar *management.AccessRequest) (*apiv1.ResourceInstance, error) {
	instRef := ar.GetReferenceByGVK(management.APIServiceInstanceGVK())
	instID := instRef.ID
	instance, err := h.cache.GetAPIServiceInstanceByID(instID)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func (h *accessRequestHandler) getMigratingAccessRequest(ar *management.AccessRequest) *apiv1.ResourceInstance {
	if ar.Spec.AccessRequest == "" {
		return nil
	}
	accessReqRef := ar.GetReferenceByNameAndGVK(ar.Spec.AccessRequest, management.AccessRequestGVK())
	if accessReqRef.ID == "" {
		return nil
	}
	return h.cache.GetAccessRequest(accessReqRef.ID)
}

func (h *accessRequestHandler) newReq(ctx context.Context, ar *management.AccessRequest, mar *apiv1.ResourceInstance, appDetails map[string]interface{}) (*provAccReq, error) {
	instance, err := h.getServiceInstance(ctx, ar)
	if err != nil {
		return nil, err
	}
	refID := ""
	var refAccessDetails map[string]interface{}
	if mar != nil {
		refID = mar.Metadata.ID
		refAccessDetails = util.GetAgentDetails(mar)
	}
	return &provAccReq{
		appDetails:       appDetails,
		requestData:      ar.Spec.Data,
		provData:         ar.Data,
		accessDetails:    util.GetAgentDetails(ar),
		refAccessDetails: refAccessDetails,
		instanceDetails:  util.GetAgentDetails(instance),
		managedApp:       ar.Spec.ManagedApplication,
		id:               ar.Metadata.ID,
		refID:            refID,
		quota:            prov.NewQuotaFromAccessRequest(ar),
	}, nil
}

func (h *accessRequestHandler) shouldSkipAccessRequest(logger log.FieldLogger, ar *management.AccessRequest) bool {
	customAR, ok := h.prov.(prov.CustomAccessRequest)
	if !ok {
		return false
	}

	existingApisi, err := h.getServiceInstance(context.Background(), ar)
	if err != nil {
		logger.WithError(err).Error("could not get service instance from cache")
		return false
	}

	arTypes := customAR.GetIgnoredAccessRequestTypes()
	apisi := management.APIServiceInstance{}
	apisi.FromInstance(existingApisi)
	for _, ardName := range arTypes {
		if ardName == apisi.Spec.AccessRequestDefinition {
			logger.WithField("accessRequestName", ar.Name).Trace("skipping handling access request provisioning")
			return true
		}
	}
	return false
}

type provAccReq struct {
	appDetails       map[string]interface{}
	accessDetails    map[string]interface{}
	refAccessDetails map[string]interface{}
	requestData      map[string]interface{}
	instanceDetails  map[string]interface{}
	provData         interface{}
	managedApp       string
	id               string
	quota            prov.Quota
	refID            string
}

// GetApplicationName gets the application name the access request is linked too.
func (r provAccReq) GetApplicationName() string {
	return r.managedApp
}

// IsTransferring returns flag indicating the AccessRequest is for migrating referenced AccessRequest
func (r provAccReq) IsTransferring() bool {
	return r.refID != ""
}

// GetID gets the if of the access request resource
func (r provAccReq) GetID() string {
	return r.id
}

// GetReferencedID returns the ID of the referenced AccessRequest resource for the request
func (r provAccReq) GetReferencedID() string {
	return r.refID
}

// GetAccessRequestData gets the data of the access request
func (r provAccReq) GetAccessRequestData() map[string]interface{} {
	return r.requestData
}

// GetAccessRequestData gets the data of the access request
func (r provAccReq) GetAccessRequestProvisioningData() interface{} {
	return r.provData
}

// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (r provAccReq) GetApplicationDetailsValue(key string) string {
	return getDetailsValue(r.appDetails, key)
}

// GetAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
func (r provAccReq) GetAccessRequestDetailsValue(key string) string {
	return getDetailsValue(r.accessDetails, key)
}

// GetReferencedAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the referenced AccessRequest.
func (r provAccReq) GetReferencedAccessRequestDetailsValue(key string) string {
	return getDetailsValue(r.refAccessDetails, key)
}

// GetInstanceDetails returns the 'x-agent-details' sub resource of the API Service Instance
func (r provAccReq) GetInstanceDetails() map[string]interface{} {
	if r.instanceDetails == nil {
		return map[string]interface{}{}
	}

	return r.instanceDetails
}

func (r provAccReq) GetQuota() prov.Quota {
	return r.quota
}

func getDetailsValue(details map[string]interface{}, key string) string {
	if details == nil {
		return ""
	}

	return util.ToString(details[key])
}
