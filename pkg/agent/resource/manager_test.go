package resource

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/json"
)

type mockSvcClient struct {
	apiResponse []byte
}

func (m *mockSvcClient) GetEnvironment() (*v1alpha1.Environment, error) {
	return nil, nil
}

func (m *mockSvcClient) GetCentralTeamByName(_ string) (*definitions.PlatformTeam, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIRevisions(queryParams map[string]string, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIServiceInstances(queryParams map[string]string, URL string) ([]*v1alpha1.APIServiceInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIServiceRevisions(queryParams map[string]string, URL, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIV1ResourceInstancesWithPageSize(queryParams map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIV1ResourceInstances(queryParams map[string]string, URL string) ([]*v1.ResourceInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) CreateCategory(categoryName string) (*catalog.Category, error) {
	return nil, nil
}

func (m *mockSvcClient) AddCache(categoryCache cache.Cache, teamCache cache.Cache) {}

func (m *mockSvcClient) GetOrCreateCategory(category string) string { return "" }

func (m *mockSvcClient) GetAPIServiceByName(serviceName string) (*v1alpha1.APIService, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIRevisionByName(revisionName string) (*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIServiceInstanceByName(instanceName string) (*v1alpha1.APIServiceInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) SetTokenGetter(tokenGetter auth.PlatformTokenGetter) {}
func (m *mockSvcClient) PublishService(serviceBody *apic.ServiceBody) (*v1alpha1.APIService, error) {
	return nil, nil
}
func (m *mockSvcClient) RegisterSubscriptionWebhook() error { return nil }
func (m *mockSvcClient) RegisterSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema, update bool) error {
	return nil
}
func (m *mockSvcClient) UpdateSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema) error {
	return nil
}
func (m *mockSvcClient) GetSubscriptionManager() apic.SubscriptionManager { return nil }
func (m *mockSvcClient) GetCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	return "", nil
}
func (m *mockSvcClient) DeleteServiceByName(_ string) error                 { return nil }
func (m *mockSvcClient) DeleteConsumerInstance(instanceName string) error   { return nil }
func (m *mockSvcClient) DeleteAPIServiceInstance(instanceName string) error { return nil }
func (m *mockSvcClient) UpdateConsumerInstanceSubscriptionDefinition(externalAPIID, subscriptionDefinitionName string) error {
	return nil
}
func (m *mockSvcClient) GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}
func (m *mockSvcClient) GetConsumerInstancesByExternalAPIID(consumerInstanceID string) ([]*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) GetUserName(ID string) (string, error)         { return "", nil }
func (m *mockSvcClient) GetUserEmailAddress(ID string) (string, error) { return "", nil }
func (m *mockSvcClient) GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]apic.CentralSubscription, error) {
	return nil, nil
}
func (m *mockSvcClient) GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (apic.SubscriptionSchema, error) {
	return nil, nil
}
func (m *mockSvcClient) Healthcheck(name string) *hc.Status {
	return &hc.Status{Result: hc.OK}
}

func (m *mockSvcClient) UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, subscriptionSchema apic.SubscriptionSchema) error {
	return nil
}

func (m *mockSvcClient) GetCatalogItemName(ID string) (string, error) { return "", nil }
func (m *mockSvcClient) OnConfigChange(cfg config.CentralConfig)      {}
func (m *mockSvcClient) SetConfig(cfg config.CentralConfig)           {}

func (m *mockSvcClient) GetTeam(queryParams map[string]string) ([]definitions.PlatformTeam, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAccessControlList(aclName string) (*v1alpha1.AccessControlList, error) {
	return nil, nil
}

func (m *mockSvcClient) UpdateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error) {
	return nil, nil
}

func (m *mockSvcClient) CreateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error) {
	return nil, nil
}

