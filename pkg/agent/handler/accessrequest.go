package handler

import (
	"context"
	"encoding/json"
	"fmt"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/customunit"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	provision   = "provision"
	deprovision = "deprovision"
	arFinalizer = "agent.accessrequest.provisioned"
)

type arProvisioner interface {
	AccessRequestProvision(accessRequest prov.AccessRequest) (status prov.RequestStatus, data prov.AccessData)
	AccessRequestDeprovision(accessRequest prov.AccessRequest) (status prov.RequestStatus)
}

type accessRequestHandler struct {
	marketplaceHandler
	prov                 arProvisioner
	cache                agentcache.Manager
	client               client
	encryptSchema        encryptSchemaFunc
	metricServicesConfig []config.MetricServiceConfiguration
}

// NewAccessRequestHandler creates a Handler for Access Requests
func NewAccessRequestHandler(prov arProvisioner, cache agentcache.Manager, client client, metricSvcCfg []config.MetricServiceConfiguration) Handler {
	return &accessRequestHandler{
		prov:                 prov,
		cache:                cache,
		client:               client,
		encryptSchema:        encryptSchema,
		metricServicesConfig: metricSvcCfg,
	}
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
	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.cache.AddAccessRequest(resource)
	}

	if ok := isStatusFound(ar.Status); !ok {
		log.Debug("could not handle access request as it did not have a status subresource")
		return nil
	}

	if ok := h.shouldProcessPending(ar.Status, ar.Metadata.State); ok {
		log.Trace("processing resource in pending status")
		ar := h.onPending(ctx, ar)

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

		return err
	}

	if ok := h.shouldProcessDeleting(ar.Status, ar.Metadata.State, ar.Finalizers); ok {
		log.Trace("processing resource in deleting state")
		h.onDeleting(ctx, ar)
	}

	return nil
}

func (h *accessRequestHandler) onPending(ctx context.Context, ar *management.AccessRequest) *management.AccessRequest {
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

	req, err := h.newReq(ctx, ar, util.GetAgentDetails(app))
	if err != nil {
		log.WithError(err).Error("error getting resource details")
		h.onError(ctx, ar, err)
		return ar
	}

	data := map[string]interface{}{}
	status, accessData := h.prov.AccessRequestProvision(req)

	if status.GetStatus() == prov.Success && len(ar.Spec.AdditionalQuotas) > 0 {
		metricServicesConfigs := h.metricServicesConfig
		// Build quota info
		quotaInfo, err := h.buildQuotaInfo(ctx, ar, app)
		if err != nil {
			log.WithError(err).Errorf("error building quota info")
			h.onError(ctx, ar, err)
			return ar
		}
		errMessage := ""
		for _, config := range metricServicesConfigs {
			if config.MetricServiceEnabled() {
				factory := customunit.NewQuotaEnforcementClientFactory(config.URL, quotaInfo)
				client, _ := factory(ctx)
				response, err := client.QuotaEnforcementInfo()
				if err != nil {
					// if error from QE and reject on fail, we return the error back to the central
					if response.Error != "" && config.RejectOnFailEnabled() {
						errMessage = errMessage + fmt.Sprintf("TODO: message: %s", err.Error())
					}
				}
			}
		}

		if errMessage != "" {
			status = prov.NewRequestStatusBuilder().
				SetMessage(errMessage).
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

	req, err := h.newReq(ctx, ar, util.GetAgentDetails(app))
	if err != nil {
		log.WithError(err).Debug("removing finalizers on the access request")
		h.client.UpdateResourceFinalizer(ri, arFinalizer, "", false)
		h.cache.DeleteAccessRequest(ri.Metadata.ID)
		return
	}

	status := h.prov.AccessRequestDeprovision(req)

	if status.GetStatus() == prov.Success || err != nil {
		h.client.UpdateResourceFinalizer(ri, arFinalizer, "", false)
		h.cache.DeleteAccessRequest(ri.Metadata.ID)
	} else {
		err := fmt.Errorf(status.GetMessage())
		log.WithError(err).Error("request status was not Success, skipping")
		h.onError(ctx, ar, fmt.Errorf(status.GetMessage()))
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

type reference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

func (h *accessRequestHandler) getQuotaInfo(ar *management.AccessRequest) (string, int) {
	index := 0
	if len(ar.Spec.AdditionalQuotas) < index+1 {
		return "", 0
	}

	q := ar.Spec.AdditionalQuotas[index]
	for _, r := range ar.References {
		d, _ := json.Marshal(r)
		ref := &reference{}
		json.Unmarshal(d, ref)
		if ref.Kind == catalog.QuotaGVK().Kind && ref.Name == q.Name {
			return ref.Unit, int(q.Limit)
		}
	}
	return "", 0
}

func (h *accessRequestHandler) buildQuotaInfo(ctx context.Context, ar *management.AccessRequest, app *management.ManagedApplication) (*customunits.QuotaInfo, error) {
	unitRef, count := h.getQuotaInfo(ar)
	if unitRef == "" {
		return nil, nil
	}

	instance, err := h.getServiceInstance(ctx, ar)
	if err != nil {
		return nil, err
	}

	// Get service instance from access request to fetch the api service
	serviceRef := instance.GetReferenceByGVK(management.APIServiceGVK())
	service := h.cache.GetAPIServiceWithName(serviceRef.Name)
	if service == nil {
		return nil, fmt.Errorf("could not find service connected to quota")
	}
	extAPIID, err := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
	if err != nil {
		return nil, err
	}

	q := &customunits.QuotaInfo{
		ApiInfo: &customunits.APIInfo{
			ServiceDetails: util.GetAgentDetailStrings(service),
			ServiceName:    service.Name,
			ServiceID:      service.Metadata.ID,
			ExternalAPIID:  extAPIID,
		},
		AppInfo: &customunits.AppInfo{
			AppDetails: util.GetAgentDetailStrings(app),
			AppName:    app.Name,
			AppID:      app.Metadata.ID,
		},
		Quota: &customunits.Quota{
			Count: int64(count),
			Unit:  unitRef,
		},
	}

	return q, nil
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

func (h *accessRequestHandler) newReq(ctx context.Context, ar *management.AccessRequest, appDetails map[string]interface{}) (*provAccReq, error) {
	instance, err := h.getServiceInstance(ctx, ar)
	if err != nil {
		return nil, err
	}

	return &provAccReq{
		appDetails:      appDetails,
		requestData:     ar.Spec.Data,
		provData:        ar.Data,
		accessDetails:   util.GetAgentDetails(ar),
		instanceDetails: util.GetAgentDetails(instance),
		managedApp:      ar.Spec.ManagedApplication,
		id:              ar.Metadata.ID,
		quota:           prov.NewQuotaFromAccessRequest(ar),
	}, nil
}

type provAccReq struct {
	appDetails      map[string]interface{}
	accessDetails   map[string]interface{}
	requestData     map[string]interface{}
	instanceDetails map[string]interface{}
	provData        interface{}
	managedApp      string
	id              string
	quota           prov.Quota
}

// GetApplicationName gets the application name the access request is linked too.
func (r provAccReq) GetApplicationName() string {
	return r.managedApp
}

// GetID gets the if of the access request resource
func (r provAccReq) GetID() string {
	return r.id
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
	if r.appDetails == nil {
		return ""
	}

	return util.ToString(r.appDetails[key])
}

// GetAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
func (r provAccReq) GetAccessRequestDetailsValue(key string) string {
	if r.accessDetails == nil {
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

func (r provAccReq) GetQuota() prov.Quota {
	return r.quota
}
