package handler

import (
	"fmt"
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning/mock"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestAccessRequestHandler(t *testing.T) {
	tests := []struct {
		action           proto.Event_Type
		createErr        error
		expectedProvType string
		getErr           error
		hasError         bool
		inboundStatus    string
		name             string
		outboundStatus   string
		references       []v1.Reference
		subError         error
		appStatus        string
	}{
		{
			action:           proto.Event_CREATED,
			inboundStatus:    prov.Pending.String(),
			name:             "should handle a create event for an AccessRequest when status is pending",
			outboundStatus:   prov.Success.String(),
			expectedProvType: provision,
			references:       accessReq.Metadata.References,
		},
		{
			action:           proto.Event_UPDATED,
			inboundStatus:    prov.Pending.String(),
			name:             "should handle an update event for an AccessRequest when status is pending",
			outboundStatus:   prov.Success.String(),
			expectedProvType: provision,
			references:       accessReq.Metadata.References,
		},
		{
			action:         proto.Event_CREATED,
			inboundStatus:  prov.Pending.String(),
			name:           "should return nil with the appStatus is not success",
			outboundStatus: prov.Error.String(),
			references:     accessReq.Metadata.References,
			appStatus:      prov.Error.String(),
		},
		{
			action: proto.Event_SUBRESOURCEUPDATED,
			name:   "should return nil when the event is for subresources",
		},
		{
			action:        proto.Event_UPDATED,
			inboundStatus: prov.Error.String(),
			name:          "should return nil and not process anything when status is set to Error",
			references:    accessReq.Metadata.References,
		},
		{
			action:        proto.Event_UPDATED,
			inboundStatus: prov.Success.String(),
			name:          "should return nil and not process anything when the status is set to Success",
			references:    accessReq.Metadata.References,
		},
		{
			action:        proto.Event_CREATED,
			inboundStatus: "",
			name:          "should return nil and not process anything when the status field is empty",
			references:    accessReq.Metadata.References,
		},
		{
			action:         proto.Event_CREATED,
			getErr:         fmt.Errorf("error getting managed app"),
			inboundStatus:  prov.Pending.String(),
			name:           "should handle an error when retrieving the managed app, and set a failed status",
			outboundStatus: prov.Error.String(),
			references:     accessReq.Metadata.References,
		},
		{
			action:           proto.Event_CREATED,
			hasError:         true,
			inboundStatus:    prov.Pending.String(),
			name:             "should handle an error when updating the AccessRequest subresources",
			outboundStatus:   prov.Success.String(),
			expectedProvType: provision,
			references:       accessReq.Metadata.References,
			subError:         fmt.Errorf("error updating subresources"),
		},
		{
			action:         proto.Event_CREATED,
			inboundStatus:  prov.Pending.String(),
			name:           "should handle an error when the instance is not found in the cache, and set a failed status",
			outboundStatus: prov.Error.String(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mApp.SubResources["status"].(map[string]interface{})["level"] = prov.Success.String()
			if tc.appStatus != "" {
				mApp.SubResources["status"].(map[string]interface{})["level"] = tc.appStatus
			}

			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

			ar := accessReq
			ar.Status.Level = tc.inboundStatus
			ar.Metadata.References = tc.references

			instanceRI, _ := instance.AsInstance()
			cm.AddAPIServiceInstance(instanceRI)

			status := mock.MockRequestStatus{
				Status: prov.Success,
				Msg:    "msg",
				Properties: map[string]string{
					"status_key": "status_val",
				},
			}

			arp := &mockARProvision{
				expectedAccessDetails: util.GetAgentDetails(&ar),
				expectedAPIID:         instRefID,
				expectedAppDetails:    util.GetAgentDetails(mApp),
				expectedAppName:       managedAppRefName,
				expectedStatus:        status,
				t:                     t,
			}

			c := &mockClient{
				createErr:      tc.createErr,
				expectedStatus: tc.outboundStatus,
				getErr:         tc.getErr,
				getRI:          mApp,
				subError:       tc.subError,
				t:              t,
			}

			handler := NewAccessRequestHandler(arp, cm, c)

			ri, _ := ar.AsInstance()
			err := handler.Handle(NewEventContext(tc.action, nil, ri.Kind, ri.Name), nil, ri)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.expectedProvType, arp.expectedProvType)
			if tc.inboundStatus == prov.Pending.String() {
				assert.True(t, c.createSubCalled)
			} else {
				assert.False(t, c.createSubCalled)
			}

		})
	}
}

