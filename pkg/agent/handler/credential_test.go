package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning/mock"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestCredentialHandler(t *testing.T) {
	crdRI, _ := crd.AsInstance()

	tests := []struct {
		action           proto.Event_Type
		createErr        error
		expectedProvType string
		getAppErr        error
		getCrdErr        error
		hasError         bool
		inboundStatus    string
		name             string
		outboundStatus   string
		subError         error
	}{
		{
			action:           proto.Event_CREATED,
			expectedProvType: provision,
			inboundStatus:    statusPending,
			name:             "should handle a create event for a Credential when status is pending",
			outboundStatus:   statusSuccess,
		},
		{
			action:           proto.Event_UPDATED,
			expectedProvType: provision,
			inboundStatus:    statusPending,
			name:             "should handle an update event for a Credential when status is pending",
			outboundStatus:   statusSuccess,
		},
		{
			action: proto.Event_SUBRESOURCEUPDATED,
			name:   "should return nil when the event is for subresources",
		},
		{
			action:        proto.Event_UPDATED,
			inboundStatus: statusErr,
			name:          "should return nil and not process anything when the Credential status is set to Error",
		},
		{
			action:        proto.Event_UPDATED,
			inboundStatus: statusSuccess,
			name:          "should return nil and not process anything when the Credential status is set to Success",
		},
		{
			action:         proto.Event_CREATED,
			getAppErr:      fmt.Errorf("error getting managed app"),
			inboundStatus:  statusPending,
			name:           "should handle an error when retrieving the managed app, and set a failed status",
			outboundStatus: statusErr,
		},
		{
			action:         proto.Event_CREATED,
			getCrdErr:      fmt.Errorf("error getting credential request definition"),
			inboundStatus:  statusPending,
			name:           "should handle an error when retrieving the credential request definition, and set a failed status",
			outboundStatus: statusErr,
		},
		{
			action:           proto.Event_CREATED,
			expectedProvType: provision,
			hasError:         true,
			inboundStatus:    statusPending,
			name:             "should handle an error when updating the Credential subresources",
			outboundStatus:   statusSuccess,
			subError:         fmt.Errorf("error updating subresources"),
		},
		{
			action: proto.Event_CREATED,
			name:   "should return nil and not process anything when the status field is empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cred := credential
			cred.Status.Level = tc.inboundStatus

			p := &mockCredProv{
				t: t,
				expectedStatus: mock.MockRequestStatus{
					Status: prov.Success,
					Msg:    "msg",
					Properties: map[string]string{
						"status_key": "status_val",
					},
				},
				expectedAppDetails:  util.GetAgentDetails(credApp),
				expectedCredDetails: util.GetAgentDetails(&cred),
				expectedManagedApp:  credAppRefName,
				expectedCredType:    cred.Spec.CredentialRequestDefinition,
			}

			c := &credClient{
				t:              t,
				expectedStatus: tc.outboundStatus,
				managedApp:     credApp,
				crd:            crdRI,
				getAppErr:      tc.getAppErr,
				getCrdErr:      tc.getCrdErr,
				createErr:      tc.createErr,
				subError:       tc.subError,
			}

			handler := NewCredentialHandler(p, c)
			v := handler.(*credentials)
			v.encryptSchema = func(_, _ map[string]interface{}, _, _, _ string) (map[string]interface{}, error) {
				return map[string]interface{}{}, nil
			}

			ri, _ := cred.AsInstance()
			err := handler.Handle(tc.action, nil, ri)
			assert.Equal(t, tc.expectedProvType, p.expectedProvType)
			util.AssertError(t, tc.hasError, err)
		})
	}
}

