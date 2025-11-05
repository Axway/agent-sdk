package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning/mock"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestManagedApplicationHandler(t *testing.T) {
	tests := []struct {
		action           proto.Event_Type
		createErr        error
		getErr           error
		hasError         bool
		name             string
		expectedProvType string
		inboundStatus    string
		subError         error
		teamName         string
		outboundStatus   string
		retryCount       int
		expectedStatus   *mock.MockRequestStatus
	}{
		{
			name:             "should handle a create event for a ManagedApplication when status is pending",
			action:           proto.Event_CREATED,
			teamName:         teamName,
			expectedProvType: provision,
			inboundStatus:    prov.Pending.String(),
			outboundStatus:   prov.Success.String(),
		},
		{
			name:             "should handle an update event for a ManagedApplication when status is pending",
			action:           proto.Event_UPDATED,
			expectedProvType: provision,
			inboundStatus:    prov.Pending.String(),
			outboundStatus:   prov.Success.String(),
		},
		{
			name:   "should return nil when the event is for subresources",
			action: proto.Event_SUBRESOURCEUPDATED,
		},
		{
			name:   "should return nil when status field is empty",
			action: proto.Event_CREATED,
		},
		{
			name:          "should return nil when status field is Success",
			action:        proto.Event_CREATED,
			inboundStatus: prov.Success.String(),
		},
		{
			name:          "should return nil when status field is Error",
			action:        proto.Event_CREATED,
			inboundStatus: prov.Error.String(),
		},
		{
			action:           proto.Event_CREATED,
			inboundStatus:    prov.Pending.String(),
			name:             "handle ManagedApp, onPending, fails once, retry triggered, failed after",
			outboundStatus:   prov.Error.String(),
			retryCount:       1,
			expectedProvType: provision,
			expectedStatus: &mock.MockRequestStatus{
				Status: prov.Error,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := managedAppForTest
			app.Status.Level = tc.inboundStatus

			status := mock.MockRequestStatus{
				Status: prov.Success,
				Msg:    "msg",
				Properties: map[string]string{
					"status_key": "status_val",
				},
			}

			p := &mockManagedAppProv{
				expectedManagedApp:     app.Name,
				expectedManagedAppData: util.GetAgentDetails(&app),
				expectedTeamName:       tc.teamName,
				status:                 status,
				t:                      t,
			}
			if tc.expectedStatus != nil {
				p.status = *tc.expectedStatus
			}

			c := &mockClient{
				subError:       tc.subError,
				expectedStatus: tc.outboundStatus,
				t:              t,
			}

			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
			if tc.teamName != "" {
				cm.AddTeam(team)
			}
			handler := NewManagedApplicationHandler(p, cm, c, WithManagedAppRetryCount(tc.retryCount))

			ri, _ := app.AsInstance()
			err := handler.Handle(NewEventContext(tc.action, nil, ri.Kind, ri.Name), nil, ri)

			assert.Equal(t, tc.expectedProvType, p.prov)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}

			if tc.inboundStatus == prov.Pending.String() {
				assert.True(t, c.createSubCalled)
			} else {
				assert.False(t, c.createSubCalled)
			}
		})
	}
}

func TestManagedApplicationHandler_deleting(t *testing.T) {
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
			app := managedAppForTest
			app.Status.Level = prov.Success.String()
			app.Metadata.State = apiv1.ResourceDeleting
			app.Finalizers = []apiv1.Finalizer{{Name: maFinalizer}}

			status := mock.MockRequestStatus{
				Status: tc.outboundStatus,
				Msg:    "msg",
				Properties: map[string]string{
					"status_key": "status_val",
				},
			}

			p := &mockManagedAppProv{
				expectedManagedApp:     app.Name,
				expectedManagedAppData: util.GetAgentDetails(&app),
				expectedTeamName:       "",
				status:                 status,
				t:                      t,
			}

			c := &mockClient{
				expectedStatus: tc.outboundStatus.String(),
				isDeleting:     true,
				t:              t,
			}

			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

			handler := NewManagedApplicationHandler(p, cm, c)

			ri, _ := app.AsInstance()
			err := handler.Handle(NewEventContext(proto.Event_UPDATED, nil, ri.Kind, ri.Name), nil, ri)

			assert.Equal(t, deprovision, p.prov)
			assert.Nil(t, err)

			if tc.outboundStatus.String() == prov.Success.String() {
				assert.False(t, c.createSubCalled)
			} else {
				assert.True(t, c.createSubCalled)
			}
		})
	}
}

func TestManagedApplicationHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{
		t: t,
	}
	p := &mockManagedAppProv{}
	handler := NewManagedApplicationHandler(p, cm, c)
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.EnvironmentGVK(),
		},
	}
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
}

func Test_managedApp(t *testing.T) {
	m := provManagedApp{
		managedAppName: "managed-app-name",
		teamName:       "123",
		data:           map[string]interface{}{"abc": "123"},
		id:             "app-id",
	}

	assert.Equal(t, m.managedAppName, m.GetManagedApplicationName())
	assert.Equal(t, m.teamName, m.GetTeamName())
	assert.Equal(t, m.id, m.GetID())
	assert.Equal(t, m.data["abc"].(string), m.GetApplicationDetailsValue("abc"))

	m.data = nil
	assert.Empty(t, m.GetApplicationDetailsValue("abc"))
}

type mockManagedAppProv struct {
	t                      *testing.T
	status                 mock.MockRequestStatus
	expectedManagedApp     string
	expectedManagedAppData map[string]interface{}
	expectedTeamName       string
	prov                   string
}

func (m *mockManagedAppProv) ApplicationRequestProvision(ma prov.ApplicationRequest) (status prov.RequestStatus) {
	m.prov = provision
	v := ma.(provManagedApp)
	assert.Equal(m.t, m.expectedManagedApp, v.managedAppName)
	assert.Equal(m.t, m.expectedManagedAppData, v.data)
	assert.Equal(m.t, m.expectedTeamName, v.teamName)
	return m.status
}

func (m *mockManagedAppProv) ApplicationRequestDeprovision(ma prov.ApplicationRequest) (status prov.RequestStatus) {
	m.prov = deprovision
	v := ma.(provManagedApp)
	assert.Equal(m.t, m.expectedManagedApp, v.managedAppName)
	assert.Equal(m.t, m.expectedManagedAppData, v.data)
	assert.Equal(m.t, m.expectedTeamName, v.teamName)
	return m.status
}

const teamName = "team-a"

var team = &defs.PlatformTeam{
	ID:      "1122",
	Name:    teamName,
	Default: true,
}

var managedAppForTest = management.ManagedApplication{
	ResourceMeta: apiv1.ResourceMeta{
		Name: "app-test",
		Metadata: apiv1.Metadata{
			ID: "11",
			Scope: apiv1.MetadataScope{
				Kind: management.EnvironmentGVK().Kind,
				Name: "env-1",
			},
		},
		SubResources: map[string]interface{}{
			defs.XAgentDetails: map[string]interface{}{
				"sub_manage_app_key": "sub_manage_app_val",
			},
		},
	},
	Owner: &apiv1.Owner{
		Type: 0,
		ID:   team.ID,
	},
	Spec: management.ManagedApplicationSpec{},
	Status: &apiv1.ResourceStatus{
		Level: prov.Pending.String(),
	},
}
