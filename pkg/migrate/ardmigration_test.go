package migrate

import (
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestArdMigration(t *testing.T) {
	cfg := &config.CentralConfiguration{
		Environment: "mock-env",
	}
	c := &mockArdMigClient{}
	mig := NewArdMigration(c, cfg)
	ard := mv1a.NewAccessRequestDefinition("asdf", cfg.GetEnvironmentName())
	ard.Spec.Schema = map[string]interface{}{
		"properties": map[string]interface{}{
			"scopes": []string{"scope1", "scope2"},
		},
	}
	ri, _ := ard.AsInstance()
	ri, err := mig.Migrate(ri)
	ard.FromInstance(ri)
	assert.Nil(t, err)
	scopes := mig.getScopes(ard.Spec.Schema)
	assert.Nil(t, scopes)
}

type mockArdMigClient struct{}

func (m mockArdMigClient) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	return nil, nil
}

func (m mockArdMigClient) GetAPIV1ResourceInstancesWithPageSize(query map[string]string, URL string, pageSize int) ([]*apiv1.ResourceInstance, error) {
	return nil, nil
}

func (m mockArdMigClient) UpdateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
	r, err := ri.AsInstance()
	return r, err
}

func (m mockArdMigClient) CreateOrUpdateResource(data apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return nil, nil
}

func (m mockArdMigClient) CreateSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) error {
	return nil
}