func TestCredentialHandler_deleting(t *testing.T) {
	crdRI, _ := crd.AsInstance()

	tests := []struct {
		name           string
		outboundStatus prov.Status
	}{
		{
			name:           "should deprovision with no error",
			outboundStatus: prov.Success,
		},
		{
			name:           "should fail to deprovision and set the status to error",
			outboundStatus: prov.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cred := credential
			cred.Status.Level = statusSuccess
			cred.Metadata.State = v1.ResourceDeleting
			cred.Finalizers = []v1.Finalizer{{Name: crFinalizer}}

			p := &mockCredProv{
				t: t,
				expectedStatus: mock.MockRequestStatus{
					Status: tc.outboundStatus,
					Msg:    "msg",
					Properties: map[string]string{
						"status_key": "status_val",
					},
				},
				expectedAppDetails:  map[string]interface{}{},
				expectedCredDetails: util.GetAgentDetails(&cred),
				expectedManagedApp:  credAppRefName,
				expectedCredType:    cred.Spec.CredentialRequestDefinition,
			}

			c := &credClient{
				crd:            crdRI,
				expectedStatus: tc.outboundStatus.String(),
				isDeleting:     true,
				managedApp:     credApp,
				t:              t,
			}

			handler := NewCredentialHandler(p, c)
			v := handler.(*credentials)
			v.encryptSchema = func(_, _ map[string]interface{}, _, _, _ string) (map[string]interface{}, error) {
				return map[string]interface{}{}, nil
			}

			ri, _ := cred.AsInstance()
			err := handler.Handle(proto.Event_UPDATED, nil, ri)
			assert.Nil(t, err)
			assert.Equal(t, deprovision, p.expectedProvType)

			if tc.outboundStatus.String() == statusSuccess {
				assert.False(t, c.createSubCalled)
			} else {
				assert.True(t, c.createSubCalled)
			}
		})
	}
}

func TestCredentialHandler_wrong_kind(t *testing.T) {
	c := &mockClient{}
	p := &mockCredProv{}
	handler := NewCredentialHandler(p, c)
	ri := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1.EnvironmentGVK(),
		},
	}
	err := handler.Handle(proto.Event_CREATED, nil, ri)
	assert.Nil(t, err)
}

func Test_creds(t *testing.T) {
	c := provCreds{
		managedApp: "app-name",
		credType:   "api-key",
		credDetails: map[string]interface{}{
			"abc": "123",
		},
		appDetails: map[string]interface{}{
			"def": "456",
		},
	}

	assert.Equal(t, c.managedApp, c.GetApplicationName())
	assert.Equal(t, c.credType, c.GetCredentialType())
	assert.Equal(t, c.credDetails["abc"], c.GetCredentialDetailsValue("abc"))
	assert.Equal(t, c.appDetails["def"], c.GetApplicationDetailsValue("def"))

	c.credDetails = nil
	c.appDetails = nil
	assert.Empty(t, c.GetApplicationDetailsValue("app_details_key"))
	assert.Empty(t, c.GetCredentialDetailsValue("access_details_key"))
}

type mockCredProv struct {
	expectedAppDetails  map[string]interface{}
	expectedCredDetails map[string]interface{}
	expectedCredType    string
	expectedManagedApp  string
	expectedProvType    string
	expectedStatus      mock.MockRequestStatus
	t                   *testing.T
}

func (m *mockCredProv) CredentialProvision(cr prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential) {
	m.expectedProvType = provision
	v := cr.(*provCreds)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedCredDetails, v.credDetails)
	assert.Equal(m.t, m.expectedManagedApp, v.managedApp)
	assert.Equal(m.t, m.expectedCredType, v.credType)
	return m.expectedStatus, &mockProvCredential{}
}

func (m *mockCredProv) CredentialDeprovision(cr prov.CredentialRequest) (status prov.RequestStatus) {
	m.expectedProvType = deprovision
	v := cr.(*provCreds)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedCredDetails, v.credDetails)
	assert.Equal(m.t, m.expectedManagedApp, v.managedApp)
	assert.Equal(m.t, m.expectedCredType, v.credType)
	return m.expectedStatus
}

type mockProvCredential struct{}

func (m *mockProvCredential) GetData() map[string]interface{} {
	return map[string]interface{}{}
}

