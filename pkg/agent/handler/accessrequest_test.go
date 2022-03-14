package handler

import (
	"fmt"
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestAccessRequestHandler(t *testing.T) {
	instRefID := "inst-id-1"
	instRefName := "inst-name-1"
	managedAppRefName := "managed-app-name"

	instance := &mv1.APIServiceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: instRefName,
			Metadata: v1.Metadata{
				ID: instRefID,
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID: instRefID,
				},
			},
		},
		Spec: mv1.ApiServiceInstanceSpec{},
	}

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

	accessReq := mv1.AccessRequest{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: "11",
				References: []v1.Reference{
					{
						ID:   instRefID,
						Name: instRefName,
					},
				},
				Scope: v1.MetadataScope{
					Kind: mv1.EnvironmentGVK().Kind,
					Name: "env-1",
				},
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					"sub_access_request_key": "sub_access_request_val",
				},
			},
		},
		References: mv1.AccessRequestReferences{},
		Spec: mv1.AccessRequestSpec{
			ApiServiceInstance: instRefName,
			ManagedApplication: managedAppRefName,
		},
		Status: &v1.ResourceStatus{
			Level: statusPending,
		},
	}

	tests := []struct {
		action     proto.Event_Type
		createErr  error
		getErr     error
		hasError   bool
		name       string
		status     string
		state      string
		subError   error
		provType   string
		references []v1.Reference
	}{
		{
			name:       "should handle a create event for an AccessRequest when status is pending, and state is provision",
			provType:   provision,
			hasError:   false,
			action:     proto.Event_CREATED,
			status:     statusPending,
			state:      provision,
			references: accessReq.Metadata.References,
		},
		{
			name:       "should handle an update event for an AccessRequest when status is pending, and state is provision",
			provType:   provision,
			hasError:   false,
			action:     proto.Event_UPDATED,
			status:     statusPending,
			state:      provision,
			references: accessReq.Metadata.References,
		},
		{
			name:       "should handle an update event for an AccessRequest when status is pending, and state is deprovision",
			provType:   deprovision,
			hasError:   false,
			action:     proto.Event_UPDATED,
			status:     statusPending,
			state:      deprovision,
			references: accessReq.Metadata.References,
		},
		{
			name:       "should deprovision when receiving a delete event",
			provType:   deprovision,
			hasError:   false,
			action:     proto.Event_DELETED,
			state:      deprovision,
			status:     statusPending,
			references: accessReq.Metadata.References,
		},
		{
			name:     "should return nil when the event is for subresources",
			hasError: false,
			action:   proto.Event_SUBRESOURCEUPDATED,
			provType: "",
		},
		{
			name:       "should return nil when status is set to Error",
			provType:   "",
			hasError:   false,
			action:     proto.Event_UPDATED,
			status:     statusErr,
			references: accessReq.Metadata.References,
			state:      provision,
		},
		{
			name:       "should return nil when the status is set to Success",
			provType:   "",
			hasError:   false,
			action:     proto.Event_UPDATED,
			status:     statusSuccess,
			references: accessReq.Metadata.References,
			state:      provision,
		},
		{
			name:       "should return nil when the status field is empty",
			provType:   "",
			action:     proto.Event_CREATED,
			status:     "",
			state:      provision,
			references: accessReq.Metadata.References,
		},
		{
			name:       "should handle an error when retrieving the managed app",
			provType:   "",
			hasError:   true,
			getErr:     fmt.Errorf("error getting managed app"),
			action:     proto.Event_CREATED,
			references: accessReq.Metadata.References,
			status:     statusPending,
			state:      provision,
		},
		{
			name:       "should handle an error when updating the AccessRequest subresources",
			provType:   provision,
			hasError:   true,
			subError:   fmt.Errorf("error updating subresources"),
			action:     proto.Event_CREATED,
			references: accessReq.Metadata.References,
			status:     statusPending,
			state:      provision,
		},
		{
			name:     "should handle an error when the instance is not found in the cache",
			provType: "",
			hasError: true,
			action:   proto.Event_CREATED,
			status:   statusPending,
			state:    provision,
		},
		{
			name:       "should return nil when the AccessRequest does not have a State.Name field",
			provType:   "",
			action:     proto.Event_CREATED,
			status:     statusPending,
			state:      "",
			references: accessReq.Metadata.References,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

			ar := accessReq
			ar.Status.Level = tc.status
			ar.Metadata.References = tc.references

			instanceRI, _ := instance.AsInstance()
			cm.AddAPIServiceInstance(instanceRI)

			c := &mockClient{
				getRI:     mApp,
				getErr:    tc.getErr,
				createErr: tc.createErr,
				subError:  tc.subError,
			}

			arp := &mockARProvision{
				t:                     t,
				expectedAPIID:         instRefID,
				expectedAppName:       managedAppRefName,
				expectedAccessDetails: util.GetAgentDetails(&ar),
				expectedAppDetails:    util.GetAgentDetails(mApp),
				status: mockRequestStatus{
					status: prov.Success,
					msg:    "msg",
					properties: map[string]interface{}{
						"status_key": "status_val",
					},
				},
			}
			handler := NewAccessRequestHandler(arp, cm, c)

			ri, _ := ar.AsInstance()

			err := handler.Handle(tc.action, nil, ri)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestAccessRequestHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{}
	ar := &mockARProvision{}
	handler := NewAccessRequestHandler(ar, cm, c)
	ri := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1.EnvironmentGVK(),
		},
	}
	err := handler.Handle(proto.Event_CREATED, nil, ri)
	assert.Nil(t, err)
}

