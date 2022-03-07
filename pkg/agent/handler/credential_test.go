package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	cat "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

// TODO: validate that the right Provision/Deprovision method was called for each test.

func TestCredentialHandler(t *testing.T) {
	managedAppRefName := "managed-app-name"

	mApp := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: managedAppRefName,
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					"sub_managed_app_key": "sub_managed_app_val",
				},
			},
		},
	}

	tests := []struct {
		action    proto.Event_Type
		createErr error
		getErr    error
		hasError  bool
		name      string
		resource  *mv1.Credential
		subError  error
		provType  string
	}{
		{
			name:     "should handle a create event for a Credential when status is pending",
			hasError: false,
			action:   proto.Event_CREATED,
			provType: provision,
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
		{
			name:     "should handle an update event for a Credential when status is pending",
			hasError: false,
			action:   proto.Event_UPDATED,
			provType: provision,
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
		{
			name:     "should deprovision when a delete event is received",
			hasError: false,
			action:   proto.Event_DELETED,
			provType: deprovision,
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
		{
			name:     "should return nil when the Credential status is set to Error",
			hasError: false,
			action:   proto.Event_CREATED,
			provType: "",
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusErr,
				},
			},
		},
		{
			name:     "should return nil when the Credential status is set to Success",
			hasError: false,
			action:   proto.Event_CREATED,
			provType: "",
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusSuccess,
				},
			},
		},
		{
			name:     "should handle an error when retrieving the managed app",
			hasError: true,
			getErr:   fmt.Errorf("error getting managed app"),
			action:   proto.Event_CREATED,
			provType: "",
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
		{
			name:     "should handle an error when updating the Credential subresources",
			hasError: true,
			subError: fmt.Errorf("error updating subresources"),
			action:   proto.Event_CREATED,
			provType: provision,
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
		{
			name:     "should return nil error when the Credential does not have a Status.Level field",
			action:   proto.Event_CREATED,
			hasError: false,
			provType: "",
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: "",
				},
			},
		},
		{
			name:     "should return nil error when status is Success",
			action:   proto.Event_CREATED,
			hasError: false,
			provType: "",
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusSuccess,
				},
			},
		},
		{
			name:     "should return nil error when status is Error",
			action:   proto.Event_CREATED,
			hasError: false,
			provType: "",
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusErr,
				},
			},
		},
		{
			name:     "should return nil when the event is for subresources",
			hasError: false,
			action:   proto.Event_SUBRESOURCEUPDATED,
			provType: "",
			resource: &mv1.Credential{
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
					ManagedApplication:          managedAppRefName,
					Data:                        nil,
				},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &mockCredProv{
				t: t,
				status: mockRequestStatus{
					status: prov.Success,
					msg:    "msg",
					properties: map[string]interface{}{
						"status_key": "status_val",
					},
				},
				expectedAppDetails:  util.GetAgentDetails(mApp),
				expectedCredDetails: util.GetAgentDetails(tc.resource),
				expectedManagedApp:  managedAppRefName,
				expectedCredType:    tc.resource.Spec.CredentialRequestDefinition,
				prov:                tc.provType,
			}
			c := &mockClient{
				getRI:     mApp,
				getErr:    tc.getErr,
				createErr: tc.createErr,
				subError:  tc.subError,
			}
			handler := NewCredentialHandler(p, c)
			v := handler.(*credentials)
			v.encrypt = func(_ cat.ApplicationSpecSecurity, _, data map[string]interface{}) map[string]interface{} {
				return data
			}

			ri, _ := tc.resource.AsInstance()
			err := handler.Handle(tc.action, nil, ri)

			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
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
	c := creds{
		managedApp:  "app-name",
		credType:    "api-key",
		requestType: "Provision",
		credDetails: map[string]interface{}{
			"abc": "123",
		},
		appDetails: map[string]interface{}{
			"def": "456",
		},
	}

	assert.Equal(t, c.managedApp, c.GetApplicationName())
	assert.Equal(t, c.credType, c.GetCredentialType())
	assert.Equal(t, c.requestType, c.GetRequestType())
	assert.Equal(t, c.credDetails["abc"], c.GetCredentialDetailsValue("abc"))
	assert.Equal(t, c.appDetails["def"], c.GetApplicationDetailsValue("def"))
}