func decrypt(pk *rsa.PrivateKey, alg string, data map[string]interface{}) map[string]interface{} {
	enc := func(v string) ([]byte, error) {
		switch alg {
		case "RSA-OAEP":
			bts, _ := base64.StdEncoding.DecodeString(v)
			return rsa.DecryptOAEP(sha256.New(), rand.Reader, pk, bts, nil)
		case "PKCS":
			bts, _ := base64.StdEncoding.DecodeString(v)
			return rsa.DecryptPKCS1v15(rand.Reader, pk, bts)
		default:
			return nil, fmt.Errorf("unexpected algorithm")
		}
	}

	for key, value := range data {
		v, ok := value.(string)
		if !ok {
			continue
		}

		bts, err := enc(v)
		if err != nil {
			log.Errorf("Failed to decrypt: %s\n", err)
			continue
		}
		data[key] = string(bts)
	}

	return data
}

func Test_encrypt(t *testing.T) {
	var crdSchema = `{
    "type": "object",
    "$schema": "http://json-schema.org/draft-07/schema#",
    "required": [
        "abc"
    ],
    "properties": {
        "one": {
            "type": "string",
            "description": "abc.",
						"x-axway-encrypted": true
        },
        "two": {
            "type": "string",
            "description": "def."
        },
        "three": {
            "type": "string",
            "description": "ghi.",
						"x-axway-encrypted": true
        }
    },
    "description": "sample."
}`

	crd := map[string]interface{}{}
	err := json.Unmarshal([]byte(crdSchema), &crd)
	assert.Nil(t, err)

	pub, priv, err := newKeyPair()
	assert.Nil(t, err)

	tests := []struct {
		alg           string
		hasErr        bool
		hasEncryptErr bool
		hash          string
		name          string
		publicKey     string
		privateKey    string
	}{
		{
			name:       "should encrypt when the algorithm is PKCS",
			alg:        "PKCS",
			hash:       "SHA256",
			publicKey:  pub,
			privateKey: priv,
		},
		{
			name:       "should encrypt when the algorithm is RSA-OAEP",
			alg:        "RSA-OAEP",
			hash:       "SHA256",
			publicKey:  pub,
			privateKey: priv,
		},
		{
			name:       "should return an error when the algorithm is unknown",
			hasErr:     true,
			alg:        "fake",
			hash:       "SHA256",
			publicKey:  pub,
			privateKey: priv,
		},
		{
			name:       "should return an error when the hash is unknown",
			hasErr:     true,
			alg:        "RSA-OAEP",
			hash:       "fake",
			publicKey:  pub,
			privateKey: priv,
		},
		{
			name:       "should return an error when the public key cannot be parsed",
			hasErr:     true,
			alg:        "RSA-OAEP",
			hash:       "SHA256",
			publicKey:  "fake",
			privateKey: priv,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schemaData := map[string]interface{}{
				"one":   "abc",
				"two":   "def",
				"three": "ghi",
			}

			encrypted, err := encryptSchema(crd, schemaData, tc.publicKey, tc.alg, tc.hash)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.NotEqual(t, "abc", schemaData["one"])
				assert.Equal(t, "def", schemaData["two"])
				assert.NotEqual(t, "ghi", schemaData["three"])

				decrypted := decrypt(parsePrivateKey(tc.privateKey), tc.alg, encrypted)
				assert.Equal(t, "abc", decrypted["one"])
				assert.Equal(t, "def", decrypted["two"])
				assert.Equal(t, "ghi", decrypted["three"])
			}
		})
	}

}

type credClient struct {
	managedApp      *v1.ResourceInstance
	crd             *v1.ResourceInstance
	getAppErr       error
	getCrdErr       error
	createErr       error
	createSubCalled bool
	updateErr       error
	subError        error
	expectedStatus  string
	t               *testing.T
	isDeleting      bool
}

func (m *credClient) GetResource(url string) (*v1.ResourceInstance, error) {
	if strings.Contains(url, "/managedapplications") {
		return m.managedApp, m.getAppErr
	}
	if strings.Contains(url, "/credentialrequestdefinitions") {
		return m.crd, m.getCrdErr
	}

	return nil, fmt.Errorf("mock client - resource not found")
}

