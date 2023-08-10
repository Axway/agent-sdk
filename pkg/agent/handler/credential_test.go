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
	"net/http"
	"strings"
	"testing"
	"time"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning/mock"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestCredentialHandler(t *testing.T) {
	crdRI, _ := crd.AsInstance()

	tests := []struct {
		action           proto.Event_Type
		expectedProvType string
		getAppErr        error
		getCrdErr        error
		hasError         bool
		isRenew          bool
		inboundStatus    string
		inboundState     prov.CredentialAction
		name             string
		outboundStatus   string
		subError         error
		appStatus        string
	}{
		{
			action:           proto.Event_CREATED,
			expectedProvType: provision,
			inboundStatus:    prov.Pending.String(),
			name:             "should handle a create event for a Credential when status is pending",
			outboundStatus:   prov.Success.String(),
		},
		{
			action:           proto.Event_UPDATED,
			expectedProvType: provision,
			inboundStatus:    prov.Pending.String(),
			name:             "should handle an update event for a Credential when status is pending",
			outboundStatus:   prov.Success.String(),
		},
		{
			action:         proto.Event_CREATED,
			inboundStatus:  prov.Pending.String(),
			name:           "should return nil with the appStatus is not success",
			outboundStatus: prov.Error.String(),
			appStatus:      prov.Error.String(),
		},
		{
			action: proto.Event_SUBRESOURCEUPDATED,
			name:   "should return nil when the event is for subresources",
		},
		{
			action:        proto.Event_UPDATED,
			inboundStatus: prov.Error.String(),
			name:          "should return nil and not process anything when the Credential status is set to Error",
		},
		{
			action:        proto.Event_UPDATED,
			inboundStatus: prov.Success.String(),
			name:          "should return nil and not process anything when the Credential status is set to Success",
		},
		{
			action:         proto.Event_CREATED,
			getAppErr:      fmt.Errorf("error getting managed app"),
			inboundStatus:  prov.Pending.String(),
			name:           "should handle an error when retrieving the managed app, and set a failed status",
			outboundStatus: prov.Error.String(),
		},
		{
			action:         proto.Event_CREATED,
			getCrdErr:      fmt.Errorf("error getting credential request definition"),
			inboundStatus:  prov.Pending.String(),
			name:           "should handle an error when retrieving the credential request definition, and set a failed status",
			outboundStatus: prov.Error.String(),
		},
		{
			action:           proto.Event_CREATED,
			expectedProvType: provision,
			hasError:         true,
			inboundStatus:    prov.Pending.String(),
			name:             "should handle an error when updating the Credential subresources",
			outboundStatus:   prov.Success.String(),
			subError:         fmt.Errorf("error updating subresources"),
		},
		{
			action: proto.Event_CREATED,
			name:   "should return nil and not process anything when the status field is empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			credApp.SubResources["status"].(map[string]interface{})["level"] = prov.Success.String()
			if tc.appStatus != "" {
				credApp.SubResources["status"].(map[string]interface{})["level"] = tc.appStatus
			}

			cred := credential
			cred.Status.Level = tc.inboundStatus
			cred.Spec.State.Name = tc.inboundState.String()
			if tc.inboundState.String() == "" {
				cred.Spec.State.Name = apiv1.Active
			}

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
				subError:       tc.subError,
			}

			handler := NewCredentialHandler(p, c, nil)
			v := handler.(*credentials)
			v.encryptSchema = func(_, _ map[string]interface{}, _, _, _ string) (map[string]interface{}, error) {
				return map[string]interface{}{}, nil
			}

			ri, _ := cred.AsInstance()
			err := handler.Handle(NewEventContext(tc.action, nil, ri.Kind, ri.Name), nil, ri)
			assert.Equal(t, tc.expectedProvType, p.expectedProvType)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCredentialHandler_deleting(t *testing.T) {
	crdRI, _ := crd.AsInstance()

	tests := []struct {
		name             string
		outboundStatus   prov.Status
		resourceState    string
		provStatus       string
		expectedProvType string
		specState        string
		specStateReason  string
		skipFinalizers   bool
	}{
		{
			name:             "should deprovision with no error",
			outboundStatus:   prov.Success,
			expectedProvType: deprovision,
			resourceState:    apiv1.ResourceDeleting,
			provStatus:       prov.Success.String(),
		},
		{
			name:             "should deprovision expired with no error and not Deleting",
			expectedProvType: deprovision,
			outboundStatus:   prov.Success,
			provStatus:       prov.Pending.String(),
			specState:        apiv1.Inactive,
			specStateReason:  prov.CredExpDetail,
		},
		{
			name:           "should not deprovision when error and not Deleting",
			outboundStatus: prov.Success,
			provStatus:     prov.Error.String(),
		},
		{
			name:             "should deprovision when and Deleting",
			outboundStatus:   prov.Success,
			provStatus:       prov.Error.String(),
			resourceState:    apiv1.ResourceDeleting,
			expectedProvType: deprovision,
		},
		{
			name:             "should fail to deprovision and set the status to error",
			outboundStatus:   prov.Error,
			expectedProvType: deprovision,
			resourceState:    apiv1.ResourceDeleting,
			provStatus:       prov.Success.String(),
		},
		{
			name:           "should not deprovision with no agent finalizers",
			resourceState:  apiv1.ResourceDeleting,
			provStatus:     prov.Success.String(),
			outboundStatus: prov.Success,
			skipFinalizers: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cred := credential
			cred.Spec.State.Name = tc.specState
			cred.Spec.State.Reason = tc.specStateReason
			cred.Status.Level = tc.provStatus
			cred.Status.Reasons = []apiv1.ResourceStatusReason{
				{
					Type:   tc.provStatus,
					Detail: tc.specStateReason,
				},
			}
			cred.Metadata.State = tc.resourceState
			if !tc.skipFinalizers {
				cred.Finalizers = []apiv1.Finalizer{{Name: crFinalizer}}
			}
			p := &mockCredProv{
				t: t,
				expectedStatus: mock.MockRequestStatus{
					Status: tc.outboundStatus,
					Msg:    "msg",
					Properties: map[string]string{
						"status_key": "status_val",
					},
				},
				expectedAppDetails:  util.GetAgentDetails(mApp),
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

			handler := NewCredentialHandler(p, c, nil)
			v := handler.(*credentials)
			v.encryptSchema = func(_, _ map[string]interface{}, _, _, _ string) (map[string]interface{}, error) {
				return map[string]interface{}{}, nil
			}

			ri, _ := cred.AsInstance()
			err := handler.Handle(NewEventContext(proto.Event_UPDATED, nil, ri.Kind, ri.Name), nil, ri)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedProvType, p.expectedProvType)

			if tc.outboundStatus.String() == prov.Success.String() && tc.specStateReason != prov.CredExpDetail {
				assert.False(t, c.createSubCalled)
			} else {
				assert.True(t, c.createSubCalled)
			}
		})
	}
}

