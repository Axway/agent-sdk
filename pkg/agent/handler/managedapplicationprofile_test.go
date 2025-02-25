package handler

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	v1alpha1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning/mock"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

type mockCache struct {
	team    *defs.PlatformTeam
	profDef *apiv1.ResourceInstance
}

func (m mockCache) GetTeamByID(id string) *defs.PlatformTeam {
	return m.team
}

func (m mockCache) GetApplicationProfileDefinitionByName(name string) (*apiv1.ResourceInstance, error) {
	return m.profDef, nil
}

func TestManagedApplicationProfileHandler(t *testing.T) {
	//setup test cache
	apd, _ := prov.NewApplicationProfileBuilder(
		func(a *v1alpha1.ApplicationProfileDefinition) (*v1alpha1.ApplicationProfileDefinition, error) {
			return a, nil
		}).
		SetName("APD").
		SetTitle("APD Title").
		SetRequestSchema(prov.NewSchemaBuilder().
			AddProperty(prov.NewSchemaPropertyBuilder().
				SetName("enumMapData").
				IsString().
				AddEnumValueMap(map[string]interface{}{
					"label1": "val1",
					"label2": "val2",
					"label3": "val3",
					"label4": "val4",
				}),
			),
		).
		Register()
	apdRI, _ := apd.AsInstance()

	manAppRI, _ := managedAppForTest.AsInstance()

	tests := map[string]struct {
		skip             bool
		action           proto.Event_Type
		createErr        error
		getErr           error
		hasError         bool
		expectedProvType string
		inboundStatus    string
		subError         error
		teamName         string
		outboundStatus   string
	}{
		"should handle a create event for a ManagedApplicationProfile when status is pending": {
			skip:             false,
			action:           proto.Event_CREATED,
			teamName:         teamName,
			expectedProvType: provision,
			inboundStatus:    prov.Pending.String(),
			outboundStatus:   prov.Success.String(),
		},
		"should return nil when the event type is for updating": {
			skip:   false,
			action: proto.Event_UPDATED,
		},
		"should return nil when the event is for subresources": {
			skip:   false,
			action: proto.Event_SUBRESOURCEUPDATED,
		},
		"should return nil when status field is empty": {
			skip:   false,
			action: proto.Event_CREATED,
		},
		"should return nil when status field is Success": {
			skip:          false,
			action:        proto.Event_CREATED,
			inboundStatus: prov.Success.String(),
		},
		"should return nil when status field is Error": {
			skip:          false,
			action:        proto.Event_CREATED,
			inboundStatus: prov.Error.String(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				return
			}
			profile := managedAppForProfTest
			profile.Status.Level = tc.inboundStatus

			status := mock.MockRequestStatus{
				Status: prov.Success,
				Msg:    "msg",
				Properties: map[string]string{
					"status_key": "status_val",
				},
			}

			p := &mockManagedAppProfProv{
				expectedManagedApp:     profile.Spec.ManagedApplication,
				expectedManagedAppData: util.GetAgentDetails(&managedAppForTest),
				expectedTeamName:       tc.teamName,
				status:                 status,
				t:                      t,
			}

			c := &mockClient{
				subError:       tc.subError,
				expectedStatus: tc.outboundStatus,
				t:              t,
				manApp:         manAppRI,
			}

			testCache := mockCache{
				team:    team,
				profDef: apdRI,
			}

			handler := NewManagedApplicationProfileHandler(p, testCache, c)

			ri, _ := profile.AsInstance()
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

func TestManagedApplicationProfileHandler_wrong_kind(t *testing.T) {
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	c := &mockClient{
		t: t,
	}
	p := &mockManagedAppProv{}
	handler := NewManagedApplicationHandler(p, cm, c)
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: v1alpha1.EnvironmentGVK(),
		},
	}
	err := handler.Handle(NewEventContext(proto.Event_CREATED, nil, ri.Kind, ri.Name), nil, ri)
	assert.Nil(t, err)
}

func Test_managedAppProf(t *testing.T) {
	m := provManagedAppProfile{
		managedAppName:    "managed-app-name",
		teamName:          "123",
		data:              map[string]interface{}{"abc": "123"},
		attributes:        map[string]interface{}{"abc": "123"},
		profileDefinition: "prof-def",
		id:                "app-id",
		consumerOrgID:     "org-id",
	}

	assert.Equal(t, m.managedAppName, m.GetManagedApplicationName())
	assert.Equal(t, m.teamName, m.GetTeamName())
	assert.Equal(t, m.id, m.GetID())
	assert.Equal(t, m.data["abc"].(string), m.GetApplicationDetailsValue("abc"))
	assert.Equal(t, m.profileDefinition, m.GetApplicationProfileDefinitionName())
	assert.Equal(t, m.attributes, m.GetApplicationProfileData())
	assert.Equal(t, m.consumerOrgID, m.GetConsumerOrgID())

	m.data = nil
	assert.Empty(t, m.GetApplicationDetailsValue("abc"))
}

type mockManagedAppProfProv struct {
	t                      *testing.T
	status                 mock.MockRequestStatus
	expectedManagedApp     string
	expectedManagedAppData map[string]interface{}
	expectedTeamName       string
	prov                   string
}

func (m *mockManagedAppProfProv) ApplicationProfileRequestProvision(profile prov.ApplicationProfileRequest) (status prov.RequestStatus) {
	m.prov = provision
	v := profile.(provManagedAppProfile)
	assert.Equal(m.t, m.expectedManagedApp, v.managedAppName)
	assert.Equal(m.t, m.expectedManagedAppData, v.data)
	assert.Equal(m.t, m.expectedTeamName, v.teamName)
	return m.status
}

var managedAppForProfTest = management.ManagedApplicationProfile{
	ResourceMeta: apiv1.ResourceMeta{
		Name: "app-test",
		Metadata: apiv1.Metadata{
			ID: "11",
			Scope: apiv1.MetadataScope{
				Kind: v1alpha1.EnvironmentGVK().Kind,
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
	Spec: management.ManagedApplicationProfileSpec{
		ManagedApplication:           "app-test",
		ApplicationProfileDefinition: "app-prof-def",
		Data: map[string]interface{}{
			"enumMapData": "label2",
			"data2":       "value2",
		},
	},
	Status: &apiv1.ResourceStatus{
		Level: prov.Pending.String(),
	},
}
