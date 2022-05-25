package handler

import (
	"context"
	"encoding/base64"
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	xAxwayEncrypted = "x-axway-encrypted"
	crFinalizer     = "agent.credential.provisioned"
)

type credProv interface {
	CredentialProvision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential)
	CredentialDeprovision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus)
}

type credentials struct {
	prov          credProv
	client        client
	encryptSchema encryptSchemaFunc
}

// encryptSchemaFunc func signature for encryptSchema
type encryptSchemaFunc func(schema, credData map[string]interface{}, key, alg, hash string) (map[string]interface{}, error)

// NewCredentialHandler creates a Handler for Credentials
func NewCredentialHandler(prov credProv, client client) Handler {
	return &credentials{
		prov:          prov,
		client:        client,
		encryptSchema: encryptSchema,
	}
}

// Handle processes grpc events triggered for Credentials
func (h *credentials) Handle(ctx context.Context, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != mv1.CredentialGVK().Kind || h.prov == nil || isNotStatusSubResourceUpdate(action, meta) {
		return nil
	}

	logger := getLoggerFromContext(ctx).WithComponent("credentialHandler")
	ctx = setLoggerInContext(ctx, logger)

	cr := &mv1.Credential{}
	err := cr.FromInstance(resource)
	if err != nil {
		logger.WithError(err).Error("could not handle credential request")
		return nil
	}

	if ok := isStatusFound(cr.Status); !ok {
		logger.Debugf("could not handle credential request as it did not have a status subresource")
		return nil
	}

	if ok := shouldProcessPending(cr.Status.Level, cr.Metadata.State); ok {
		logger.Tracef("processing resource in pending status")
		cr := h.onPending(ctx, cr)
		err := h.client.CreateSubResource(cr.ResourceMeta, cr.SubResources)
		if err != nil {
			logger.WithError(err).Error("error creating subresources")
			return err
		}
		return h.client.CreateSubResource(cr.ResourceMeta, map[string]interface{}{"status": cr.Status})
	}

	if ok := shouldProcessDeleting(cr.Status.Level, cr.Metadata.State, len(cr.Finalizers)); ok {
		logger.Trace("processing resource in deleting state")
		h.onDeleting(ctx, cr)
	}

	return nil
}

func (h *credentials) onPending(ctx context.Context, cred *mv1.Credential) *mv1.Credential {
	logger := getLoggerFromContext(ctx)
	app, err := h.getManagedApp(ctx, cred)
	if err != nil {
		logger.WithError(err).Error("error getting managed app")
		h.onError(ctx, cred, err)
		return cred
	}

	// check the application status
	if app.Status.Level != prov.Success.String() {
		err = fmt.Errorf("cannot handle credential when application is not yet successful")
		h.onError(ctx, cred, err)
		return cred
	}

	crd, err := h.getCRD(ctx, cred)
	if err != nil {
		logger.WithError(err).Errorf("error getting resource details: %s")
		h.onError(ctx, cred, err)
		return cred
	}

	provCreds := newProvCreds(cred, util.GetAgentDetails(app))
	status, credentialData := h.prov.CredentialProvision(provCreds)

	if status.GetStatus() == prov.Success {
		sec := app.Spec.Security
		data, err := h.encryptSchema(
			crd.Spec.Provision.Schema,
			credentialData.GetData(),
			sec.EncryptionKey, sec.EncryptionAlgorithm, sec.EncryptionHash,
		)

		if err != nil {
			status = prov.NewRequestStatusBuilder().
				SetMessage(fmt.Sprintf("error encrypting credential: %s", err.Error())).
				Failed()
		} else {
			cred.Data = data
		}
	}

	cred.Status = prov.NewStatusReason(status)

	details := util.MergeMapStringString(util.GetAgentDetailStrings(cred), status.GetProperties())
	util.SetAgentDetails(cred, util.MapStringStringToMapStringInterface(details))

	ri, _ := cred.AsInstance()
	if cred.Status.Level == prov.Success.String() {
		// only add finalizer on success
		h.client.UpdateResourceFinalizer(ri, crFinalizer, "", true)
	}
	cred.SubResources = map[string]interface{}{
		defs.XAgentDetails: util.GetAgentDetails(cred),
		"data":             cred.Data,
	}

	return cred
}

func (h *credentials) onDeleting(ctx context.Context, cred *mv1.Credential) {
	provCreds := newProvCreds(cred, map[string]interface{}{})
	status := h.prov.CredentialDeprovision(provCreds)
	logger := getLoggerFromContext(ctx)

	if status.GetStatus() == prov.Success {
		ri, _ := cred.AsInstance()
		h.client.UpdateResourceFinalizer(ri, crFinalizer, "", false)
	} else {
		err := fmt.Errorf(status.GetMessage())
		logger.WithError(err).Error("request status was not Success, skipping")
		h.onError(ctx, cred, err)
		h.client.CreateSubResource(cred.ResourceMeta, cred.SubResources)
	}
}

// onError updates the AccessRequest with an error status
func (h *credentials) onError(_ context.Context, cred *mv1.Credential, err error) {
	ps := prov.NewRequestStatusBuilder()
	status := ps.SetMessage(err.Error()).Failed()
	cred.Status = prov.NewStatusReason(status)
	cred.SubResources = map[string]interface{}{
		"status": cred.Status,
	}
}

func (h *credentials) getManagedApp(ctx context.Context, cred *mv1.Credential) (*mv1.ManagedApplication, error) {
	app := mv1.NewManagedApplication(cred.Spec.ManagedApplication, cred.Metadata.Scope.Name)
	ri, err := h.client.GetResource(app.GetSelfLink())
	if err != nil {
		return nil, err
	}

	app = &mv1.ManagedApplication{}
	err = app.FromInstance(ri)
	return app, err
}

func (h *credentials) getCRD(ctx context.Context, cred *mv1.Credential) (*mv1.CredentialRequestDefinition, error) {
	crd := mv1.NewCredentialRequestDefinition(cred.Spec.CredentialRequestDefinition, cred.Metadata.Scope.Name)
	ri, err := h.client.GetResource(crd.GetSelfLink())
	if err != nil {
		return nil, err
	}

	crd = &mv1.CredentialRequestDefinition{}
	err = crd.FromInstance(ri)
	return crd, err
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

type provCreds struct {
	managedApp  string
	credType    string
	credData    map[string]interface{}
	credDetails map[string]interface{}
	appDetails  map[string]interface{}
}

func newProvCreds(cr *mv1.Credential, appDetails map[string]interface{}) *provCreds {
	credDetails := util.GetAgentDetails(cr)

	return &provCreds{
		appDetails:  appDetails,
		credDetails: credDetails,
		credType:    cr.Spec.CredentialRequestDefinition,
		credData:    cr.Spec.Data,
		managedApp:  cr.Spec.ManagedApplication,
	}
}

// GetApplicationName gets the name of the managed application
func (c provCreds) GetApplicationName() string {
	return c.managedApp
}

// GetCredentialType gets the type of the credential
func (c provCreds) GetCredentialType() string {
	return c.credType
}

// GetCredentialData gets the data of the credential
func (c provCreds) GetCredentialData() map[string]interface{} {
	return c.credData
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
	enc, err := util.NewEncryptor(key, alg, hash)
	if err != nil {
		return nil, err
	}

	schemaProps, ok := schema["properties"]
	if !ok {
		return nil, fmt.Errorf("properties field not found on schema")
	}

	props, ok := schemaProps.(map[string]interface{})
	if !ok {
		props = make(map[string]interface{})
	}

	data := encryptMap(enc, props, credData)
	return data, nil
}