func Test_arReq(t *testing.T) {
	r := provAccReq{
		apiID: "123",
		appDetails: map[string]interface{}{
			"app_details_key": "app_details_value",
		},
		accessDetails: map[string]interface{}{
			"access_details_key": "access_details_value",
		},
		managedApp: "managed-app-name",
	}

	assert.Equal(t, r.apiID, r.GetAPIID())
	assert.Equal(t, r.managedApp, r.GetApplicationName())
	assert.Equal(t, r.appDetails["app_details_key"], r.GetApplicationDetailsValue("app_details_key"))
	assert.Equal(t, r.accessDetails["access_details_key"], r.GetAccessRequestDetailsValue("access_details_key"))

	r.accessDetails = nil
	r.appDetails = nil
	assert.Nil(t, r.GetApplicationDetailsValue("app_details_key"))
	assert.Nil(t, r.GetAccessRequestDetailsValue("access_details_key"))
}

type mockClient struct {
	getRI     *v1.ResourceInstance
	getErr    error
	createErr error
	updateErr error
	subError  error
}

func (m *mockClient) GetResource(_ string) (*v1.ResourceInstance, error) {
	return m.getRI, m.getErr
}

func (m *mockClient) CreateResource(_ string, _ []byte) (*v1.ResourceInstance, error) {
	return nil, m.createErr
}

func (m *mockClient) UpdateResource(_ string, _ []byte) (*v1.ResourceInstance, error) {
	return nil, m.updateErr
}

func (m *mockClient) CreateSubResourceScoped(_ v1.ResourceMeta, _ map[string]interface{}) error {
	return m.subError
}

func (m *mockClient) UpdateResourceFinalizer(_ *v1.ResourceInstance, _, _ string, _ bool) (*v1.ResourceInstance, error) {
	return nil, nil
}

type mockARProvision struct {
	t                     *testing.T
	expectedAPIID         string
	expectedAppName       string
	expectedAppDetails    map[string]interface{}
	expectedAccessDetails map[string]interface{}
	status                mockRequestStatus
	prov                  string
}

func (m *mockARProvision) AccessRequestProvision(ar prov.AccessRequest) (status prov.RequestStatus) {
	m.prov = provision
	v := ar.(*provAccReq)
	assert.Equal(m.t, m.expectedAPIID, v.apiID)
	assert.Equal(m.t, m.expectedAppName, v.managedApp)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedAccessDetails, v.accessDetails)
	return m.status
}

func (m *mockARProvision) AccessRequestDeprovision(ar prov.AccessRequest) (status prov.RequestStatus) {
	m.prov = deprovision
	v := ar.(*provAccReq)
	assert.Equal(m.t, m.expectedAPIID, v.apiID)
	assert.Equal(m.t, m.expectedAppName, v.managedApp)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedAccessDetails, v.accessDetails)
	return m.status
}

type mockRequestStatus struct {
	status     prov.Status
	msg        string
	properties map[string]interface{}
}

func (m mockRequestStatus) GetStatus() prov.Status {
	return m.status
}

func (m mockRequestStatus) GetMessage() string {
	return m.msg
}

func (m mockRequestStatus) GetProperties() map[string]interface{} {
	return m.properties
}
