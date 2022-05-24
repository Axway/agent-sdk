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
	maFinalizer = "agent.managedapplication.provisioned"
	maLogPrefix = "managed application handler - %s"
)

type managedAppProvision interface {
	ApplicationRequestProvision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus)
	ApplicationRequestDeprovision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus)
}

type managedApplication struct {
	prov   managedAppProvision
	cache  agentcache.Manager
	client client
}

// NewManagedApplicationHandler creates a Handler for Credentials
func NewManagedApplicationHandler(prov managedAppProvision, cache agentcache.Manager, client client) Handler {
	return &managedApplication{
		prov:   prov,
		cache:  cache,
		client: client,
	}
}

// Handle processes grpc events triggered for ManagedApplications
func (h *managedApplication) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != mv1.ManagedApplicationGVK().Kind || h.prov == nil || isNotStatusSubResourceUpdate(action, meta) {
		return nil
	}

	log := getLoggerFromContext(ctx).WithComponent("managedApplicationHandler")
	ctx = setLoggerInContext(ctx, log)

	app := &mv1.ManagedApplication{}
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
		teamName:       h.getTeamName(ctx, app.Owner),
		data:           util.GetAgentDetails(app),
	}

	if ok := shouldProcessPending(app.Status.Level, app.Metadata.State); ok {
		log.Trace("processing resource in pending status")
		return h.onPending(ctx, app, ma)
	}

	if ok := shouldProcessDeleting(app.Status.Level, app.Metadata.State, len(app.Finalizers)); ok {
		log.Trace("processing resource in deleting state")
		h.onDeleting(ctx, app, ma)
	}

	return nil
}

func (h *managedApplication) onPending(_ context.Context, app *mv1.ManagedApplication, pma provManagedApp) error {
	status := h.prov.ApplicationRequestProvision(pma)

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
		return err
	}
	return h.client.CreateSubResource(app.ResourceMeta, map[string]interface{}{"status": app.Status})
}

func (h *managedApplication) onDeleting(ctx context.Context, app *mv1.ManagedApplication, pma provManagedApp) {
	log := getLoggerFromContext(ctx)
	status := h.prov.ApplicationRequestDeprovision(pma)

	if status.GetStatus() == prov.Success {
		ri, _ := app.AsInstance()
		h.client.UpdateResourceFinalizer(ri, maFinalizer, "", false)
	} else {
		err := fmt.Errorf(status.GetMessage())
		log.WithError(err).Error("request status was not Success, skipping")
		h.onError(app, err)
		h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
	}
}

// onError updates the managed app with an error status
func (h *managedApplication) onError(ar *mv1.ManagedApplication, err error) {
	ps := prov.NewRequestStatusBuilder()
	status := ps.SetMessage(err.Error()).Failed()
	ar.Status = prov.NewStatusReason(status)
	ar.SubResources = map[string]interface{}{
		"status": ar.Status,
	}
}

func (h *managedApplication) getTeamName(_ context.Context, owner *v1.Owner) string {
	teamName := ""
	if owner != nil && owner.ID != "" {
		team := h.cache.GetTeamByID(owner.ID)
		if team != nil {
			teamName = team.Name
		}
	}
	return teamName
}

type provManagedApp struct {
	managedAppName string
	teamName       string
	data           map[string]interface{}
}

// GetManagedApplicationName returns the name of the managed application
func (a provManagedApp) GetManagedApplicationName() string {
	return a.managedAppName
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