func (m *credClient) CreateResource(_ string, _ []byte) (*v1.ResourceInstance, error) {
	return nil, m.createErr
}

func (m *credClient) UpdateResource(_ string, _ []byte) (*v1.ResourceInstance, error) {
	return nil, m.updateErr
}

func (m *credClient) CreateSubResourceScoped(_ v1.ResourceMeta, subs map[string]interface{}) error {
	status := subs["status"].(*v1.ResourceStatus)
	assert.Equal(m.t, m.expectedStatus, status.Level, status.Reasons)
	m.createSubCalled = true
	return m.subError
}

func (m *credClient) UpdateResourceFinalizer(ri *v1.ResourceInstance, _, _ string, addAction bool) (*v1.ResourceInstance, error) {
	if m.isDeleting {
		assert.False(m.t, addAction, "addAction should be false when the resource is deleting")
	} else {
		assert.True(m.t, addAction, "addAction should be true when the resource is not deleting")
	}

	return nil, nil
}

func parsePrivateKey(priv string) *rsa.PrivateKey {
	block, _ := pem.Decode([]byte(priv))
	if block == nil {
		panic("failed to parse PEM block containing the public key")
	}

	pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse private key: " + err.Error())
	}

	return pk
}

func newKeyPair() (public string, private string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	pkBts := x509.MarshalPKCS1PrivateKey(priv)
	fmt.Println(pkBts)
	pvBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pkBts,
	}

	privBuff := bytes.NewBuffer([]byte{})
	err = pem.Encode(privBuff, pvBlock)
	if err != nil {
		return "", "", err
	}

	pubKeyBts, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", err
	}

	pubKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBts,
	}

	pubKeyBuff := bytes.NewBuffer([]byte{})
	err = pem.Encode(pubKeyBuff, pubKeyBlock)
	if err != nil {
		return "", "", err
	}

	return pubKeyBuff.String(), privBuff.String(), nil
}

type mockEncryptor struct {
}

func (m mockEncryptor) Encrypt(str string) (string, error) {
	return "abc", nil
}

const credAppRefName = "managed-app-name"

var credApp = &v1.ResourceInstance{
	ResourceMeta: v1.ResourceMeta{
		Name: credAppRefName,
		SubResources: map[string]interface{}{
			defs.XAgentDetails: map[string]interface{}{
				"sub_managed_app_key": "sub_managed_app_val",
			},
		},
	},
}

var crd = &mv1.CredentialRequestDefinition{
	ResourceMeta: v1.ResourceMeta{
		Name: credAppRefName,
		SubResources: map[string]interface{}{
			defs.XAgentDetails: map[string]interface{}{
				"sub_crd_key": "sub_crd_val",
			},
		},
	},
	Owner:      nil,
	References: mv1.CredentialRequestDefinitionReferences{},
	Spec: mv1.CredentialRequestDefinitionSpec{
		Schema: nil,
		Provision: &mv1.CredentialRequestDefinitionSpecProvision{
			Schema: map[string]interface{}{
				"properties": map[string]interface{}{},
			},
		},
		Capabilities: nil,
		Webhooks:     nil,
	},
}

var credential = mv1.Credential{
	ResourceMeta: v1.ResourceMeta{
		Metadata: v1.Metadata{
			ID: "11",
			Scope: v1.MetadataScope{
				Kind: mv1.EnvironmentGVK().Kind,
				Name: "env-1",
			},
		},
		SubResources: map[string]interface{}{
			defs.XAgentDetails: map[string]interface{}{
				"sub_credential_key": "sub_credential_val",
			},
		},
	},
	Spec: mv1.CredentialSpec{
		CredentialRequestDefinition: "api-key",
		ManagedApplication:          credAppRefName,
		Data:                        nil,
	},
	Status: &v1.ResourceStatus{
		Level: "",
	},
}
