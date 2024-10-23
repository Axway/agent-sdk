package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning/idp"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
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
}

// encryptSchemaFunc func signature for encryptSchema
type encryptSchemaFunc func(schema, credData map[string]interface{}, key, alg, hash string) (map[string]interface{}, error)

// NewCredentialHandler creates a Handler for Credentials
func NewCredentialHandler(prov credProv, client client, providerRegistry oauth.IdPRegistry) Handler {
	return &credentials{
		prov:                prov,
		client:              client,
		encryptSchema:       encryptSchema,
		idpProviderRegistry: providerRegistry,
	}
}

// Handle processes grpc events triggered for Credentials
func (h *credentials) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.CredentialGVK().Kind || h.prov == nil || h.shouldIgnoreSubResourceUpdate(action, meta) {
		return nil
	}

	logger := getLoggerFromContext(ctx).WithComponent("credentialHandler")
	ctx = setLoggerInContext(ctx, logger)

	cr := &management.Credential{}
	err := cr.FromInstance(resource)
	if err != nil {
		logger.WithError(err).Error("could not handle credential request")
		return nil
	}

	if ok := isStatusFound(cr.Status); !ok {
		logger.Debugf("could not handle credential request as it did not have a status subresource")
		return nil
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

	var credential *management.Credential
	if ok := h.shouldProcessPending(cr); ok {
		log.Trace("processing resource in pending status")
		credential = h.onPending(ctx, cr)
	} else if actions := h.shouldProcessUpdating(cr); len(actions) != 0 {
		log.Trace("processing resource in updating status")
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
	provData := h.deprovisionPreProcess(ctx, cred)
	crd, err := h.getCRD(ctx, cred)
	if err != nil {
		logger.WithError(err).Error("error getting credential request definition")
		h.onError(ctx, cred, err)
		return
	}
	app, err := h.getManagedApp(ctx, cred)
	if err != nil {
		logger.WithError(err).Error("error getting managed app")
		h.onError(ctx, cred, err)
		return
	}

	provCreds, err := h.newProvCreds(cred, app, provData, 0, crd)

	if err != nil {
		logger.WithError(err).Error("error preparing credential request")
		h.onError(ctx, cred, err)
		return
	}

	status := h.prov.CredentialDeprovision(provCreds)

	h.deprovisionPostProcess(status, provCreds, logger, ctx, cred)
}

func (*credentials) deprovisionPreProcess(_ context.Context, cred *management.Credential) map[string]interface{} {
	var provData map[string]interface{}
	if cred.Data != nil {
		if m, ok := cred.Data.(map[string]interface{}); ok {
			provData = m
		}
	}
	return provData
}

func (h *credentials) deprovisionPostProcess(status prov.RequestStatus, provCreds *provCreds, logger log.FieldLogger, ctx context.Context, cred *management.Credential) {
	if status.GetStatus() == prov.Success {
		if provCreds.IsIDPCredential() {
			err := provCreds.idpProvisioner.UnregisterClient()
			if err != nil {
				logger.
					WithError(err).
					WithField("client_id", provCreds.idpProvisioner.GetIDPCredentialData().GetClientID()).
					WithField("provider", provCreds.GetIDPProvider().GetName()).
					Warn("error deprovisioning credential request from IDP, please ask administrator to remove the client from IdP")
			}
		}

		ri, _ := cred.AsInstance()
		h.client.UpdateResourceFinalizer(ri, crFinalizer, "", false)

		// update sub resources when expire
		if cred.Metadata.State != v1.ResourceDeleting {
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
	} else {
		err := fmt.Errorf(status.GetMessage())
		logger.WithError(err).Error("request status was not Success, skipping")
		h.onError(ctx, cred, err)
		h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
	}
}

func (h *credentials) onPending(ctx context.Context, cred *management.Credential) *management.Credential {
	// check the application status
	logger := getLoggerFromContext(ctx)
	app, crd, shouldReturn := h.provisionPreProcess(ctx, cred)
	if shouldReturn {
		return cred
	}

	provCreds, err := h.newProvCreds(cred, app, nil, 0, crd)
	if err != nil {
		logger.WithError(err).Error("error preparing credential request")
		h.onError(ctx, cred, err)
		return cred
	}

	if provCreds.IsIDPCredential() {
		err := provCreds.idpProvisioner.RegisterClient()
		if err != nil {
			logger.WithError(err).Error("error provisioning credential request with IDP")
			h.onError(ctx, cred, err)
			return cred
		}
	}

	status, credentialData := h.prov.CredentialProvision(provCreds)

	h.provisionPostProcess(status, credentialData, app, crd, provCreds, cred)

	return cred
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

	return app, crd, false
}

func (h *credentials) provisionPostProcess(status prov.RequestStatus, credentialData prov.Credential, app *management.ManagedApplication, crd *management.CredentialRequestDefinition, provCreds *provCreds, cred *management.Credential) {
	var err error
	data := map[string]interface{}{}
	idpAgentDetails := make(map[string]string)
	if status.GetStatus() == prov.Success {
		credentialData := h.getProvisionedCredentialData(provCreds, credentialData)
		if credentialData != nil {
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
	util.SetAgentDetails(cred, util.MapStringStringToMapStringInterface(details))

	h.processCredentialLevelSuccess(provCreds, cred)

	cred.SubResources = map[string]interface{}{
		defs.XAgentDetails: util.GetAgentDetails(cred),
		"data":             cred.Data,
		"policies":         cred.Policies,
		"state":            cred.State,
	}
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
	provData := h.deprovisionPreProcess(ctx, cred)
	if shouldReturn {
		return cred
	}

	for _, action := range actions {
		provCreds, err := h.newProvCreds(cred, app, provData, action, crd)
		if err != nil {
			logger.WithError(err).Error("error preparing credential request")
			h.onError(ctx, cred, err)
			return cred
		}

		if action != prov.Suspend && provCreds.IsIDPCredential() {
			err := provCreds.idpProvisioner.RegisterClient()
			if err != nil {
				logger.WithError(err).Error("error provisioning credential request with IDP")
				h.onError(ctx, cred, err)
				return cred
			}
		}

		status, credentialData := h.prov.CredentialUpdate(provCreds)
		h.provisionPostProcess(status, credentialData, app, crd, provCreds, cred)
	}

	return cred
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
}

func (h *credentials) newProvCreds(cr *management.Credential, app *management.ManagedApplication, provData map[string]interface{}, action prov.CredentialAction, crd *management.CredentialRequestDefinition) (*provCreds, error) {
	credDetails := util.GetAgentDetails(cr)

	provCred := &provCreds{
		appDetails:  util.GetAgentDetails(app),
		credDetails: credDetails,
		credType:    cr.Spec.CredentialRequestDefinition,
		credData:    cr.Spec.Data,
		managedApp:  cr.Spec.ManagedApplication,
		id:          cr.Metadata.ID,
		name:        cr.Name,
		credAction:  action,
		days:        0,
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

			str, err := enc.Encrypt(v)
			if err != nil {

				log.Error(err)
				continue
			}

			data[key] = base64.StdEncoding.EncodeToString([]byte(str))
		}
	}

	return data
}
