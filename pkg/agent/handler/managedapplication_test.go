package handler

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestManagedApplicationHandler(t *testing.T) {
	tests := []struct {
		action    proto.Event_Type
		createErr error
		getErr    error
		hasError  bool
		name      string
		resource  *mv1.ManagedApplication
		subError  error
	}{
		{
			name:     "should handle a create event for a ManagedApplication when status is pending",
			hasError: false,
			action:   proto.Event_CREATED,
			resource: &mv1.ManagedApplication{
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
							"sub_manage_app_key": "sub_manage_app_val",
						},
					},
				},
				Spec: mv1.ManagedApplicationSpec{},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
		{
			name:     "should handle an update event for a ManagedApplication when status is pending",
			hasError: false,
			action:   proto.Event_UPDATED,
			resource: &mv1.ManagedApplication{
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
							"sub_manage_app_key": "sub_manage_app_val",
						},
					},
				},
				Spec: mv1.ManagedApplicationSpec{},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
		{
			name:     "should deprovision when a delete event is received",
			hasError: false,
			action:   proto.Event_DELETED,
			resource: &mv1.ManagedApplication{
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
							"sub_manage_app_key": "sub_manage_app_val",
						},
					},
				},
				Spec: mv1.ManagedApplicationSpec{},
				Status: &v1.ResourceStatus{
					Level: statusPending,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &mockManagedAppProv{
				t: t,
				status: mockRequestStatus{
					status: prov.Success,
					msg:    "msg",
					properties: map[string]interface{}{
						"status_key": "status_val",
					},
				},
				expectedManagedApp:     tc.resource.Name,
				expectedManagedAppData: util.GetAgentDetails(tc.resource),
			}
			c := &mockClient{
				subError: tc.subError,
			}
			handler := NewManagedApplicationHandler(p, c)

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

type mockManagedAppProv struct {
	t                      *testing.T
	status                 mockRequestStatus
	expectedManagedApp     string
	expectedManagedAppData map[string]interface{}
}

func (m *mockManagedAppProv) ApplicationRequestProvision(ma prov.ApplicationRequest) (status prov.RequestStatus) {
	v := ma.(managedApp)
	assert.Equal(m.t, m.expectedManagedApp, v.managedAppName)
	assert.Equal(m.t, m.expectedManagedAppData, v.data)
	return m.status
}

func (m *mockManagedAppProv) ApplicationRequestDeprovision(ma prov.ApplicationRequest) (status prov.RequestStatus) {
	v := ma.(managedApp)
	assert.Equal(m.t, m.expectedManagedApp, v.managedAppName)
	assert.Equal(m.t, m.expectedManagedAppData, v.data)
	return m.status
}