func TestCredentialHandler_update(t *testing.T) {
	crdRI, _ := crd.AsInstance()

	tests := []struct {
		name             string
		isRotating       bool
		inboundStatus    string
		inboundSpecState string
		inboundState     string
		outboundState    string
		expectedProvType string
		outboundStatus   prov.Status
	}{
		{
			name:             "should update credential on rotate",
			isRotating:       true,
			inboundState:     apiv1.Active,
			expectedProvType: update,
		},
		{
			name:             "should update credential on suspend",
			inboundSpecState: apiv1.Active,
			inboundState:     apiv1.Inactive,
			outboundState:    apiv1.Active,
			expectedProvType: update,
		},
		{
			name:             "should update credential on enable",
			inboundSpecState: apiv1.Inactive,
			inboundState:     apiv1.Active,
			outboundState:    apiv1.Inactive,
			expectedProvType: update,
		},
		{
			name:             "should update credential on suspend and rotate",
			inboundSpecState: apiv1.Active,
			inboundState:     apiv1.Inactive,
			outboundState:    apiv1.Active,
			expectedProvType: update,
			isRotating:       true,
		},
		{
			name:             "should update credential on rotate and enable",
			inboundSpecState: apiv1.Inactive,
			inboundState:     apiv1.Active,
			outboundState:    apiv1.Inactive,
			expectedProvType: update,
			isRotating:       true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cred := credential
			cred.Status.Level = tc.inboundStatus
			if tc.inboundStatus == "" {
				cred.Status.Level = prov.Pending.String()
			}
			cred.Metadata.State = tc.inboundStatus
			cred.State.Name = tc.inboundState
			cred.Spec.State.Name = tc.inboundSpecState
			cred.Spec.State.Rotate = tc.isRotating
			cred.Finalizers = []apiv1.Finalizer{{Name: crFinalizer}}

			if tc.outboundStatus.String() == "" {
				tc.outboundStatus = prov.Success
			}

			p := &mockCredProv{
				t:                t,
				expectedProvType: tc.expectedProvType,
				expectedStatus: mock.MockRequestStatus{
					Status: tc.outboundStatus,
					Msg:    "msg",
				},
				expectedAppDetails:  util.GetAgentDetails(credApp),
				expectedCredDetails: util.GetAgentDetails(&cred),
				expectedManagedApp:  credAppRefName,
				expectedCredType:    cred.Spec.CredentialRequestDefinition,
			}

			c := &credClient{
				crd:            crdRI,
				expectedStatus: tc.outboundStatus.String(),
				managedApp:     credApp,
				t:              t,
			}

			handler := NewCredentialHandler(p, c, nil)
			v := handler.(*credentials)
			v.encryptSchema = func(_, _ map[string]interface{}, _, _, _ string) (map[string]interface{}, error) {
				return map[string]interface{}{}, nil
			}

			ri, _ := cred.AsInstance()
			err := handler.Handle(NewEventContext(proto.Event_UPDATED, nil, ri.Kind, ri.Name), nil, ri)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedProvType, p.expectedProvType)
		})
	}
}

