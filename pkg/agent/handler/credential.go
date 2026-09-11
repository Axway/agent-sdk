package handler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/provisioningwebhook"
	"github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning/idp"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	update          = "update"
	xAxwayEncrypted = "x-axway-encrypted"
	crFinalizer     = "agent.credential.provisioned"
)

type credProv interface {
	CredentialProvision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentials prov.Credential)
	CredentialDeprovision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus)
	CredentialUpdate(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentials prov.Credential)
}

type credentials struct {
	marketplaceHandler
	prov                credProv
	client              client
	encryptSchema       encryptSchemaFunc
	idpProviderRegistry oauth.IdPRegistry
	retryCount          int
	webhookCfg          config.ProvisioningWebhookEndpointConfig
	webhookClient       api.Client
}

func WithCredentialRetryCount(rc int) func(c *credentials) {
	return func(c *credentials) {
		c.retryCount = rc
	}
}

// WithCredentialProvisioningWebhook configures the webhook the handler calls instead of its own
// registered Provisioning implementation, when cfg.IsConfigured()
func WithCredentialProvisioningWebhook(cfg config.ProvisioningWebhookEndpointConfig, client api.Client) func(c *credentials) {
	return func(c *credentials) {
		c.webhookCfg = cfg
		c.webhookClient = client
	}
}

// encryptSchemaFunc func signature for encryptSchema
type encryptSchemaFunc func(schema, credData map[string]interface{}, key, alg, hash string) (map[string]interface{}, error)

// NewCredentialHandler creates a Handler for Credentials
func NewCredentialHandler(prov credProv, client client, providerRegistry oauth.IdPRegistry, opts ...func(*credentials)) Handler {
	c := &credentials{
		prov:                prov,
		client:              client,
		encryptSchema:       encryptSchema,
		idpProviderRegistry: providerRegistry,
		webhookCfg:          config.NewProvisioningWebhookEndpointConfig("provisioningWebhook.credential"),
	}

	for _, o := range opts {
		o(c)
	}
	return c
}

func (h *credentials) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	action := GetActionFromContext(ctx)
	if action == proto.Event_SUBRESOURCEUPDATED && event.Metadata.GetSubresource() == defs.XWebhookDetails {
		return h.webhookCfg.IsConfigured()
	}
	if action == proto.Event_DELETED || h.prov == nil || h.shouldIgnore(action, event.Metadata) {
		return false
	}
	return true
}

// Handle processes grpc events triggered for Credentials
func (h *credentials) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	logger := getLoggerFromContext(ctx).WithComponent("credentialHandler")
	ctx = setLoggerInContext(ctx, logger)

	cr := &management.Credential{}
	err := cr.FromInstance(resource)
	if err != nil {
		logger.WithError(err).Error("could not handle credential request")
		return nil
	}

	action := GetActionFromContext(ctx)
	if action == proto.Event_SUBRESOURCEUPDATED && meta.GetSubresource() == defs.XWebhookDetails {
		if webhookDispatchedFor(cr, webhookOperationProvision) || webhookDispatchedFor(cr, update) {
			return h.postWebhookProvisionProcess(cr)
		}
		if webhookDispatchedFor(cr, webhookOperationDeprovision) {
			h.postWebhookDeprovisionProcess(logger, cr)
		}
		return nil
	}

	if ok := isStatusFound(cr.Status); !ok {
		logger.Debug("could not handle credential request as it did not have a status subresource")
		return nil
	}

	if ok := h.shouldDeprovisionWebhook(cr); ok {
		logger.Trace("processing resource in deleting state via provisioning webhook")
		return h.onWebhookDeprovision(ctx, cr)
	}

	if ok := h.shouldProcessDeleting(cr); ok {
		logger.Trace("processing resource in deleting state")
		h.onDeleting(ctx, cr)
		return nil
	}

	if c, ok := h.prov.(prov.CustomCredential); ok {
		creds := c.GetIgnoredCredentialTypes()
		for _, cred := range creds {
			if cred == cr.Spec.CredentialRequestDefinition {
				logger.WithField("crdName", cred).Debug("skipping handling credential provisioning")
				return nil
			}
		}
	}

	if ok := h.shouldProvisionWebhook(cr); ok {
		logger.Trace("processing resource in pending status via provisioning webhook")
		return h.onWebhookProvision(ctx, cr)
	}

	if actions := h.shouldUpdateWebhook(cr); len(actions) != 0 {
		logger.Trace("processing resource in updating status via provisioning webhook")
		return h.onWebhookUpdate(ctx, cr, actions)
	}

	var credential *management.Credential
	if ok := h.shouldProcessPending(cr); ok {
		logger.Trace("processing resource in pending status")
		credential = h.onPending(ctx, cr)
	} else if actions := h.shouldProcessUpdating(cr); len(actions) != 0 {
		logger.Trace("processing resource in updating status")
		credential = h.onUpdates(ctx, cr, actions)
	}

	if credential != nil {
		err = h.client.CreateSubResource(cr.ResourceMeta, cr.SubResources)
		if err != nil {
			logger.WithError(err).Error("error creating subresources")
		}

		// update the status resource regardless of errors updating the other subresources
		statusErr := h.client.CreateSubResource(credential.ResourceMeta, map[string]interface{}{"status": credential.Status})
		if statusErr != nil {
			logger.WithError(statusErr).Error("error creating status subresources")
			return statusErr
		}
	}

	return err
}

