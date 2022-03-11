package handler

import (
	"encoding/json"
	"fmt"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
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

	app := &mv1.ManagedApplication{}
	err := app.FromInstance(resource)
	if err != nil {
		return err
	}

	ok := isStatusFound(app.Status)
	if !ok {
		return nil
	}

	if app.Status.Level != statusPending && !(app.Status.Level == statusSuccess && app.Metadata.State == v1.ResourceDeleting) {
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

		// add finalizer
		h.updateResourceFinalizer(app, true)

		err = h.client.CreateSubResourceScoped(
			mv1.EnvironmentResourceName,
			app.Metadata.Scope.Name,
			app.PluralName(),
			app.Name,
			app.Group,
			app.APIVersion,
			map[string]interface{}{
				defs.XAgentDetails: util.GetAgentDetails(app),
				"status":           app.Status,
			},
		)
	}

	// check for deleting state on success status
	if app.Status.Level == statusSuccess && app.Metadata.State == v1.ResourceDeleting {
		status = h.prov.ApplicationRequestDeprovision(ma)

		if status.GetStatus() == provisioning.Success {
			h.updateResourceFinalizer(app, false)
		}
	}

	return err
}

func (h *managedApplication) updateResourceFinalizer(ma *mv1.ManagedApplication, add bool) error {
	const finalizer = "agent.managedapplication.provisioned"

	url := fmt.Sprintf(
		"/management/v1alpha1/environments/%s/managedapplications/%s",
		ma.Metadata.Scope.Name,
		ma.Name,
	)

	if add {
		ma.Finalizers = append(ma.Finalizers, v1.Finalizer{Name: finalizer})
	} else {
		ma.Finalizers = make([]v1.Finalizer, 0)
		for _, f := range ma.Finalizers {
			if f.Name != finalizer {
				ma.Finalizers = append(ma.Finalizers, f)
			}
		}
	}
	bts, err := json.Marshal(ma)
	if err != nil {
		return err
	}

	_, err = h.client.UpdateResource(url, bts)
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
