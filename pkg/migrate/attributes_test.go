package migrate

import (
	"encoding/json"
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestAttributeMigration(t *testing.T) {
	tests := []struct {
		name            string
		attrs           map[string]string
		updateCalled    bool
		createSubCalled bool
	}{
		{
			name:            "should move api service attributes to the x-agent-details sub resource",
			updateCalled:    true,
			createSubCalled: true,
			attrs: map[string]string{
				defs.AttrPreviousAPIServiceRevisionID: "1",
				defs.AttrExternalAPIID:                "2",
				defs.AttrExternalAPIPrimaryKey:        "3",
				defs.AttrExternalAPIName:              "api-name",
				defs.AttrExternalAPIStage:             "stage",
				defs.AttrCreatedBy:                    "created-by",
				"majorHash":                           "major",
				"minorHash":                           "minor",
				"az-api-hash":                         "azhash",
				"az-resource-id":                      "resourceid",
				"random":                              "abc",
			},
		},
		{
			name:            "should not call update when there are no attributes to move",
			updateCalled:    false,
			createSubCalled: false,
			attrs: map[string]string{
				"random": "abc",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			AddAttr("majorHash", "minorHash")
			AddPattern("az-")
			res := []*apiv1.ResourceInstance{
				{
					ResourceMeta: apiv1.ResourceMeta{
						GroupVersionKind: mv1a.APIServiceGVK(),
						Name:             "item-one",
						Title:            "item-one",
						Metadata:         apiv1.Metadata{},
						Attributes:       tc.attrs,
					},
				},
			}
			c := &mockClient{
				res: res,
				t:   t,
			}
			cfg := &config.CentralConfiguration{}
			am := NewAttributeMigration(c, cfg)
			err := am.migrate("/apiservices", nil)
			assert.Equal(t, tc.updateCalled, c.updateCalled)
			assert.Equal(t, tc.createSubCalled, c.createSubCalled)
			assert.Nil(t, err)
		})
	}
}

func TestMigrate(t *testing.T) {
	c := &mockClient{
		t: t,
	}
	cfg := &config.CentralConfiguration{}
	am := NewAttributeMigration(c, cfg)
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: mv1a.APIServiceGVK(),
			Attributes:       map[string]string{},
		},
	}

	c.execRes = ri

	err := am.Migrate(ri)
	assert.Nil(t, err)
}

func Test_getPlural(t *testing.T) {
	kind, _ := getPlural(mv1a.APIServiceGVK().Kind)
	assert.Equal(t, mv1a.APIServiceResourceName, kind)

	kind, _ = getPlural(mv1a.APIServiceRevisionGVK().Kind)
	assert.Equal(t, mv1a.APIServiceRevisionResourceName, kind)

	kind, _ = getPlural(mv1a.APIServiceInstanceGVK().Kind)
	assert.Equal(t, mv1a.APIServiceInstanceResourceName, kind)

	kind, _ = getPlural(mv1a.ConsumerInstanceGVK().Kind)
	assert.Equal(t, mv1a.ConsumerInstanceResourceName, kind)

	kind, err := getPlural("abc")
	assert.Empty(t, kind)
	assert.Error(t, err)
}

type mockClient struct {
	res             []*apiv1.ResourceInstance
	t               *testing.T
	updateCalled    bool
	createSubCalled bool
	execRes         *apiv1.ResourceInstance
}

func (m *mockClient) GetAPIV1ResourceInstancesWithPageSize(_ map[string]string, _ string, _ int) ([]*apiv1.ResourceInstance, error) {
	return m.res, nil
}

func (m *mockClient) UpdateAPIV1ResourceInstance(_ string, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	m.updateCalled = true
	assert.NotContains(m.t, ri.Attributes, defs.AttrPreviousAPIServiceRevisionID)
	assert.NotContains(m.t, ri.Attributes, defs.AttrExternalAPIID)
	assert.NotContains(m.t, ri.Attributes, defs.AttrExternalAPIPrimaryKey)
	assert.NotContains(m.t, ri.Attributes, defs.AttrExternalAPIName)
	assert.NotContains(m.t, ri.Attributes, defs.AttrExternalAPIStage)
	assert.NotContains(m.t, ri.Attributes, defs.AttrCreatedBy)
	assert.NotContains(m.t, ri.Attributes, "majorHash")
	assert.NotContains(m.t, ri.Attributes, "minorHash")
	assert.NotContains(m.t, ri.Attributes, "az-api-hash")
	assert.NotContains(m.t, ri.Attributes, "az-resource-id")
	assert.Contains(m.t, ri.Attributes, "random")

	sub := util.GetAgentDetails(ri)
	assert.Contains(m.t, sub, defs.AttrPreviousAPIServiceRevisionID)
	assert.Contains(m.t, sub, defs.AttrExternalAPIID)
	assert.Contains(m.t, sub, defs.AttrExternalAPIPrimaryKey)
	assert.Contains(m.t, sub, defs.AttrExternalAPIName)
	assert.Contains(m.t, sub, defs.AttrExternalAPIStage)
	assert.Contains(m.t, sub, defs.AttrCreatedBy)
	assert.Contains(m.t, sub, "majorHash")
	assert.Contains(m.t, sub, "minorHash")
	assert.Contains(m.t, sub, "az-api-hash")
	assert.Contains(m.t, sub, "az-resource-id")
	assert.NotContains(m.t, sub, "random")

	return nil, nil
}

func (m *mockClient) CreateSubResourceScoped(_, _, _, _, _, _ string, _ map[string]interface{}) error {
	m.createSubCalled = true
	return nil
}

func (m *mockClient) ExecuteAPI(_, _ string, _ map[string]string, _ []byte) ([]byte, error) {
	return json.Marshal(m.execRes)
}