// shouldProcessDeleting
// Finalizers = has agent finalizer and
//  (Spec.State.Name = Inactive, StateReason = Credential Expired, Status.Level = Pending) or
//  (Metadata.State = Deleting)

func (h *credentials) shouldProcessDeleting(cr *management.Credential) bool {
	if !hasAgentCredentialFinalizer(cr.Finalizers) {
		return false
	}

	if cr.Spec.State.Name == v1.Inactive && cr.Spec.State.Reason == prov.CredExpDetail && cr.Status.Level == prov.Pending.String() {
		// expired credential
		return true
	}

	if cr.Metadata.State == v1.ResourceDeleting {
		// don't process delete when error from agent
		return !hasAgentCredentialError(cr.Status)
	}

	return false
}

// shouldDeprovisionWebhook returns true when the credential is deleting and a provisioning webhook is
// configured - i.e. this event should be dispatched to the webhook instead of the agent's own registered
// Provisioning implementation.
func (h *credentials) shouldDeprovisionWebhook(cr *management.Credential) bool {
	return h.shouldProcessDeleting(cr) && h.webhookCfg.IsConfigured()
}

// shouldProvision
// Status.Level = Pending and
// Metadata.State = !Deleting and
// Spec.State.Name = Active and
// Spec.State.Rotate = false and
// Finalizers = no agent finalizer
func (h *credentials) shouldProcessPending(cr *management.Credential) bool {
	if h.marketplaceHandler.shouldProcessPending(cr.Status, cr.Metadata.State) {
		return cr.Spec.State.Name == v1.Active && !cr.Spec.State.Rotate && !hasAgentCredentialFinalizer(cr.Finalizers)
	}
	return false
}

// shouldProvisionWebhook returns true when the credential is pending and a provisioning webhook is
// configured - i.e. this event should be dispatched to the webhook instead of the agent's own registered
// Provisioning implementation.
func (h *credentials) shouldProvisionWebhook(cr *management.Credential) bool {
	return h.shouldProcessPending(cr) && h.webhookCfg.IsConfigured()
}

// shouldUpdateWebhook returns the pending credential update actions when a provisioning webhook is
// configured - i.e. this event should be dispatched to the webhook instead of the agent's own registered
// Provisioning implementation.
func (h *credentials) shouldUpdateWebhook(cr *management.Credential) []prov.CredentialAction {
	if !h.webhookCfg.IsConfigured() {
		return nil
	}
	return h.shouldProcessUpdating(cr)
}

// shouldProcessUpdating
func (h *credentials) shouldProcessUpdating(cr *management.Credential) []prov.CredentialAction {
	actions := []prov.CredentialAction{}
	inter := reflect.TypeOf((*credProv)(nil)).Elem()
	if !reflect.TypeOf(h.prov).Implements(inter) {
		log.Debugf("credential updates not supported by agent")
		return actions
	}

	if !hasAgentCredentialFinalizer(cr.Finalizers) || cr.Status.Level != prov.Pending.String() {
		return actions
	}

	// suspend
	if cr.Spec.State.Name == v1.Inactive && (cr.State.Name == v1.Active || cr.State.Name == "") {
		actions = append(actions, prov.Suspend)
	}

	// rotate
	if cr.Spec.State.Rotate {
		actions = append(actions, prov.Rotate)
	}

	// enable
	if cr.Spec.State.Name == v1.Active && cr.State.Name == v1.Inactive {
		actions = append(actions, prov.Enable)
	}
	return actions
}

