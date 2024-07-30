package migrate

import (
	"context"
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestArdMigration(t *testing.T) {
	cfg := &config.CentralConfiguration{
		Environment: "mock-env",
	}
	c := &mockArdMigClient{}
	mig := NewArdMigration(c, cfg)
	ard := management.NewAccessRequestDefinition("asdf", cfg.GetEnvironmentName())
	ard.Spec.Schema = map[string]interface{}{
		"properties": map[string]interface{}{
			"scopes": []string{"scope1", "scope2"},
		},
	}
	ri, _ := ard.AsInstance()
	ri, err := mig.Migrate(context.Background(), ri)
	ard.FromInstance(ri)
	assert.Nil(t, err)
	scopes := mig.getScopes(ard.Spec.Schema)
	assert.Nil(t, scopes)
}

type mockArdMigClient struct{}

func (m mockArdMigClient) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	return nil, nil
}

func (m mockArdMigClient) GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*apiv1.ResourceInstance, error) {
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

func (m mockArdMigClient) DeleteResourceInstance(ri apiv1.Interface) error {
	return nil
}

func (m mockArdMigClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	return nil, nil
}
