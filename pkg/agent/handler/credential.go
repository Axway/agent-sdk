package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"hash"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const xAxwayEncrypted = "x-axway-encrypted"

type credProv interface {
	CredentialProvision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential)
	CredentialDeprovision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus)
}

type encryptFunc func(enc encryptStr, schema, data map[string]interface{}) map[string]interface{}

type credentials struct {
	prov    credProv
	client  client
	encrypt encryptFunc
}

// NewCredentialHandler creates a Handler for Access Requests
func NewCredentialHandler(prov credProv, client client) Handler {
	return &credentials{
		prov:    prov,
		client:  client,
		encrypt: encryptMap,
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

	ok := isStatusFound(cr.Status)
	if !ok {
		return nil
	}

	if cr.Status.Level != statusPending {
		return nil
	}

	app, err := h.getManagedApp(cr)
	if err != nil {
		return err
	}

	crd, err := h.getCRD(cr)
	if err != nil {
		return err
	}

	creds := newProvCreds(cr, util.GetAgentDetails(app))

	if action == proto.Event_DELETED {
		h.prov.CredentialDeprovision(creds)
		return nil
	}

	var status prov.RequestStatus
	var credentialData prov.Credential

	if cr.Status.Level == statusPending {
		status, credentialData = h.prov.CredentialProvision(creds)
		sec := app.Spec.Security
		enc, err := newEncryptor(sec.EncryptionKey, sec.EncryptionAlgorithm, sec.EncryptionHash)
		if err != nil {
			status = prov.NewRequestStatusBuilder().SetMessage(fmt.Sprintf("error encrypting credential: %s", err.Error())).Failed()
		} else {
			cr.Data = h.encrypt(enc, crd.Spec.Provision.Schema, credentialData.GetData())
		}

		cr.Status = prov.NewStatusReason(status)

		details := util.MergeMapStringInterface(util.GetAgentDetails(cr), status.GetProperties())
		util.SetAgentDetails(cr, details)

		// TODO add finalizer

		err = h.client.CreateSubResourceScoped(
			mv1.EnvironmentResourceName,
			cr.Metadata.Scope.Name,
			cr.PluralName(),
			cr.Name,
			cr.Group,
			cr.APIVersion,
			map[string]interface{}{
				defs.XAgentDetails: util.GetAgentDetails(cr),
				"status":           cr.Status,
				"data":             cr.Data,
			},
		)

		return err
	}

	// check for deleting state on success status
	if cr.Status.Level == statusSuccess && cr.Metadata.State == v1.ResourceDeleting {
		status = h.prov.CredentialDeprovision(creds)

		// TODO remoce finalizer
		_ = status
	}

	return nil
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

	ma := &mv1.ManagedApplication{}
	err = ma.FromInstance(ri)
	return ma, err
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
func encryptMap(enc encryptStr, schema, data map[string]interface{}) map[string]interface{} {
	properties, ok := schema["properties"]
	if !ok {
		return data
	}

	props := properties.(map[string]interface{})
	for key, value := range data {
		schemaValue := props[key]
		v, ok := schemaValue.(map[string]interface{})
		if !ok {
			continue
		}

		if _, ok := v[xAxwayEncrypted]; ok {
			v, ok := value.(string)
			if !ok {
				continue
			}

			str, err := enc.encrypt(v)
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
func (c provCreds) GetCredentialDetailsValue(key string) interface{} {
	if c.credDetails == nil {
		return nil
	}
	return c.credDetails[key]
}

// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (c provCreds) GetApplicationDetailsValue(key string) interface{} {
	if c.appDetails == nil {
		return nil
	}
	return c.appDetails[key]
}

// encryptStr is an interface for encrypting strings
type encryptStr interface {
	encrypt(str string) (string, error)
}

// encryptor implements the encryptStr interface
type encryptor struct {
	alg  string
	key  *rsa.PublicKey
	hash hash.Hash
}

// newEncryptor creates a struct to handle encryption based on the provided key, algorithm, and hash.
func newEncryptor(key, alg, hash string) (*encryptor, error) {
	enc := &encryptor{
		alg: alg,
	}

	pub, err := enc.newPub(key)
	if err != nil {
		return nil, err
	}

	h, err := enc.newHash(hash)
	if err != nil {
		return nil, err
	}

	ok := enc.validateAlg()
	if !ok {
		return nil, fmt.Errorf("unexpected encryption algorithm: %s", alg)
	}

	enc.hash = h
	enc.key = pub
	return enc, nil
}

// encryptStr encrypts a string based on the provided app security
func (e *encryptor) encrypt(str string) (string, error) {
	bts, err := e.encAlgorithm(str)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %s", err)
	}

	return string(bts), nil
}

func (e *encryptor) newPub(key string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, fmt.Errorf("failed to decode public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %s", err)
	}

	p, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected public key type to be *rsa.PublicKey but received %T", pub)
	}

	return p, nil
}

func (e *encryptor) newHash(hash string) (hash.Hash, error) {
	switch hash {
	case "":
		fallthrough
	case "SHA256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unexpected encryption hash: %s", hash)
	}
}

func (e *encryptor) validateAlg() bool {
	switch e.alg {
	case "":
		fallthrough
	case "RSA-OAEP":
		return true
	case "PKCS":
		return true
	default:
		return false
	}
}

func (e *encryptor) encAlgorithm(msg string) ([]byte, error) {
	switch e.alg {
	case "":
		fallthrough
	case "RSA-OAEP":
		return rsa.EncryptOAEP(e.hash, rand.Reader, e.key, []byte(msg), nil)
	case "PKCS":
		return rsa.EncryptPKCS1v15(rand.Reader, e.key, []byte(msg))
	default:
		return nil, fmt.Errorf("unexpected encryption algorithm: %s", e.alg)
	}
}