func (h *credentials) onDeleting(ctx context.Context, cred *management.Credential) {
	logger := getLoggerFromContext(ctx)

	app, provCreds, err := h.buildDeprovisionCreds(ctx, cred)
	if err != nil {
		h.onError(ctx, cred, err)
		return
	}

	status := h.prov.CredentialDeprovision(provCreds)

	h.deprovisionPostProcess(status, provCreds, logger, ctx, cred, app)
}

// buildDeprovisionCreds fetches the credential request definition and managed app, and builds the
// deprovisioning request - shared by the classic and webhook-dispatch paths, which only differ in how they
// report a failure here (classic reports via onError, webhook logs only since setting credential status is
// the webhook's responsibility).
func (h *credentials) buildDeprovisionCreds(ctx context.Context, cred *management.Credential) (*management.ManagedApplication, *provCreds, error) {
	logger := getLoggerFromContext(ctx)

	crd, err := h.getCRD(ctx, cred)
	if err != nil {
		logger.WithError(err).Error("error getting credential request definition")
		return nil, nil, err
	}
	app, err := h.getManagedApp(ctx, cred)
	if err != nil {
		logger.WithError(err).Error("error getting managed app")
		return nil, nil, err
	}
	provCreds, err := h.newProvCreds(cred, app, 0, crd)
	if err != nil {
		logger.WithError(err).Error("error preparing credential request")
		return nil, nil, err
	}
	return app, provCreds, nil
}

// onWebhookDeprovision dispatches the deprovision request to the configured webhook instead of calling this
// agent's own registered Provisioning implementation.
func (h *credentials) onWebhookDeprovision(ctx context.Context, cred *management.Credential) error {
	logger := getLoggerFromContext(ctx)

	if webhookDispatchedFor(cred, webhookOperationDeprovision) {
		return nil
	}

	_, provCreds, err := h.buildDeprovisionCreds(ctx, cred)
	if err != nil {
		h.onError(ctx, cred, err)
		return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
	}

	if err := provisioningwebhook.Dispatch(h.webhookClient, h.webhookCfg, newWebhookCredentialRequest(webhookOperationDeprovision, provCreds)); err != nil {
		logger.WithError(err).Error("provisioning webhook dispatch failed")
		h.onError(ctx, cred, err)
		return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
	}
	markWebhookDispatched(cred, webhookOperationDeprovision)
	return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
}

// postWebhookDeprovisionProcess removes the finalizer once the webhook reports it successfully completed a
// deprovision. cred.Status can't be used for this - it's whatever it already was when
// shouldDeprovisionWebhook let the event through, not a signal from the webhook - so success/failure is read
// from the reserved "status" (and, on failure, "message") keys the webhook writes inside x-webhook-details
// itself. Status itself is the webhook's own responsibility (see docs/discovery/provisioning-webhook.md), so
// on failure this only logs - it doesn't write anything. Only called when the dispatched operation was a
// deprovision - see Handle.
func (h *credentials) postWebhookDeprovisionProcess(log log.FieldLogger, cred *management.Credential) {
	if webhookDetailsValue(cred, webhookStatusKey) != webhookStatusSuccess {
		message := webhookDetailsValue(cred, webhookMessageKey)
		log.WithField("message", message).Error("provisioning webhook reported deprovision failure")
		return
	}

	if hasAgentCredentialFinalizer(cred.Finalizers) {
		ri, _ := cred.AsInstance()
		h.client.UpdateResourceFinalizer(ri, crFinalizer, "", false)
	}
}

