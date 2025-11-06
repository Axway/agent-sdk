package handler

import (
	"context"
	"errors"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	maFinalizer = "agent.managedapplication.provisioned"
)

type managedApplication struct {
	marketplaceHandler
	prov       prov.ApplicationProvisioner
	cache      agentcache.Manager
	client     client
	retryCount int
}

func WithManagedAppRetryCount(rc int) func(c *managedApplication) {
	return func(c *managedApplication) {
		c.retryCount = rc
	}
}

// NewManagedApplicationHandler creates a Handler for Credentials
func NewManagedApplicationHandler(prov prov.ApplicationProvisioner, cache agentcache.Manager, client client, opts ...func(c *managedApplication)) Handler {
	ma := &managedApplication{
		prov:   prov,
		cache:  cache,
		client: client,
	}
	for _, o := range opts {
		o(ma)
	}
	return ma
}

// Handle processes grpc events triggered for ManagedApplications
func (h *managedApplication) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.ManagedApplicationGVK().Kind || h.prov == nil || h.shouldIgnoreSubResourceUpdate(action, meta) {
		return nil
	}

	log := getLoggerFromContext(ctx).WithComponent("managedApplicationHandler")
	ctx = setLoggerInContext(ctx, log)

	app := &management.ManagedApplication{}
	err := app.FromInstance(resource)
	if err != nil {
		log.WithError(err).Error("could not handle application request")
		return nil
	}

	if ok := isStatusFound(app.Status); !ok {
		log.Debug("could not handle application request as it did not have a status subresource")
		return nil
	}

	ma := provManagedApp{
		managedAppName: app.Name,
		teamName:       getTeamName(h.cache, app.Owner),
		data:           util.GetAgentDetails(app),
		consumerOrgID:  getConsumerOrgID(app),
		id:             app.Metadata.ID,
	}

	if ok := h.shouldProcessPending(app.Status, app.Metadata.State); ok {
		log.Trace("processing resource in pending status")
		return h.onPending(ctx, app, ma)
	}

	if ok := h.shouldProcessDeleting(app.Status, app.Metadata.State, app.Finalizers); ok {
		log.Trace("processing resource in deleting state")
		h.onDeleting(ctx, app, ma)
	}

	return nil
}

func (h *managedApplication) onPending(ctx context.Context, app *management.ManagedApplication, pma provManagedApp) error {
	log := getLoggerFromContext(ctx)
	status := h.provision(pma)
	app.Status = prov.NewStatusReason(status)

	details := util.MergeMapStringString(util.GetAgentDetailStrings(app), status.GetProperties())
	util.SetAgentDetails(app, util.MapStringStringToMapStringInterface(details))

	// add finalizer
	ri, _ := app.AsInstance()
	if app.Status.Level == prov.Success.String() {
		// only add finalizer on success
		h.client.UpdateResourceFinalizer(ri, maFinalizer, "", true)
	}

	app.SubResources = map[string]interface{}{
		defs.XAgentDetails: util.GetAgentDetails(app),
	}

	err := h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
	if err != nil {
		log.WithError(err).Error("error creating subresources")
	}

	statusErr := h.client.CreateSubResource(app.ResourceMeta, map[string]interface{}{"status": app.Status})
	if statusErr != nil {
		log.WithError(statusErr).Error("error creating status subresources")
		return statusErr
	}

	return err
}

func (h *managedApplication) provision(pma provManagedApp) prov.RequestStatus {
	status := h.prov.ApplicationRequestProvision(pma)
	resourceStatus := prov.NewStatusReason(status)
	if resourceStatus.Level == prov.Success.String() {
		return status
	}

	for i := range h.retryCount {
		// Exponential backoff: 15s, 30s, 45s
		if util.IsNotTest() {
			time.Sleep(time.Duration(15*(i+1)) * time.Second)
		}

		status = h.prov.ApplicationRequestProvision(pma)
		resourceStatus = prov.NewStatusReason(status)
		if resourceStatus.Level == prov.Success.String() {
			return status
		}
	}
	return status
}

func (h *managedApplication) onDeleting(ctx context.Context, app *management.ManagedApplication, pma provManagedApp) {
	log := getLoggerFromContext(ctx)
	status := h.prov.ApplicationRequestDeprovision(pma)

	if status.GetStatus() == prov.Success {
		ri, _ := app.AsInstance()
		h.client.UpdateResourceFinalizer(ri, maFinalizer, "", false)
	} else {
		err := errors.New(status.GetMessage())
		log.WithError(err).Error("request status was not Success, skipping")
		h.onError(app, err)
		h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
	}
}

// onError updates the managed app with an error status
func (h *managedApplication) onError(ar *management.ManagedApplication, err error) {
	ps := prov.NewRequestStatusBuilder()
	status := ps.SetMessage(err.Error()).Failed()
	ar.Status = prov.NewStatusReason(status)
	ar.SubResources = map[string]interface{}{
		"status": ar.Status,
	}
}

type provManagedApp struct {
	managedAppName string
	teamName       string
	consumerOrgID  string
	id             string
	data           map[string]interface{}
}

// GetManagedApplicationName returns the name of the managed application
func (a provManagedApp) GetManagedApplicationName() string {
	return a.managedAppName
}

// GetTeamName gets the owning team name for the managed application
func (a provManagedApp) GetID() string {
	return a.id
}

// GetTeamName gets the owning team name for the managed application
func (a provManagedApp) GetTeamName() string {
	return a.teamName
}

// GetApplicationDetailsValue returns a value found on the managed application
func (a provManagedApp) GetApplicationDetailsValue(key string) string {
	if a.data == nil {
		return ""
	}

	return util.ToString(a.data[key])
}

// GetConsumerOrgID returns the ID of the consumer org for the managed application
func (a provManagedApp) GetConsumerOrgID() string {
	return a.consumerOrgID
}

func getTeamName(cache getTeamByID, owner *apiv1.Owner) string {
	teamName := ""
	if owner != nil && owner.ID != "" {
		team := cache.GetTeamByID(owner.ID)
		if team != nil {
			teamName = team.Name
		}
	}
	return teamName
}

func getConsumerOrgID(app *management.ManagedApplication) string {
	consumerOrgID := ""
	if app != nil && app.Marketplace.Resource.Owner != nil && app.Marketplace.Resource.Owner.Organization.ID != "" {
		consumerOrgID = app.Marketplace.Resource.Owner.Organization.ID
	}
	return consumerOrgID
}
