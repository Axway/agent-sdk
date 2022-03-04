package handler

import (
	"fmt"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
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