func (h *credentials) deprovisionPostProcess(status prov.RequestStatus, provCreds *provCreds, logger log.FieldLogger,
	ctx context.Context, cred *management.Credential, app *management.ManagedApplication) {
	if status.GetStatus() != prov.Success {
		err := errors.New(status.GetMessage())
		logger.WithError(err).Error("request status was not Success, skipping")
		h.onError(ctx, cred, err)
		h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
		return
	}

	if provCreds.IsIDPCredential() && !isExternalCredential(cred) {
		clientID := provCreds.idpProvisioner.GetIDPCredentialData().GetClientID()
		tokenURL := provCreds.GetIDPProvider().GetTokenEndpoint()
		providerName := provCreds.GetIDPProvider().GetName()
		logger = logger.WithField("client_id", clientID).WithField("provider", providerName).WithField("tokenURL", tokenURL)

		unregisterErr := provCreds.idpProvisioner.UnregisterClient()
		if unregisterErr != nil {
			logger.WithError(unregisterErr).Warn("error deprovisioning credential request from IDP, please ask administrator to remove the client from IdP")
		} else {
			if err := removeClientIDFromManagedApp(logger, h.client, app, clientID, tokenURL); err != nil {
				logger.WithError(err).Warn("error removing clientID from Managed App clientIDs array")
			}
		}
	}

	ri, _ := cred.AsInstance()
	h.client.UpdateResourceFinalizer(ri, crFinalizer, "", false)

	// update sub resources when expire
	if cred.Metadata.State == v1.ResourceDeleting {
		return
	}

	cred.State.Name = v1.Inactive
	cred.Status.Level = prov.Success.String()
	cred.Status.Reasons = []v1.ResourceStatusReason{}
	h.client.CreateSubResource(cred.ResourceMeta, map[string]interface{}{
		"state": cred.State,
	})
	h.client.CreateSubResource(cred.ResourceMeta, map[string]interface{}{
		"status": cred.Status,
	})
}

func (h *credentials) onPending(ctx context.Context, cred *management.Credential) *management.Credential {
	// check the application status
	logger := getLoggerFromContext(ctx)

	app, crd, shouldReturn := h.provisionPreProcess(ctx, cred)
	if shouldReturn {
		return cred
	}

	provCreds, err := h.newProvCreds(cred, app, 0, crd)
	if err != nil {
		logger.WithError(err).Error("error preparing credential request")
		h.onError(ctx, cred, err)
		return cred
	}

	if err := h.registerIDPClient(cred, provCreds, app, logger); err != nil {
		logger.WithError(err).Error("error registering IDP client")
		h.onError(ctx, cred, err)
		return cred
	}

	status, credentialData := h.provision(provCreds)

	h.provisionPostProcess(status, credentialData, app, crd, provCreds, cred)

	return cred
}

// onWebhookProvision dispatches the provision request to the configured webhook instead of calling this
// agent's own registered Provisioning implementation. Unlike onPending, it does not call registerIDPClient -
// IDP client registration is the webhook's own responsibility in webhook mode.
func (h *credentials) onWebhookProvision(ctx context.Context, cred *management.Credential) error {
	logger := getLoggerFromContext(ctx)

	if webhookDispatchedFor(cred, webhookOperationProvision) {
		return nil
	}

	app, crd, shouldReturn := h.provisionPreProcess(ctx, cred)
	if shouldReturn {
		return nil
	}

	provCreds, err := h.newProvCreds(cred, app, 0, crd)
	if err != nil {
		logger.WithError(err).Error("error preparing credential request")
		h.onError(ctx, cred, err)
		return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
	}

	if err := provisioningwebhook.Dispatch(h.webhookClient, h.webhookCfg, newWebhookCredentialRequest(webhookOperationProvision, provCreds)); err != nil {
		logger.WithError(err).Error("provisioning webhook dispatch failed")
		h.onError(ctx, cred, err)
		return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
	}
	markWebhookDispatched(cred, webhookOperationProvision)
	return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
}

// postWebhookProvisionProcess mirrors the webhook's x-webhook-details into x-agent-details, and adds the
// finalizer once status reflects success - status is the webhook's own responsibility (see
// docs/discovery/provisioning-webhook.md), so cred.Status already reflects the outcome by the time this
// runs. Only called when the dispatched operation was a provision - see Handle.
func (h *credentials) postWebhookProvisionProcess(cred *management.Credential) error {
	// normal (non-webhook) provisioning always sets this so agents-controller can schedule credential
	// expiry off the status subresource
	util.SetAgentDetailsKey(cred, prov.HandleCredentialExpiry, "true")

	if cred.Status.Level == prov.Success.String() && !hasAgentCredentialFinalizer(cred.Finalizers) {
		// only add finalizer on success
		ri, _ := cred.AsInstance()
		h.client.UpdateResourceFinalizer(ri, crFinalizer, "", true)
	}
	if mirrorWebhookDetails(cred) {
		return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
	}
	return nil
}

