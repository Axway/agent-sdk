package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/provisioningwebhook"
	"github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	maFinalizer = "agent.managedapplication.provisioned"
)

type teamFetcher interface {
	GetTeam(query map[string]string) ([]defs.PlatformTeam, error)
}

type managedApplication struct {
	idpRegistry oauth.IdPRegistry
	marketplaceHandler
	prov          prov.ApplicationProvisioner
	cache         agentcache.Manager
	client        client
	teamClient    teamFetcher
	retryCount    int
	webhookCfg    config.ProvisioningWebhookEndpointConfig
	webhookClient api.Client
}

func WithManagedAppRetryCount(rc int) func(c *managedApplication) {
	return func(c *managedApplication) {
		c.retryCount = rc
	}
}

func WithManagedAppIDPRegistry(registry oauth.IdPRegistry) func(c *managedApplication) {
	return func(c *managedApplication) {
		c.idpRegistry = registry
	}
}

// WithManagedAppProvisioningWebhook configures the webhook the handler calls instead of its own
// registered Provisioning implementation, when cfg.IsConfigured()
func WithManagedAppProvisioningWebhook(cfg config.ProvisioningWebhookEndpointConfig, client api.Client) func(c *managedApplication) {
	return func(c *managedApplication) {
		c.webhookCfg = cfg
		c.webhookClient = client
	}
}

func NewManagedApplicationHandler(prov prov.ApplicationProvisioner, cache agentcache.Manager, client client, opts ...func(c *managedApplication)) Handler {
	ma := &managedApplication{
		prov:       prov,
		cache:      cache,
		client:     client,
		webhookCfg: config.NewProvisioningWebhookEndpointConfig("provisioningWebhook.managedApplication"),
	}
	if tc, ok := client.(teamFetcher); ok {
		ma.teamClient = tc
	}
	for _, o := range opts {
		o(ma)
	}
	return ma
}

func (h *managedApplication) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	action := GetActionFromContext(ctx)
	if action == proto.Event_SUBRESOURCEUPDATED && event.Metadata.GetSubresource() == defs.XWebhookDetails {
		return h.webhookCfg.IsConfigured()
	}
	if h.prov == nil || h.shouldIgnore(action, event.Metadata) {
		return false
	}

	return true
}

// Handle processes grpc events triggered for ManagedApplications
func (h *managedApplication) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	log := getLoggerFromContext(ctx).WithComponent("managedApplicationHandler")
	ctx = setLoggerInContext(ctx, log)

	app := &management.ManagedApplication{}
	err := app.FromInstance(resource)
	if err != nil {
		log.WithError(err).Error("could not handle application request")
		return nil
	}

	action := GetActionFromContext(ctx)
	if action == proto.Event_SUBRESOURCEUPDATED && meta.GetSubresource() == defs.XWebhookDetails {
		if webhookDispatchedFor(app, webhookOperationProvision) {
			return h.postWebhookProvisionProcess(app)
		}
		if webhookDispatchedFor(app, webhookOperationDeprovision) {
			h.postWebhookDeprovisionProcess(log, app)
		}
		return nil
	}

	if ok := isStatusFound(app.Status); !ok {
		log.Debug("could not handle application request as it did not have a status subresource")
		return nil
	}

	owner := app.Owner
	if owner == nil {
		owner = app.Marketplace.Resource.Owner
	}
	ma := provManagedApp{
		managedAppName: app.Name,
		teamName:       h.resolveTeamName(owner),
		data:           util.GetAgentDetails(app),
		consumerOrgID:  getConsumerOrgID(app),
		id:             app.Metadata.ID,
	}

	if ok := h.shouldProvisionWebhook(app.Status, app.Metadata.State, h.webhookCfg); ok {
		log.Trace("processing resource in pending status via provisioning webhook")
		return h.onWebhookProvision(log, app, ma)
	}

	if ok := h.shouldProcessPending(app.Status, app.Metadata.State); ok {
		log.Trace("processing resource in pending status")
		return h.onPending(ctx, app, ma)
	}

	if ok := h.shouldDeprovisionWebhook(app.Status, app.Metadata.State, app.Finalizers, h.webhookCfg); ok {
		log.Trace("processing resource in deleting state via deprovisioning webhook")
		h.onWebhookDeprovision(ctx, log, app, ma)
		return nil
	}

	if ok := h.shouldProcessDeleting(app.Status, app.Metadata.State, app.Finalizers); ok {
		log.Trace("processing resource in deleting state")
		h.onDeleting(ctx, app, ma)
	}

	return nil
}

// postWebhookProvisionProcess mirrors the webhook's x-webhook-details into x-agent-details, and adds the
// finalizer once status reflects success - status is the webhook's own responsibility (see
// docs/discovery/provisioning-webhook.md), so app.Status already reflects the outcome by the time this
// runs. Only called when the dispatched operation was a provision - see Handle.
func (h *managedApplication) postWebhookProvisionProcess(app *management.ManagedApplication) error {
	if app.Status.Level == prov.Success.String() && !hasFinalizer(app.Finalizers, maFinalizer) {
		// only add finalizer on success
		ri, _ := app.AsInstance()
		h.client.UpdateResourceFinalizer(ri, maFinalizer, "", true)
	}
	if mirrorWebhookDetails(app) {
		return h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
	}
	return nil
}