func TestCredentialHandler_wrong_kind(t *testing.T) {
	c := &mockClient{}
	p := &mockCredProv{}
	handler := NewCredentialHandler(p, c, nil)
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
		},
	}
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
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
		credData: map[string]interface{}{
			"def": "789",
		},
		id: "cred-id",
		credSchema: map[string]interface{}{
			"properties": "test",
		},
		credProvSchema: map[string]interface{}{
			"properties": "test",
		},
	}

	assert.Equal(t, c.managedApp, c.GetApplicationName())
	assert.Equal(t, c.credType, c.GetCredentialType())
	assert.Equal(t, c.id, c.GetID())
	assert.Equal(t, c.credData, c.GetCredentialData())
	assert.Equal(t, c.credDetails["abc"], c.GetCredentialDetailsValue("abc"))
	assert.Equal(t, c.appDetails["def"], c.GetApplicationDetailsValue("def"))
	assert.Equal(t, c.credSchema, c.GetCredentialSchema())
	assert.Equal(t, c.credProvSchema, c.GetCredentialProvisionSchema())
	assert.Empty(t, c.GetCredentialSchemaDetailsValue("prop"))

	c.credSchemaDetails = map[string]interface{}{
		"detail": "test",
	}
	assert.Equal(t, c.credSchemaDetails["prop"], c.GetCredentialSchemaDetailsValue("prop"))

	c.credDetails = nil
	c.appDetails = nil
	assert.Empty(t, c.GetApplicationDetailsValue("app_details_key"))
	assert.Empty(t, c.GetCredentialDetailsValue("access_details_key"))
	assert.Empty(t, c.GetCredentialSchemaDetailsValue("invalid_key"))
}

func TestIDPCredentialProvisioning(t *testing.T) {
	crdRI, _ := crd.AsInstance()
	s := oauth.NewMockIDPServer()
	defer s.Close()

	tests := []struct {
		name               string
		metadataURL        string
		tokenURL           string
		expectedProvType   string
		outboundStatus     prov.Status
		registrationStatus int
		handlerInvoked     bool
		hasError           bool
	}{
		{
			name:               "should provision IDP credential with no error",
			metadataURL:        s.GetMetadataURL(),
			tokenURL:           s.GetTokenURL(),
			expectedProvType:   provision,
			outboundStatus:     prov.Success,
			registrationStatus: http.StatusCreated,
		},
		{
			name:           "should fail to provision and set the status to error",
			metadataURL:    s.GetMetadataURL(),
			tokenURL:       "test",
			outboundStatus: prov.Error,
			hasError:       true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idpConfig := createIDPConfig(s)
			idpProviderRegistry := oauth.NewProviderRegistry()
			idpProviderRegistry.RegisterProvider(idpConfig, config.NewTLSConfig(), "", 30*time.Second)

			cred := credential
			cred.Status.Level = prov.Pending.String()

			cred.Spec.Data = map[string]interface{}{
				"idpTokenURL": tc.tokenURL,
			}

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
				crd:            crdRI,
				expectedStatus: tc.outboundStatus.String(),
				managedApp:     credApp,
				t:              t,
			}

			handler := NewCredentialHandler(p, c, idpProviderRegistry)
			v := handler.(*credentials)
			v.encryptSchema = func(_, _ map[string]interface{}, _, _, _ string) (map[string]interface{}, error) {
				return map[string]interface{}{}, nil
			}

			ri, _ := cred.AsInstance()
			s.SetRegistrationResponseCode(tc.registrationStatus)
			err := handler.Handle(NewEventContext(proto.Event_UPDATED, nil, ri.Kind, ri.Name), nil, ri)
			assert.Equal(t, tc.expectedProvType, p.expectedProvType)
			assert.Nil(t, err)
		})
	}
}

