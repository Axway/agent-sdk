package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"hash"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	cat "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const xAgentEncrypted = "x-agent-encrypted"

type credProv interface {
	CredentialProvision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential)
	CredentialDeprovision(credentialRequest prov.CredentialRequest) (status prov.RequestStatus)
}

type encryptor func(sec cat.ApplicationSpecSecurity, schema, data map[string]interface{}) map[string]interface{}

type credentials struct {
	prov    credProv
	client  client
	encrypt encryptor
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
func (h *credentials) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != mv1.CredentialGVK().Kind || h.prov == nil || action == proto.Event_SUBRESOURCEUPDATED {
		return nil
	}
	cApp := cat.Application{
		Spec: cat.ApplicationSpec{Security: cat.ApplicationSpecSecurity{
			EncryptionKey:       "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAr0uezHaYsIvhPMYZjSLd\nmMi3GiKTi9e4dGaqZ/xxs7MytlO7eMKTjQ/JQLQcZ3p6JyaeGy2ya4f69Ppcfmgs\n+Iq+vbLrvgZCKiktn8DEB+DTI6uhvfbR9agVyx6MK3NHT8tNMX1no+paZA//G3V9\nT5k9Y0HkC4wOO3OCdUPBF9Q/SaUPy6NJxoFgn/uzu3vUEcF/dlMsJytlo4FvjUsG\nibsfYBsAKyLoEFNFuuQCAuFcmbS0mNw8ULnXYYfXdo/b9OBIEpLmKxsvw/Ov+WtU\n7c+IzOpY0Hbr7O4R+kxiFJNxlV7Cv3Rsw7Y0mNe5qKfgNu9gIixmJuhsOWzRU6U5\n1QIDAQAB\n-----END PUBLIC KEY-----\n",
			EncryptionAlgorithm: "PKCS",
			EncryptionHash:      "SHA256",
		}},
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

	log.Infof("Received a %s event for a AccessRequest", action.String())
	bts, _ := json.MarshalIndent(cr, "", "\t")
	log.Info(string(bts))

	app, err := h.getManagedApp(cr)
	if err != nil {
		return err
	}

	crd, err := h.getCRD(cr)
	if err != nil {
		return err
	}

	credDetails := util.GetAgentDetails(cr)
	appDetails := util.GetAgentDetails(app)

	creds := &creds{
		appDetails:  appDetails,
		credDetails: credDetails,
		credType:    cr.Spec.CredentialRequestDefinition,
		managedApp:  cr.Spec.ManagedApplication,
		requestType: string(cr.Request),
	}

	if action == proto.Event_DELETED {
		log.Info("Deprovisioning the Credentials")
		h.prov.CredentialDeprovision(creds)
		return nil
	}

	var status prov.RequestStatus
	var credentialData prov.Credential

	if cr.Status.Level == statusPending {
		log.Info("Provisioning the Credentials")
		log.Infof("%+v", creds)
		status, credentialData = h.prov.CredentialProvision(creds)
		cr.Data = h.encrypt(cApp.Spec.Security, crd.Spec.Provision.Schema, credentialData.GetData())
	}

	s := prov.NewStatusReason(status)
	cr.Status = &s

	details := util.MergeMapStringInterface(util.GetAgentDetails(cr), status.GetProperties())
	util.SetAgentDetails(cr, details)

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

func (h *credentials) getManagedApp(cred *mv1.Credential) (*v1.ResourceInstance, error) {
	url := fmt.Sprintf(
		"/management/v1alpha1/environments/%s/managedapplications/%s",
		cred.Metadata.Scope.Name,
		cred.Spec.ManagedApplication,
	)
	return h.client.GetResource(url)
}

func (h *credentials) getCRD(cred *mv1.Credential) (*mv1.CredentialRequestDefinition, error) {
	url := fmt.Sprintf(
		"/management/v1alpha1/environments/%s/managedapplications/%s",
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

// encryptMap loops through all provided data and checks the value against the provisioning schema to see if it should be encrypted.
func encryptMap(sec cat.ApplicationSpecSecurity, schema, data map[string]interface{}) map[string]interface{} {
	for key, value := range data {
		schemaValue := schema[key]
		v, ok := schemaValue.(map[string]interface{})
		if !ok {
			continue
		}

		if _, ok := v[xAgentEncrypted]; ok {
			v, ok := value.(string)
			if !ok {
				continue
			}

			str, err := encryptStr(sec.EncryptionKey, sec.EncryptionHash, v)
			if err != nil {
				log.Error(err)
				continue
			}

			data[key] = str
		}
	}

	return data
}

func encryptStr(encKey, encHash, str string) (string, error) {
	block, _ := pem.Decode([]byte(encKey))
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %s", err)
	}

	var h hash.Hash
	switch encHash {
	case "SHA256":
		h = sha256.New()
	default:
		return "", fmt.Errorf("unexpected encryption hash: %s", encHash)
	}

	p := pub.(*rsa.PublicKey)
	bts, err := rsa.EncryptOAEP(h, rand.Reader, p, []byte(str), nil)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %s", err)
	}

	return string(bts), nil
}

type creds struct {
	managedApp  string
	credType    string
	requestType string
	credDetails map[string]interface{}
	appDetails  map[string]interface{}
}

// GetApplicationName gets the name of the managed application
func (c creds) GetApplicationName() string {
	return c.managedApp
}

// GetCredentialType gets the type of the credential
func (c creds) GetCredentialType() string {
	return c.credType
}

// GetRequestType returns the type of request for the credentials
func (c creds) GetRequestType() string {
	return c.requestType
}

// GetCredentialDetailsValue returns a value found on the 'x-agent-details' sub resource of the Credentials.
func (c creds) GetCredentialDetailsValue(key string) interface{} {
	if c.credDetails == nil {
		return nil
	}
	return c.credDetails[key]
}

// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
func (c creds) GetApplicationDetailsValue(key string) interface{} {
	if c.appDetails == nil {
		return nil
	}
	return c.appDetails[key]
}