type mockCredProv struct {
	t                   *testing.T
	status              mockRequestStatus
	expectedAppDetails  map[string]interface{}
	expectedCredDetails map[string]interface{}
	expectedManagedApp  string
	expectedCredType    string
	prov                string
}

func (m *mockCredProv) CredentialProvision(cr prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential) {
	m.prov = provision
	v := cr.(*creds)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedCredDetails, v.credDetails)
	assert.Equal(m.t, m.expectedManagedApp, v.managedApp)
	assert.Equal(m.t, m.expectedCredType, v.credType)
	return m.status, &mockProvCredential{}
}

func (m *mockCredProv) CredentialDeprovision(cr prov.CredentialRequest) (status prov.RequestStatus) {
	m.prov = deprovision
	v := cr.(*creds)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedCredDetails, v.credDetails)
	assert.Equal(m.t, m.expectedManagedApp, v.managedApp)
	assert.Equal(m.t, m.expectedCredType, v.credType)
	return m.status
}

type mockProvCredential struct {
	data map[string]interface{}
}

func (m *mockProvCredential) GetData() map[string]interface{} {
	return map[string]interface{}{}
}

func newPrivateKey() *rsa.PrivateKey {
	privateKey, err := ioutil.ReadFile("./testdata/private_key.pem")
	if err != nil {
		panic(fmt.Sprintf("failed to read private key file: %s", err))
	}
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		panic("failed to parse PEM block containing the public key")
	}

	pk, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse private key: " + err.Error())
	}

	priv := pk.(*rsa.PrivateKey)
	return priv
}

func decrypt(data map[string]interface{}) map[string]interface{} {
	pk := newPrivateKey()

	for key, value := range data {
		v, ok := value.(string)
		if !ok {
			continue
		}
		txt, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, pk, []byte(v), nil)
		if err != nil {
			log.Errorf("Failed to decrypt: %s\n", err)
			continue
		}
		data[key] = string(txt)
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
						"x-agent-encrypted": "x-agent-encrypted"
        },
        "two": {
            "type": "string",
            "description": "def."
        },
        "three": {
            "type": "string",
            "description": "ghi.",
						"x-agent-encrypted": "x-agent-encrypted"
        }
    },
    "description": "sample."
}`

	publicKey, err := ioutil.ReadFile("./testdata/public_key.pem")
	if err != nil {
		panic(fmt.Sprintf("failed to read public key file: %s", err))
	}
	cApp := cat.Application{
		Spec: cat.ApplicationSpec{Security: cat.ApplicationSpecSecurity{
			EncryptionKey:       string(publicKey),
			EncryptionAlgorithm: "PKCS",
			EncryptionHash:      "SHA256",
		}},
	}

	crd := map[string]interface{}{}
	err = json.Unmarshal([]byte(crdSchema), &crd)
	assert.Nil(t, err)

	schemaData := map[string]interface{}{
		"one":   "abc",
		"two":   "def",
		"three": "ghi",
	}

	props := crd["properties"]
	p := props.(map[string]interface{})
	encrypted := encryptMap(cApp.Spec.Security, p, schemaData)
	assert.NotEqual(t, "abc", schemaData["one"])
	assert.Equal(t, "def", schemaData["two"])
	assert.NotEqual(t, "ghi", schemaData["three"])

	decrypted := decrypt(encrypted)
	assert.Equal(t, "abc", decrypted["one"])
	assert.Equal(t, "def", decrypted["two"])
	assert.Equal(t, "ghi", decrypted["three"])
}