func (h *credentials) provisionPreProcess(ctx context.Context, cred *management.Credential) (*management.ManagedApplication, *management.CredentialRequestDefinition, bool) {
	logger := getLoggerFromContext(ctx)
	app, err := h.getManagedApp(ctx, cred)
	if err != nil {
		logger.WithError(err).Error("error getting managed app")
		h.onError(ctx, cred, err)
		return nil, nil, true
	}

	if app.Status.Level != prov.Success.String() {
		err = fmt.Errorf("cannot handle credential when application is not yet successful")
		h.onError(ctx, cred, err)
		return nil, nil, true
	}

	crd, err := h.getCRD(ctx, cred)
	if err != nil {
		logger.WithError(err).Error("error getting credential request definition")
		h.onError(ctx, cred, err)
		return nil, nil, true
	}

	updateDataFromEnumMap(cred.Spec.Data, crd.Spec.Schema)

	return app, crd, false
}

func (h *credentials) provision(cr prov.CredentialRequest) (prov.RequestStatus, prov.Credential) {
	status, credentialData := h.prov.CredentialProvision(cr)
	if status.GetStatus() == prov.Success {
		return status, credentialData
	}

	timeout := baseRetryTimeout
	for range h.retryCount {
		if util.IsNotTest() {
			time.Sleep(timeout)
		}
		timeout = timeout * 2

		status, credentialData = h.prov.CredentialProvision(cr)
		if status.GetStatus() == prov.Success {
			return status, credentialData
		}
	}
	return status, credentialData
}

func (h *credentials) provisionPostProcess(status prov.RequestStatus, credentialData prov.Credential, app *management.ManagedApplication, crd *management.CredentialRequestDefinition, provCreds *provCreds, cred *management.Credential) {
	var err error
	data := map[string]interface{}{}
	idpAgentDetails := make(map[string]string)
	isExternal := isExternalCredential(cred)
	if status.GetStatus() == prov.Success {
		credentialData := h.getProvisionedCredentialData(provCreds, credentialData)
		if credentialData != nil {
			if !isExternal {
				sec := app.Spec.Security
				d := credentialData.GetData()
				if crd.Spec.Provision == nil {
					data = d
				} else if d != nil {
					data, err = h.encryptSchema(
						crd.Spec.Provision.Schema,
						d,
						sec.EncryptionKey, sec.EncryptionAlgorithm, sec.EncryptionHash,
					)
				}
			}
			if provCreds.IsIDPCredential() {
				idpAgentDetails, err = provCreds.idpProvisioner.GetAgentDetails()
			}
			if err != nil {
				status = prov.NewRequestStatusBuilder().
					SetMessage(fmt.Sprintf("error encrypting credential: %s", err.Error())).
					SetCurrentStatusReasons(cred.Status.Reasons).
					Failed()
			}
		}
	}

	cred.Data = data
	cred.Status = prov.NewStatusReason(status)

	// use the expiration time sent back with the data
	if credentialData != nil && !credentialData.GetExpirationTime().IsZero() {
		cred.Policies.Expiry = &management.CredentialPoliciesExpiry{
			Timestamp: v1.Time(credentialData.GetExpirationTime()),
		}
	} else if provCreds.days != 0 {
		// update the expiration timestamp
		expTS := time.Now().AddDate(0, 0, provCreds.days)

		cred.Policies.Expiry = &management.CredentialPoliciesExpiry{
			Timestamp: v1.Time(expTS),
		}
	}

	details := util.MergeMapStringString(util.GetAgentDetailStrings(cred), status.GetProperties(), idpAgentDetails)
	// Adding this flag to perform the credential expiry for on-prem agents in agents-controller through status subresoruce event.
	details[prov.HandleCredentialExpiry] = "true"
	util.SetAgentDetails(cred, util.MapStringStringToMapStringInterface(details))

	h.processCredentialLevelSuccess(provCreds, cred)

	if isExternal {
		cred.SubResources = map[string]interface{}{
			defs.XAgentDetails: util.GetAgentDetails(cred),
			"state":            cred.State,
		}
		return
	}

	cred.SubResources = map[string]interface{}{
		defs.XAgentDetails: util.GetAgentDetails(cred),
		"policies":         cred.Policies,
		"state":            cred.State,
	}
	if !isSuspendOrEnableAction(provCreds) {
		cred.SubResources["data"] = cred.Data
	}
}

