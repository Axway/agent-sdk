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

	log.Infof("%s event for ManagedApplication", action.String())

	app := &mv1.ManagedApplication{}
	err := app.FromInstance(resource)
	if err != nil {
		return err
	}

	ok := isStatusFound(app.Status)
	if !ok {
		return nil
	}

	if app.Status.Level != statusPending {
		return nil
	}

	ma := provManagedApp{
		managedAppName: app.Name,
		teamName:       h.getTeamName(app.Owner),
		data:           util.GetAgentDetails(app),
	}

	var status prov.RequestStatus
	if app.Status.Level == statusPending {
		status = h.prov.ApplicationRequestProvision(ma)

		app.Status = prov.NewStatusReason(status)

		details := util.MergeMapStringInterface(util.GetAgentDetails(app), status.GetProperties())
		util.SetAgentDetails(app, details)

		err = h.client.CreateSubResourceScoped(
			mv1.EnvironmentResourceName,
			app.PluralName(),
			app.ResourceMeta,
			map[string]interface{}{
				defs.XAgentDetails: util.GetAgentDetails(app),
				"status":           app.Status,
			},
		)
	}

	// check for deleting state on success status
	if app.Status.Level == statusSuccess && app.Metadata.State == v1.ResourceDeleting {
		status = h.prov.ApplicationRequestDeprovision(ma)

		// TODO remove finalizer
		_ = status
	}

	return err
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

// GetAgentDetailsValue returns a value found on the managed application
func (a provManagedApp) GetAgentDetailsValue(key string) interface{} {
	if a.data == nil {
		return nil
	}
	return a.data[key]
}