// postWebhookDeprovisionProcess removes the finalizer (if still present) and drops the application from
// cache once the webhook reports it successfully completed a deprovision. app.Status can't be used for
// this - it's whatever it already was when shouldDeprovisionWebhook let the event through, not a signal
// from the webhook - so success/failure is read from the reserved "status" (and, on failure, "message")
// keys the webhook writes inside x-webhook-details itself. Status itself is the webhook's own
// responsibility (see docs/discovery/provisioning-webhook.md), so on failure this only logs - it doesn't
// write anything. No mirroring here - deprovision typically has no data to report. Only called when the
// dispatched operation was a deprovision - see Handle.
func (h *managedApplication) postWebhookDeprovisionProcess(log log.FieldLogger, app *management.ManagedApplication) {
	if webhookDetailsValue(app, webhookStatusKey) != webhookStatusSuccess {
		message := webhookDetailsValue(app, webhookMessageKey)
		log.WithField("message", message).Error("provisioning webhook reported deprovision failure")
		return
	}

	if hasFinalizer(app.Finalizers, maFinalizer) {
		ri, _ := app.AsInstance()
		h.client.UpdateResourceFinalizer(ri, maFinalizer, "", false)
	}
	h.cache.DeleteManagedApplication(app.Metadata.ID)
}

func (h *managedApplication) onPending(ctx context.Context, app *management.ManagedApplication, pma provManagedApp) error {
	log := getLoggerFromContext(ctx)

	status := h.provision(pma)
	app.Status = prov.NewStatusReason(status)

	util.SetAgentDetailsKey(app, prov.AgentDetailTeamName, pma.GetTeamName())
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

// onWebhookProvision dispatches the provision request to the configured webhook instead of calling this
// agent's own registered Provisioning implementation
func (h *managedApplication) onWebhookProvision(log log.FieldLogger, app *management.ManagedApplication, pma provManagedApp) error {
	if webhookDispatchedFor(app, webhookOperationProvision) {
		return nil
	}
	if err := provisioningwebhook.Dispatch(h.webhookClient, h.webhookCfg, newWebhookApplicationRequest(webhookOperationProvision, pma)); err != nil {
		log.WithError(err).Error("provisioning webhook dispatch failed")
		h.onError(app, err)
		return h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
	}
	markWebhookDispatched(app, webhookOperationProvision)
	return h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
}

func (h *managedApplication) provision(pma provManagedApp) prov.RequestStatus {
	status := h.prov.ApplicationRequestProvision(pma)
	resourceStatus := prov.NewStatusReason(status)
	if resourceStatus.Level == prov.Success.String() {
		return status
	}

	timeout := baseRetryTimeout
	for range h.retryCount {
		if util.IsNotTest() {
			time.Sleep(timeout)
		}
		timeout = timeout * 2

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

	if err := cleanupManagedApplicationIDPClients(ctx, log, h.idpRegistry, app); err != nil {
		log.WithError(err).Error("error cleaning up managed application IDP clients")
		h.onError(app, err)
		h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
		return
	}

	status := h.prov.ApplicationRequestDeprovision(pma)
	if status.GetStatus() == prov.Success {
		ri, _ := app.AsInstance()
		h.client.UpdateResourceFinalizer(ri, maFinalizer, "", false)
		h.cache.DeleteManagedApplication(app.Metadata.ID)
	} else {
		err := errors.New(status.GetMessage())
		log.WithError(err).Error("request status was not Success, skipping")
		h.onError(app, err)
		h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
	}
}

// onWebhookDeprovision dispatches the deprovision request to the configured webhook instead of calling
// this agent's own registered Provisioning implementation
func (h *managedApplication) onWebhookDeprovision(ctx context.Context, log log.FieldLogger, app *management.ManagedApplication, pma provManagedApp) {
	if webhookDispatchedFor(app, webhookOperationDeprovision) {
		return
	}

	if err := cleanupManagedApplicationIDPClients(ctx, log, h.idpRegistry, app); err != nil {
		log.WithError(err).Error("error cleaning up managed application IDP clients")
		h.onError(app, err)
		h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
		return
	}

	if err := provisioningwebhook.Dispatch(h.webhookClient, h.webhookCfg, newWebhookApplicationRequest(webhookOperationDeprovision, pma)); err != nil {
		log.WithError(err).Error("provisioning webhook dispatch failed")
		h.onError(app, err)
		h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
		return
	}
	markWebhookDispatched(app, webhookOperationDeprovision)
	h.client.CreateSubResource(app.ResourceMeta, app.SubResources)
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

func (h *managedApplication) resolveTeamName(owner *apiv1.Owner) string {
	if name := getTeamName(h.cache, owner); name != "" {
		return name
	}
	if h.teamClient == nil || owner == nil || owner.ID == "" {
		return ""
	}
	teams, err := h.teamClient.GetTeam(map[string]string{"query": fmt.Sprintf("guid==%q", owner.ID)})
	if err != nil || len(teams) == 0 {
		return ""
	}
	h.cache.AddTeam(&teams[0])
	return teams[0].Name
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

// hasFinalizer returns true if name is already present in finalizers
func hasFinalizer(finalizers []apiv1.Finalizer, name string) bool {
	for _, f := range finalizers {
		if f.Name == name {
			return true
		}
	}
	return false
}