func TestAccessRequestHandler_deleting(t *testing.T) {
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
			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
			ar := accessReq
			ar.Status.Level = prov.Success.String()
			ar.Metadata.State = v1.ResourceDeleting
			ar.Finalizers = []v1.Finalizer{{Name: arFinalizer}}

			instanceRI, _ := instance.AsInstance()
			cm.AddAPIServiceInstance(instanceRI)

			arp := &mockARProvision{
				t:                     t,
				expectedAPIID:         instRefID,
				expectedAppName:       managedAppRefName,
				expectedAccessDetails: util.GetAgentDetails(&ar),
				expectedAppDetails:    map[string]interface{}{},
				expectedStatus: mock.MockRequestStatus{
					Status: tc.outboundStatus,
					Msg:    "msg",
					Properties: map[string]string{
						"status_key": "status_val",
					},
				},
			}

			c := &mockClient{
				expectedStatus: tc.outboundStatus.String(),
				getRI:          mApp,
				isDeleting:     true,
				t:              t,
			}

			handler := NewAccessRequestHandler(arp, cm, c)

			ri, _ := ar.AsInstance()

			err := handler.Handle(NewEventContext(proto.Event_UPDATED, nil, ri.Kind, ri.Name), nil, ri)
			assert.Nil(t, err)
			assert.Equal(t, deprovision, arp.expectedProvType)

			if tc.outboundStatus.String() == prov.Success.String() {
				assert.False(t, c.createSubCalled)
			} else {
				assert.True(t, c.createSubCalled)
			}
		})
	}
}

func TestAccessRequestHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{
		t: t,
	}
	ar := &mockARProvision{}
	handler := NewAccessRequestHandler(ar, cm, c)
	ri := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1.EnvironmentGVK(),
		},
	}
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
}

func Test_arReq(t *testing.T) {
	r := provAccReq{
		appDetails: map[string]interface{}{
			"app_details_key": "app_details_value",
		},
		accessDetails: map[string]interface{}{
			"access_details_key": "access_details_value",
		},
		accessData: map[string]interface{}{
			"key": "val",
		},
		managedApp: "managed-app-name",
		instanceDetails: map[string]interface{}{
			defs.AttrExternalAPIStage: "api-stage",
			defs.AttrExternalAPIID:    "123",
		},
	}

	assert.Equal(t, r.managedApp, r.GetApplicationName())
	assert.Equal(t, r.appDetails["app_details_key"], r.GetApplicationDetailsValue("app_details_key"))
	assert.Equal(t, r.accessDetails["access_details_key"], r.GetAccessRequestDetailsValue("access_details_key"))
	assert.Equal(t, r.accessData, r.GetAccessRequestData())

	r.accessDetails = nil
	r.appDetails = nil
	assert.Empty(t, r.GetApplicationDetailsValue("app_details_key"))
	assert.Empty(t, r.GetAccessRequestDetailsValue("access_details_key"))
}

type mockClient struct {
	createErr       error
	createSubCalled bool
	expectedStatus  string
	getErr          error
	getRI           *v1.ResourceInstance
	isDeleting      bool
	subError        error
	t               *testing.T
	updateErr       error
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

func (m *mockClient) CreateSubResourceScoped(_ v1.ResourceMeta, subs map[string]interface{}) error {
	if statusI, ok := subs["status"]; ok {
		status := statusI.(*v1.ResourceStatus)
		assert.Equal(m.t, m.expectedStatus, status.Level, status.Reasons)
	}
	m.createSubCalled = true
	return m.subError
}

func (m *mockClient) UpdateResourceFinalizer(_ *v1.ResourceInstance, _, _ string, addAction bool) (*v1.ResourceInstance, error) {
	if m.isDeleting {
		assert.False(m.t, addAction, "addAction should be false when the resource is deleting")
	} else {
		assert.True(m.t, addAction, "addAction should be true when the resource is not deleting")
	}

	return nil, nil
}

type mockARProvision struct {
	expectedAccessDetails map[string]interface{}
	expectedAPIID         string
	expectedAppDetails    map[string]interface{}
	expectedAppName       string
	expectedProvType      string
	expectedStatus        mock.MockRequestStatus
	t                     *testing.T
}

func (m *mockARProvision) AccessRequestProvision(ar prov.AccessRequest) (status prov.RequestStatus) {
	m.expectedProvType = provision
	v := ar.(*provAccReq)
	assert.Equal(m.t, m.expectedAPIID, v.instanceDetails[defs.AttrExternalAPIID])
	assert.Equal(m.t, m.expectedAppName, v.managedApp)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedAccessDetails, v.accessDetails)
	return m.expectedStatus
}

func (m *mockARProvision) AccessRequestDeprovision(ar prov.AccessRequest) (status prov.RequestStatus) {
	m.expectedProvType = deprovision
	v := ar.(*provAccReq)
	assert.Equal(m.t, m.expectedAPIID, v.instanceDetails[defs.AttrExternalAPIID])
	assert.Equal(m.t, m.expectedAppName, v.managedApp)
	assert.Equal(m.t, m.expectedAppDetails, v.appDetails)
	assert.Equal(m.t, m.expectedAccessDetails, v.accessDetails)
	return m.expectedStatus
}

const instRefID = "inst-id-1"
const instRefName = "inst-name-1"
const managedAppRefName = "managed-app-name"

var instance = &mv1.APIServiceInstance{
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

var mApp = &v1.ResourceInstance{
	ResourceMeta: v1.ResourceMeta{
		Name: managedAppRefName,
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

var accessReq = mv1.AccessRequest{
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
	Spec: mv1.AccessRequestSpec{
		ApiServiceInstance: instRefName,
		ManagedApplication: managedAppRefName,
	},
	Status: &v1.ResourceStatus{
		Level: prov.Pending.String(),
	},
}
