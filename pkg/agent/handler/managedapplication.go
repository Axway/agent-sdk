package handler

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const maFinalizer = "agent.managedapplication.provisioned"

type managedAppProvision interface {
	ApplicationRequestProvision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus)
	ApplicationRequestDeprovision(applicationRequest prov.ApplicationRequest) (status prov.RequestStatus)
}

type managedApplication struct {
	prov   managedAppProvision
	cache  agentcache.Manager
	client client
}

// NewManagedApplicationHandler creates a Handler for Access Requests
func NewManagedApplicationHandler(prov managedAppProvision, cache agentcache.Manager, client client) Handler {
	return &managedApplication{
		prov:   prov,
		cache:  cache,
		client: client,
	}
}

// Handle processes grpc events triggered for ManagedApplications
func (h *managedApplication) Handle(action proto.Event_Type, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.ManagedApplicationGVK().Kind || h.prov == nil || isNotStatusSubResourceUpdate(action, meta) {
		return nil
	}

	log.Debugf("%s event for ManagedApplication", action.String())

	app := &mv1.ManagedApplication{}
	err := app.FromInstance(resource)
	if err != nil {
		return err
	}

	if ok := isStatusFound(app.Status); !ok {
		return nil
	}

	ma := provManagedApp{
		managedAppName: app.Name,
		teamName:       h.getTeamName(app.Owner),
		data:           util.GetAgentDetails(app),
	}

	if ok := shouldProcessPending(app.Status.Level, app.Metadata.State); ok {
		return h.onPending(app, ma)
	}

	if ok := shouldProcessDeleting(app.Status.Level, app.Metadata.State, len(app.Finalizers)); ok {
		h.onDeleting(app, ma)
	}

	return nil
}

func (h *managedApplication) onPending(app *mv1.ManagedApplication, pma provManagedApp) error {
	status := h.prov.ApplicationRequestProvision(pma)

	app.Status = prov.NewStatusReason(status)

	details := util.MergeMapStringString(util.GetAgentDetailStrings(app), status.GetProperties())
	util.SetAgentDetails(app, util.MapStringStringToMapStringInterface(details))

	// add finalizer
	ri, _ := app.AsInstance()
	h.client.UpdateResourceFinalizer(ri, maFinalizer, "", true)

	return h.client.CreateSubResourceScoped(
		app.ResourceMeta,
		map[string]interface{}{
			defs.XAgentDetails: util.GetAgentDetails(app),
			"status":           app.Status,
		},
	)
}

func (h *managedApplication) onDeleting(app *mv1.ManagedApplication, pma provManagedApp) {
	status := h.prov.ApplicationRequestDeprovision(pma)

	if status.GetStatus() == prov.Success {
		ri, _ := app.AsInstance()
		h.client.UpdateResourceFinalizer(ri, maFinalizer, "", false)
	}

}

func (h *managedApplication) getTeamName(owner *v1.Owner) string {
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