func isSuspendOrEnableAction(provCreds *provCreds) bool {
	if provCreds == nil {
		return false
	}

	action := provCreds.GetCredentialAction()
	return action == prov.Suspend || action == prov.Enable
}

func (h *credentials) processCredentialLevelSuccess(provCreds *provCreds, cred *management.Credential) {
	if cred.Status.Level == prov.Success.String() {
		if !hasAgentCredentialFinalizer(cred.Finalizers) {
			ri, _ := cred.AsInstance()
			// only add finalizer on success
			h.client.UpdateResourceFinalizer(ri, crFinalizer, "", true)
		}

		if provCreds.GetCredentialAction() != prov.Rotate {
			// if this is not a rotate action update the state to the desired state
			cred.State.Name = cred.Spec.State.Name
		} else {
			// if the action was rotate lets remove the rotate flag from spec
			cred.Spec.State.Rotate = false
			h.client.UpdateResourceInstance(cred)
		}
	} else if cred.State.Name == "" {
		cred.State.Name = v1.Inactive
	}
}

func (h *credentials) onUpdates(ctx context.Context, cred *management.Credential, actions []prov.CredentialAction) *management.Credential {
	logger := getLoggerFromContext(ctx)
	app, crd, shouldReturn := h.provisionPreProcess(ctx, cred)
	if shouldReturn {
		return cred
	}

	for _, action := range actions {
		provCreds, err := h.newProvCreds(cred, app, action, crd)
		if err != nil {
			logger.WithError(err).Error("error preparing credential request")
			h.onError(ctx, cred, err)
			return cred
		}

		if action == prov.Rotate {
			if err := h.registerIDPClient(cred, provCreds, app, logger); err != nil {
				logger.WithError(err).Error("error registering IDP client")
				h.onError(ctx, cred, err)
				return cred
			}
		}

		status, credentialData := h.prov.CredentialUpdate(provCreds)
		h.provisionPostProcess(status, credentialData, app, crd, provCreds, cred)
	}

	return cred
}

// onWebhookUpdate dispatches pending credential update actions (suspend/rotate/enable) to the configured
// webhook instead of calling this agent's own registered Provisioning implementation. Unlike onUpdates, it
// does not call registerIDPClient on rotate - IDP client registration is the webhook's own responsibility
// in webhook mode.
func (h *credentials) onWebhookUpdate(ctx context.Context, cred *management.Credential, actions []prov.CredentialAction) error {
	logger := getLoggerFromContext(ctx)

	if webhookDispatchedFor(cred, update) {
		return nil
	}

	app, crd, shouldReturn := h.provisionPreProcess(ctx, cred)
	if shouldReturn {
		return nil
	}

	for _, action := range actions {
		provCreds, err := h.newProvCreds(cred, app, action, crd)
		if err != nil {
			logger.WithError(err).Error("error preparing credential request")
			h.onError(ctx, cred, err)
			return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
		}

		if err := provisioningwebhook.Dispatch(h.webhookClient, h.webhookCfg, newWebhookCredentialRequest(update, provCreds)); err != nil {
			logger.WithError(err).Error("provisioning webhook dispatch failed")
			h.onError(ctx, cred, err)
			return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
		}
	}

	markWebhookDispatched(cred, update)
	return h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
}

// isExternalCredential - when mode is CredProvisionModeExternal the client was registered outside the SDK; skip RegisterClient.
func isExternalCredential(cred *management.Credential) bool {
	if cred == nil {
		return false
	}
	return cred.Spec.Provision != nil && cred.Spec.Provision.Mode == prov.CredProvisionModeExternal
}

// onError updates the AccessRequest with an error status
func (h *credentials) onError(_ context.Context, cred *management.Credential, err error) {
	ps := prov.NewRequestStatusBuilder()
	status := ps.SetMessage(fmt.Sprintf("Agent: %s", err.Error())).SetCurrentStatusReasons(cred.Status.Reasons).Failed()
	cred.Status = prov.NewStatusReason(status)
	cred.SubResources = map[string]interface{}{
		"status": cred.Status,
	}
}