func (m *mockSvcClient) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	if method == api.PUT {
		m.apiResponse = buffer
	}
	return m.apiResponse, nil
}
func createDiscoveryAgentRes(id, name, dataplane, team string) *v1.ResourceInstance {
	res := &v1alpha1.DiscoveryAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.DiscoveryAgentSpec{
			DataplaneType: dataplane,
			Config: v1alpha1.DiscoveryAgentSpecConfig{
				OwningTeam: team,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createTraceabilityAgentRes(id, name, dataplane, team string) *v1.ResourceInstance {
	res := &v1alpha1.TraceabilityAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.TraceabilityAgentSpec{
			DataplaneType: dataplane,
			Config: v1alpha1.TraceabilityAgentSpecConfig{
				OwningTeam: team,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createGovernanceAgentRes(id, name, dataplane, team string) *v1.ResourceInstance {
	res := &v1alpha1.GovernanceAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.GovernanceAgentSpec{
			DataplaneType: dataplane,
			Config: map[string]interface{}{
				"team": team,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func TestNewManager(t *testing.T) {
	cfg := &config.CentralConfiguration{}
	m, err := NewAgentResourceManager(cfg, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, m)

	cfg.AgentName = "Test-DA"
	m, err = NewAgentResourceManager(cfg, nil, nil)
	assert.NotNil(t, err)
	assert.Nil(t, m)

	resource := createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", "")
	svcClient := &mockSvcClient{}
	svcClient.apiResponse, _ = json.Marshal(resource)
	agentResChangeHandlerCall := 0
	f := func() { agentResChangeHandlerCall++ }
	cfg.AgentType = config.DiscoveryAgent
	cfg.AgentName = "Test-DA"
	m, err = NewAgentResourceManager(cfg, svcClient, f)
	assert.Nil(t, err)
	assert.NotNil(t, m)
	m.SetAgentResource(createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", "update"))
	assert.Equal(t, 1, agentResChangeHandlerCall)
}

func TestAgentConfigOverride(t *testing.T) {
	tests := []struct {
		name            string
		agentType       config.AgentType
		agentName       string
		resource        *v1.ResourceInstance
		updatedResource *v1.ResourceInstance
	}{
		{
			name:            "DiscoveryAgent override",
			agentType:       config.DiscoveryAgent,
			agentName:       "Test-DA",
			resource:        createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", ""),
			updatedResource: createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", "TestTeam"),
		},
		{
			name:            "TraceabilityAgent override",
			agentType:       config.TraceabilityAgent,
			agentName:       "Test-TA",
			resource:        createTraceabilityAgentRes("111", "Test-TA", "test-dataplane", ""),
			updatedResource: createTraceabilityAgentRes("111", "Test-TA", "test-dataplane", "TestTeam"),
		},
		{
			name:            "GovernanceAgent override",
			agentType:       config.GovernanceAgent,
			agentName:       "Test-GA",
			resource:        createGovernanceAgentRes("111", "Test-GA", "test-dataplane", ""),
			updatedResource: createGovernanceAgentRes("111", "Test-GA", "test-dataplane", "TestTeam"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.CentralConfiguration{}
			cfg.AgentName = tc.agentName
			cfg.AgentType = tc.agentType

			svcClient := &mockSvcClient{}
			svcClient.apiResponse, _ = json.Marshal(tc.resource)

			agentResChangeHandlerCall := 0
			f := func() { agentResChangeHandlerCall++ }
			m, err := NewAgentResourceManager(cfg, svcClient, f)
			assert.Nil(t, err)
			assert.NotNil(t, m)

			res := m.GetAgentResource()
			assertAgentResource(t, res, tc.resource)
			assert.Equal(t, 0, agentResChangeHandlerCall)

			// Get same resource does not invoke change handler
			m.FetchAgentResource()
			res = m.GetAgentResource()
			assertAgentResource(t, res, tc.resource)
			assert.Equal(t, 0, agentResChangeHandlerCall)

			// Updated resource invokes change handler
			svcClient.apiResponse, _ = json.Marshal(tc.updatedResource)
			m.FetchAgentResource()

			res = m.GetAgentResource()
			assertAgentResource(t, res, tc.updatedResource)
			assert.Equal(t, 1, agentResChangeHandlerCall)
		})
	}
}

func TestAgentUpdateStatus(t *testing.T) {
	tests := []struct {
		name      string
		agentType config.AgentType
		agentName string
		resource  *v1.ResourceInstance
	}{
		{
			name:      "DiscoveryAgent override",
			agentType: config.DiscoveryAgent,
			agentName: "Test-DA",
			resource:  createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", ""),
		},
		{
			name:      "TraceabilityAgent override",
			agentType: config.TraceabilityAgent,
			agentName: "Test-TA",
			resource:  createTraceabilityAgentRes("111", "Test-TA", "test-dataplane", ""),
		},
		{
			name:      "GovernanceAgent override",
			agentType: config.GovernanceAgent,
			agentName: "Test-GA",
			resource:  createGovernanceAgentRes("111", "Test-GA", "test-dataplane", ""),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.CentralConfiguration{}
			cfg.AgentName = tc.agentName
			cfg.AgentType = tc.agentType

			svcClient := &mockSvcClient{}
			svcClient.apiResponse, _ = json.Marshal(tc.resource)

			m, err := NewAgentResourceManager(cfg, svcClient, nil)

			assert.Nil(t, err)
			assert.NotNil(t, m)
			m.UpdateAgentStatus("stopped", "running", "test")
			assertAgentStatusResource(t, svcClient.apiResponse, tc.agentName, "stopped", "running", "test")
		})
	}
}

func assertAgentResource(t *testing.T, res, expectedRes *v1.ResourceInstance) {
	assert.Equal(t, expectedRes.Group, res.Group)
	assert.Equal(t, expectedRes.Kind, res.Kind)
	assert.Equal(t, expectedRes.Name, res.Name)
	assert.Equal(t, expectedRes.Metadata.ID, res.Metadata.ID)
	assert.Equal(t, expectedRes.Spec["dataplane"], res.Spec["dataplane"])
	assert.Equal(t, expectedRes.Spec["config"], res.Spec["config"])
}

func assertAgentStatusResource(t *testing.T, res []byte, agentName, state, previousState, message string) {
	var agentRes map[string]interface{}
	json.Unmarshal(res, &agentRes)
	statusSubRes := agentRes["status"].(map[string]interface{})

	assert.NotNil(t, statusSubRes)
	assert.Equal(t, statusSubRes["state"], state)
	assert.Equal(t, statusSubRes["previousState"], previousState)
	assert.Equal(t, statusSubRes["message"], message)
}
