package migrate

import (
	"context"
	"encoding/json"
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
		tags            []string
		expectedTags    int
	}{
		{
			name:            "should move api service attributes to the x-agent-details sub resource",
			updateCalled:    true,
			createSubCalled: true,
			tags:            []string{"tag1", "tag2"},
			expectedTags:    1,
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
			tags:            []string{"abc", "123"},
			expectedTags:    2,
			attrs: map[string]string{
				"random": "abc",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			MatchAttr("majorHash", "minorHash")
			MatchAttrPattern("az-")
			RemoveTagPattern("tag1")
			res := []*apiv1.ResourceInstance{
				{
					ResourceMeta: apiv1.ResourceMeta{
						GroupVersionKind: management.APIServiceGVK(),
						Name:             "item-one",
						Title:            "item-one",
						Metadata:         apiv1.Metadata{},
						Attributes:       tc.attrs,
						Tags:             tc.tags,
					},
				},
			}
			c := &mockAttrMigClient{
				res:          res,
				t:            t,
				expectedTags: tc.expectedTags,
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
	c := &mockAttrMigClient{
		t: t,
	}
	cfg := &config.CentralConfiguration{}
	am := NewAttributeMigration(c, cfg)
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: management.APIServiceGVK(),
			Attributes: map[string]string{
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
	}

	c.execRes = ri

	MatchAttr("majorHash", "minorHash")
	MatchAttrPattern("az-")

	svc, err := am.Migrate(context.Background(), ri)
	assert.Nil(t, err)
	assert.NotNil(t, util.GetAgentDetails(svc))
}

type mockAttrMigClient struct {
	res             []*apiv1.ResourceInstance
	t               *testing.T
	updateCalled    bool
	createSubCalled bool
	execRes         *apiv1.ResourceInstance
	expectedTags    int
}

func (m *mockAttrMigClient) GetAPIV1ResourceInstances(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
	return m.res, nil
}

func (m *mockAttrMigClient) UpdateResourceInstance(i apiv1.Interface) (*apiv1.ResourceInstance, error) {
	m.updateCalled = true
	ri, _ := i.AsInstance()
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

	assert.Equal(m.t, m.expectedTags, len(ri.Tags))

	return nil, nil
}

func (m *mockAttrMigClient) CreateSubResource(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
	m.createSubCalled = true
	return nil
}

func (m *mockAttrMigClient) CreateOrUpdateResource(data apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return m.execRes, nil
}

func (m *mockAttrMigClient) ExecuteAPI(_, _ string, _ map[string]string, _ []byte) ([]byte, error) {
	return json.Marshal(m.execRes)
}

func (m mockAttrMigClient) DeleteResourceInstance(ri apiv1.Interface) error {
	return nil
}

func (m mockAttrMigClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	return nil, nil
}