func (h *credentials) getManagedApp(_ context.Context, cred *management.Credential) (*management.ManagedApplication, error) {
	app := management.NewManagedApplication(cred.Spec.ManagedApplication, cred.Metadata.Scope.Name)
	ri, err := h.client.GetResource(app.GetSelfLink())
	if err != nil {
		return nil, err
	}

	app = &management.ManagedApplication{}
	err = app.FromInstance(ri)
	return app, err
}

func (h *credentials) getCRD(_ context.Context, cred *management.Credential) (*management.CredentialRequestDefinition, error) {
	crd := management.NewCredentialRequestDefinition(cred.Spec.CredentialRequestDefinition, cred.Metadata.Scope.Name)
	ri, err := h.client.GetResource(crd.GetSelfLink())
	if err != nil {
		return nil, err
	}

	crd = &management.CredentialRequestDefinition{}
	err = crd.FromInstance(ri)
	return crd, err
}

func (h *credentials) getProvisionedCredentialData(provCreds *provCreds, credentialData prov.Credential) prov.Credential {
	if provCreds.IsIDPCredential() {
		return prov.NewCredentialBuilder().SetOAuthIDAndSecret(
			provCreds.GetIDPCredentialData().GetClientID(),
			provCreds.GetIDPCredentialData().GetClientSecret(),
		)
	}
	return credentialData
}

func hasAgentCredentialError(status *v1.ResourceStatus) bool {
	for _, r := range status.Reasons {
		if strings.HasPrefix(r.Detail, "Agent:") {
			return true
		}
	}
	return false
}

func hasAgentCredentialFinalizer(finalizers []v1.Finalizer) bool {
	for _, f := range finalizers {
		if f.Name == crFinalizer {
			return true
		}
	}
	return false
}

type provCreds struct {
	managedApp        string
	credType          string
	id                string
	name              string
	days              int
	credAction        prov.CredentialAction
	credData          map[string]interface{}
	credDetails       map[string]interface{}
	appDetails        map[string]interface{}
	idpProvisioner    idp.Provisioner
	credSchema        map[string]interface{}
	credProvSchema    map[string]interface{}
	credSchemaDetails map[string]interface{}
	provisionMode     string
}

func (h *credentials) newProvCreds(cr *management.Credential, app *management.ManagedApplication, action prov.CredentialAction, crd *management.CredentialRequestDefinition) (*provCreds, error) {
	credDetails := util.GetAgentDetails(cr)

	provisionMode := ""
	if cr.Spec.Provision != nil {
		provisionMode = cr.Spec.Provision.Mode
	}

	provCred := &provCreds{
		appDetails:    util.GetAgentDetails(app),
		credDetails:   credDetails,
		credType:      cr.Spec.CredentialRequestDefinition,
		credData:      cr.Spec.Data,
		managedApp:    cr.Spec.ManagedApplication,
		id:            cr.Metadata.ID,
		name:          cr.Name,
		credAction:    action,
		days:          0,
		provisionMode: provisionMode,
	}

	if crd != nil {
		if crd.Spec.Provision != nil &&
			crd.Spec.Provision.Policies.Expiry != nil {
			provCred.days = int(crd.Spec.Provision.Policies.Expiry.Period)
		}

		credSchemaDetails := util.GetAgentDetails(crd)
		provCred.credSchema = crd.Spec.Schema
		if crd.Spec.Provision != nil {
			provCred.credProvSchema = crd.Spec.Provision.Schema
		}
		provCred.credSchemaDetails = credSchemaDetails
	}
	idpProvisioner, err := idp.NewProvisioner(context.Background(), h.idpProviderRegistry, app, cr)
	if err != nil {
		return nil, fmt.Errorf("IDP provider not found for credential request")
	}
	provCred.idpProvisioner = idpProvisioner
	return provCred, nil
}

// registerIDPClient registers a new IDP client and (for Okta) persists the reference
// on the managed app. It is a no-op when not an IDP credential or is external.
func (h *credentials) registerIDPClient(cred *management.Credential, provCreds *provCreds, app *management.ManagedApplication, logger log.FieldLogger) error {
	if !provCreds.IsIDPCredential() || isExternalCredential(cred) {
		return nil
	}
	if err := provCreds.idpProvisioner.RegisterClient(); err != nil {
		return fmt.Errorf("error provisioning credential request with IDP: %w", err)
	}
	if provCreds.GetIDPProvider().GetConfig().GetIDPType() != oauth.TypeOkta {
		return nil
	}
	return persistIDPClientOnManagedApplication(logger, h.client, app,
		provCreds.GetIDPCredentialData().GetClientID(),
		provCreds.GetIDPProvider().GetTokenEndpoint(),
	)
}

