package handler

import (
	"encoding/json"

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

func (h *managedApplication) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.ManagedApplicationGVK().Kind || h.prov == nil || action == proto.Event_SUBRESOURCEUPDATED {
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

	if app.Status.Level != statusPending {
		return nil
	}

	log.Infof("Received a %s event for a ManagedApplication", action.String())
	bts, _ := json.MarshalIndent(app, "", "\t")
	log.Info(string(bts))

	teamName := ""
	if app.Owner != nil && app.Owner.ID != "" {
		team := h.cache.GetTeamByID(app.Owner.ID)
		if team != nil {
			teamName = team.Name
		}
	}

	ma := managedApp{
		managedAppName: app.Name,
		teamName:       teamName,
		data:           util.GetAgentDetails(app),
	}

	if action == proto.Event_DELETED {
		log.Info("Deprovisioning the ManagedApplication")
		h.prov.ApplicationRequestDeprovision(ma)
		return nil
	}

	var status prov.RequestStatus
	if app.Status.Level == statusPending {
		log.Info("Provisioning the ManagedApplication")
		log.Infof("%+v", ma)
		status = h.prov.ApplicationRequestProvision(ma)
	}

	s := prov.NewStatusReason(status)
	app.Status = &s

	details := util.MergeMapStringInterface(util.GetAgentDetails(app), status.GetProperties())
	util.SetAgentDetails(app, details)

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

	return err
}

type managedApp struct {
	managedAppName string
	teamName       string
	data           map[string]interface{}
}

// GetManagedApplicationName returns the name of the managed application
func (a managedApp) GetManagedApplicationName() string {
	return a.managedAppName
}

// GetTeamName gets the owning team name for the managed application
func (a managedApp) GetTeamName() string {
	return a.teamName
}

// GetAgentDetailsValue returns a value found on the managed application
func (a managedApp) GetAgentDetailsValue(key string) interface{} {
	if a.data == nil {
		return nil
	}
	return a.data[key]
}
