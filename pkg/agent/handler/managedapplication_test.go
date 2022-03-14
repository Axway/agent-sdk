package handler

import (
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

func TestManagedApplicationHandler(t *testing.T) {
	teamName := "team-a"
	team := &defs.PlatformTeam{
		ID:      "1122",
		Name:    teamName,
		Default: true,
	}

	managedApp := mv1.ManagedApplication{
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
		Owner: &v1.Owner{
			Type: 0,
			ID:   team.ID,
		},
		Spec: mv1.ManagedApplicationSpec{},
		Status: &v1.ResourceStatus{
			Level: statusPending,
		},
	}

	tests := []struct {
		action    proto.Event_Type
		createErr error
		getErr    error
		hasError  bool
		name      string
		subError  error
		teamName  string
		provType  string
		status    string
	}{
		{
			name:     "should handle a create event for a ManagedApplication when status is pending",
			hasError: false,
			action:   proto.Event_CREATED,
			teamName: teamName,
			provType: provision,
			status:   statusPending,
		},
		{
			name:     "should handle an update event for a ManagedApplication when status is pending",
			hasError: false,
			action:   proto.Event_UPDATED,
			provType: provision,
			status:   statusPending,
		},
		{
			name:     "should return nil when the event is for subresources",
			hasError: false,
			action:   proto.Event_SUBRESOURCEUPDATED,
			provType: "",
		},
		// TODO - update to deprovision when metadata state is DELETING
		// {
		// 	name:     "should deprovision when a delete event is received",
		// 	hasError: false,
		// 	provType: deprovision,
		// 	action:   proto.Event_DELETED,
		// 	status:   statusPending,
		// },
		{
			name:     "should return nil when status field is empty",
			action:   proto.Event_CREATED,
			provType: "",
			status:   "",
		},
		{
			name:     "should return nil when status field is Success",
			action:   proto.Event_CREATED,
			provType: "",
			status:   statusSuccess,
		},
		{
			name:     "should return nil when status field is Error",
			action:   proto.Event_CREATED,
			provType: "",
			status:   statusErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := managedApp
			app.Status.Level = tc.status

			p := &mockManagedAppProv{
				t: t,
				status: mockRequestStatus{
					status: prov.Success,
					msg:    "msg",
					properties: map[string]interface{}{
						"status_key": "status_val",
					},
				},
				expectedManagedApp:     app.Name,
				expectedTeamName:       tc.teamName,
				expectedManagedAppData: util.GetAgentDetails(&app),
			}
			c := &mockClient{
				subError: tc.subError,
			}

			cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
			if tc.teamName != "" {
				cm.AddTeam(team)
			}
			handler := NewManagedApplicationHandler(p, cm, c)

			ri, _ := app.AsInstance()
			err := handler.Handle(tc.action, nil, ri)
			assert.Equal(t, tc.provType, p.prov)
			if tc.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestManagedApplicationHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{}
	p := &mockManagedAppProv{}
	handler := NewManagedApplicationHandler(p, cm, c)
	ri := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1.EnvironmentGVK(),
		},
	}
	err := handler.Handle(proto.Event_CREATED, nil, ri)
	assert.Nil(t, err)
}

func Test_managedApp(t *testing.T) {
	m := provManagedApp{
		managedAppName: "managed-app-name",
		teamName:       "123",
		data:           map[string]interface{}{"abc": "123"},
	}

	assert.Equal(t, m.managedAppName, m.GetManagedApplicationName())
	assert.Equal(t, m.teamName, m.GetTeamName())
	assert.Equal(t, m.data["abc"].(string), m.GetApplicationDetailsValue("abc"))
}

type mockManagedAppProv struct {
	t                      *testing.T
	status                 mockRequestStatus
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