// GetApplicationName gets the name of the managed application
func (c provCreds) GetApplicationName() string {
	return c.managedApp
}

// GetID gets the id of the credential resource
func (c provCreds) GetID() string {
	return c.id
}

// GetName gets the name of the credential resource
func (c provCreds) GetName() string {
	return c.name
}

// GetCredentialType gets the type of the credential
func (c provCreds) GetCredentialType() string {
	return c.credType
}

// GetCredentialData gets the data of the credential
func (c provCreds) GetCredentialData() map[string]interface{} {
	return c.credData
}

// GetCredentialAction gets the data of the credential
func (c provCreds) GetCredentialAction() prov.CredentialAction {
	return c.credAction
}

// GetProvisionMode returns the provisioning mode for the credential (e.g. "external")
func (c provCreds) GetProvisionMode() string {
	return c.provisionMode
}

// GetID gets the id of the credential resource
func (c provCreds) GetCredentialExpirationDays() int {
	return c.days
}

// GetCredentialSchema returns the schema for the credential request.
func (c provCreds) GetCredentialSchema() map[string]interface{} {
	return c.credSchema
}

// GetCredentialProvisionSchema returns the provisioning schema for the credential request.
func (c provCreds) GetCredentialProvisionSchema() map[string]interface{} {
	return c.credProvSchema
}

// GetCredentialSchemaDetailsValue returns a value found on the 'x-agent-details' sub resource of the crd.
func (c provCreds) GetCredentialSchemaDetailsValue(key string) interface{} {
	if c.credSchemaDetails == nil {
		return nil
	}

	return c.credSchemaDetails[key]
}

// IsIDPCredential returns boolean indicating if the credential request is for IDP provider
func (c provCreds) IsIDPCredential() bool {
	return c.idpProvisioner.IsIDPCredential()
}

// GetIDPProvider returns the interface for IDP provider if the credential request is for IDP provider
func (c provCreds) GetIDPProvider() oauth.Provider {
	return c.idpProvisioner.GetIDPProvider()
}

// GetIDPCredentialData returns the credential data for IDP from the request
func (c provCreds) GetIDPCredentialData() prov.IDPCredentialData {
	return c.idpProvisioner.GetIDPCredentialData()
}

// GetCredentialDetailsValue returns a value found on the 'x-agent-details' sub resource of the Credentials.
func (c provCreds) GetCredentialDetailsValue(key string) string {
	if c.credDetails == nil {
		return ""
	}

	return util.ToString(c.credDetails[key])
}

// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (c provCreds) GetApplicationDetailsValue(key string) string {
	if c.appDetails == nil {
		return ""
	}

	return util.ToString(c.appDetails[key])
}

// encryptSchema schema is the json schema. credData is the data that contains data to encrypt based on the key, alg and hash.
func encryptSchema(
	schema, credData map[string]interface{}, key, alg, hash string,
) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	enc, err := util.NewEncryptor(key, alg, hash)
	if err != nil {
		return data, err
	}

	schemaProps, ok := schema["properties"]
	if !ok {
		return data, fmt.Errorf("properties field not found on schema")
	}

	props, ok := schemaProps.(map[string]interface{})
	if !ok {
		props = make(map[string]interface{})
	}

	return encryptMap(enc, props, credData), nil
}

// encryptMap loops through all data and checks the value against the provisioning schema to see if it should be encrypted.
func encryptMap(enc util.Encryptor, schema, data map[string]interface{}) map[string]interface{} {
	for key, value := range data {
		schemaValue := schema[key]
		v, ok := schemaValue.(map[string]interface{})
		if !ok {
			continue
		}

		if _, ok := v[xAxwayEncrypted]; ok {
			v, ok := value.(string)
			if !ok {
				continue
			}

			encrypted, err := enc.Encrypt(v)
			if err != nil {
				log.Error(err)
				continue
			}

			data[key] = encrypted
		}
	}

	return data
}