func TestIDPCredentialDeprovisioning(t *testing.T) {
	crdRI, _ := crd.AsInstance()
	s := oauth.NewMockIDPServer()
	defer s.Close()

	tests := []struct {
		name               string
		metadataURL        string
		tokenURL           string
		outboundStatus     prov.Status
		registrationStatus int
		handlerInvoked     bool
	}{
		{
			name:               "should deprovision IDP credential with no error",
			metadataURL:        s.GetMetadataURL(),
			tokenURL:           s.GetTokenURL(),
			outboundStatus:     prov.Success,
			registrationStatus: http.StatusNoContent,
			handlerInvoked:     true,
		},
		{
			name:           "should fail to deprovision and set the status to error",
			metadataURL:    s.GetMetadataURL(),
			tokenURL:       "test",
			outboundStatus: prov.Error,
			handlerInvoked: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idpConfig := createIDPConfig(s)
			idpProviderRegistry := oauth.NewProviderRegistry()
			idpProviderRegistry.RegisterProvider(idpConfig, config.NewTLSConfig(), "", 30*time.Second)

			cred := credential
			cred.Status.Level = prov.Success.String()
			cred.Metadata.State = apiv1.ResourceDeleting
			cred.Finalizers = []apiv1.Finalizer{{Name: crFinalizer}}
			cred.Spec.Data = map[string]interface{}{
				"idpTokenURL": tc.tokenURL,
			}

			p := &mockCredProv{
				t: t,
				expectedStatus: mock.MockRequestStatus{
					Status: tc.outboundStatus,
					Msg:    "msg",
					Properties: map[string]string{
						"status_key": "status_val",
					},
				},
				expectedAppDetails:  util.GetAgentDetails(mApp),
				expectedCredDetails: util.GetAgentDetails(&cred),
				expectedManagedApp:  credAppRefName,
				expectedCredType:    cred.Spec.CredentialRequestDefinition,
			}

			c := &credClient{
				crd:            crdRI,
				expectedStatus: tc.outboundStatus.String(),
				managedApp:     credApp,
				isDeleting:     true,
				t:              t,
			}

			handler := NewCredentialHandler(p, c, idpProviderRegistry)
			v := handler.(*credentials)
			v.encryptSchema = func(_, _ map[string]interface{}, _, _, _ string) (map[string]interface{}, error) {
				return map[string]interface{}{}, nil
			}

			ri, _ := cred.AsInstance()
			s.SetRegistrationResponseCode(tc.registrationStatus)
			err := handler.Handle(NewEventContext(proto.Event_UPDATED, nil, ri.Kind, ri.Name), nil, ri)
			assert.Nil(t, err)
			if tc.handlerInvoked {
				assert.Equal(t, deprovision, p.expectedProvType)
				if tc.outboundStatus.String() == prov.Success.String() {
					assert.False(t, c.createSubCalled)
				} else {
					assert.True(t, c.createSubCalled)
				}
			} else {
				assert.False(t, c.createSubCalled)
			}
		})
	}

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

func (m *mockCredProv) CredentialUpdate(cr prov.CredentialRequest) (status prov.RequestStatus, credentails prov.Credential) {
	m.expectedProvType = update
	v := cr.(*provCreds)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedCredDetails, v.credDetails)
	assert.Equal(m.t, m.expectedManagedApp, v.managedApp)
	assert.Equal(m.t, m.expectedCredType, v.credType)
	return m.expectedStatus, &mockProvCredential{}
}

