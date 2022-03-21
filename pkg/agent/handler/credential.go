package handler

import (
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

type encryptFunc func(enc util.Encryptor, schema, data map[string]interface{}) map[string]interface{}

type credentials struct {
	prov        credProv
	client      client
	encrypt     encryptFunc
	encryptFunc newEncryptFunc
}

type newEncryptFunc func(key, alg, hash string) (util.Encryptor, error)

// NewCredentialHandler creates a Handler for Credentials
func NewCredentialHandler(prov credProv, client client) Handler {
	return &credentials{
		prov:        prov,
		client:      client,
		encrypt:     encryptMap,
		encryptFunc: util.NewEncryptor,
	}
}

// Handle processes grpc events triggered for Credentials
func (h *credentials) Handle(action proto.Event_Type, meta *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.CredentialGVK().Kind || h.prov == nil || isNotStatusSubResourceUpdate(action, meta) {
		return nil
	}

	cr := &mv1.Credential{}
	err := cr.FromInstance(resource)
	if err != nil {
		return err
	}

	if ok := isStatusFound(cr.Status); !ok {
		return nil
	}

	if ok := shouldProcessPending(cr.Status.Level, cr.Metadata.State); ok {
		log.Tracef("credential handler - processing resource in pending status")
		return h.onPending(cr)
	}

	// check for deleting state on success status
	if ok := shouldProcessDeleting(cr.Status.Level, cr.Metadata.State, len(cr.Finalizers)); ok {
		log.Tracef("credential handler - processing resource in deleting state")
		h.onDeleting(cr)
	}

	return nil
}

func (h *credentials) onPending(cred *mv1.Credential) error {
	app, err := h.getManagedApp(cred)
	if err != nil {
		return err
	}

	crd, err := h.getCRD(cred)
	if err != nil {
		return err
	}

	provCreds := newProvCreds(cred, util.GetAgentDetails(app))
	status, credentialData := h.prov.CredentialProvision(provCreds)

	sec := app.Spec.Security

	if status.GetStatus() == prov.Success {
		enc, err := h.encryptFunc(sec.EncryptionKey, sec.EncryptionAlgorithm, sec.EncryptionHash)
		if err != nil {
			status = prov.NewRequestStatusBuilder().
				SetMessage(fmt.Sprintf("error encrypting credential: %s", err.Error())).
				Failed()
		}

		if schemaProps, ok := crd.Spec.Provision.Schema["properties"]; ok {
			props, ok := schemaProps.(map[string]interface{})
			if !ok {
				props = make(map[string]interface{})
			}
			cred.Data = h.encrypt(enc, props, credentialData.GetData())
		}
	}

	cred.Status = prov.NewStatusReason(status)

	details := util.MergeMapStringString(util.GetAgentDetailStrings(cred), status.GetProperties())
	util.SetAgentDetails(cred, util.MapStringStringToMapStringInterface(details))

	ri, _ := cred.AsInstance()
	h.client.UpdateResourceFinalizer(ri, crFinalizer, "", true)

	return h.client.CreateSubResourceScoped(
		cred.ResourceMeta,
		map[string]interface{}{
			defs.XAgentDetails: util.GetAgentDetails(cred),
			"status":           cred.Status,
			"data":             cred.Data,
		},
	)
}

func (h *credentials) onDeleting(cred *mv1.Credential) {
	provCreds := newProvCreds(cred, map[string]interface{}{})
	status := h.prov.CredentialDeprovision(provCreds)

	if status.GetStatus() == prov.Success {
		ri, _ := cred.AsInstance()
		h.client.UpdateResourceFinalizer(ri, crFinalizer, "", false)
	}
}

func (h *credentials) getManagedApp(cred *mv1.Credential) (*mv1.ManagedApplication, error) {
	url := fmt.Sprintf(
		"/management/v1alpha1/environments/%s/managedapplications/%s",
		cred.Metadata.Scope.Name,
		cred.Spec.ManagedApplication,
	)
	ri, err := h.client.GetResource(url)
	if err != nil {
		return nil, err
	}
	app := &mv1.ManagedApplication{}
	err = app.FromInstance(ri)
	return app, err
}

func (h *credentials) getCRD(cred *mv1.Credential) (*mv1.CredentialRequestDefinition, error) {
	url := fmt.Sprintf(
		"/management/v1alpha1/environments/%s/credentialrequestdefinitions/%s",
		cred.Metadata.Scope.Name,
		cred.Spec.CredentialRequestDefinition,
	)
	ri, err := h.client.GetResource(url)
	if err != nil {
		return nil, err
	}

	crd := &mv1.CredentialRequestDefinition{}
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
	credDetails map[string]interface{}
	appDetails  map[string]interface{}
}

func newProvCreds(cr *mv1.Credential, appDetails map[string]interface{}) *provCreds {
	credDetails := util.GetAgentDetails(cr)

	return &provCreds{
		appDetails:  appDetails,
		credDetails: credDetails,
		credType:    cr.Spec.CredentialRequestDefinition,
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