type mockProvCredential struct{}

func (m *mockProvCredential) GetData() map[string]interface{} {
	return map[string]interface{}{}
}

func (m *mockProvCredential) GetExpirationTime() time.Time {
	return time.Now()
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
	managedApp      *apiv1.ResourceInstance
	crd             *apiv1.ResourceInstance
	getAppErr       error
	getCrdErr       error
	createSubCalled bool
	subError        error
	expectedStatus  string
	t               *testing.T
	isDeleting      bool
}

func (m *credClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	if strings.Contains(url, "/managedapplications") {
		return m.managedApp, m.getAppErr
	}
	if strings.Contains(url, "/credentialrequestdefinitions") {
		return m.crd, m.getCrdErr
	}

	return nil, fmt.Errorf("mock client - resource not found")
}

func (m *credClient) CreateSubResource(_ apiv1.ResourceMeta, subs map[string]interface{}) error {
	if statusI, ok := subs["status"]; ok {
		status := statusI.(*apiv1.ResourceStatus)
		assert.Equal(m.t, m.expectedStatus, status.Level, status.Reasons)
	}
	m.createSubCalled = true
	return m.subError
}

func (m *credClient) UpdateResourceFinalizer(ri *apiv1.ResourceInstance, _, _ string, addAction bool) (*apiv1.ResourceInstance, error) {
	if m.isDeleting {
		assert.False(m.t, addAction, "addAction should be false when the resource is deleting")
	} else {
		assert.True(m.t, addAction, "addAction should be true when the resource is not deleting")
	}

	return nil, nil
}

func (m *credClient) UpdateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
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

const credAppRefName = "managed-app-name"

var credApp = &apiv1.ResourceInstance{
	ResourceMeta: apiv1.ResourceMeta{
		Name: credAppRefName,
		SubResources: map[string]interface{}{
			defs.XAgentDetails: map[string]interface{}{
				"sub_managed_app_key": "sub_managed_app_val",
			},
			"status": map[string]interface{}{
				"level": prov.Success.String(),
			},
		},
	},
}

var crd = &management.CredentialRequestDefinition{
	ResourceMeta: apiv1.ResourceMeta{
		Name: credAppRefName,
		SubResources: map[string]interface{}{
			defs.XAgentDetails: map[string]interface{}{
				"sub_crd_key": "sub_crd_val",
			},
		},
	},
	Owner:      nil,
	References: management.CredentialRequestDefinitionReferences{},
	Spec: management.CredentialRequestDefinitionSpec{
		Schema: nil,
		Provision: &management.CredentialRequestDefinitionSpecProvision{
			Schema: map[string]interface{}{
				"properties": map[string]interface{}{},
			},
		},
		Webhooks: nil,
	},
}

var credential = management.Credential{
	ResourceMeta: apiv1.ResourceMeta{
		Metadata: apiv1.Metadata{
			ID: "11",
			Scope: apiv1.MetadataScope{
				Kind: management.EnvironmentGVK().Kind,
				Name: "env-1",
			},
		},
		SubResources: map[string]interface{}{
			defs.XAgentDetails: map[string]interface{}{
				"sub_credential_key": "sub_credential_val",
			},
		},
	},
	Spec: management.CredentialSpec{
		CredentialRequestDefinition: "api-key",
		ManagedApplication:          credAppRefName,
		Data:                        nil,
		State: management.CredentialSpecState{
			Name: apiv1.Active,
		},
	},
	Status: &apiv1.ResourceStatus{
		Level: "",
	},
}

func createIDPConfig(s oauth.MockIDPServer) *config.IDPConfiguration {
	return &config.IDPConfiguration{
		Name:        "test",
		Type:        "okta",
		MetadataURL: s.GetMetadataURL(),
		AuthConfig: &config.IDPAuthConfiguration{
			Type:         "client",
			ClientID:     "test",
			ClientSecret: "test",
		},
		GrantType:        "client_credentials",
		ClientScopes:     "read,write",
		AuthMethod:       "client_secret_basic",
		AuthResponseType: "token",
		ExtraProperties:  config.ExtraProperties{"key": "value"},
	}
}
