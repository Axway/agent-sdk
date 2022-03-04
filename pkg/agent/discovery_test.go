package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/stretchr/testify/assert"
)

type mockSvcClient struct {
	apiSvc *v1alpha1.APIService
	err    error
}

func (m mockSvcClient) SetTokenGetter(_ auth.PlatformTokenGetter) {
}

func (m mockSvcClient) SetConfig(_ corecfg.CentralConfig) {
}

func (m mockSvcClient) PublishService(_ *apic.ServiceBody) (*v1alpha1.APIService, error) {
	return m.apiSvc, nil
}

func (m mockSvcClient) RegisterSubscriptionWebhook() error {
	return m.err
}

func (m mockSvcClient) RegisterSubscriptionSchema(schema apic.SubscriptionSchema, update bool) error {
	return nil
}

func (m mockSvcClient) UpdateSubscriptionSchema(schema apic.SubscriptionSchema) error {
	return nil
}

func (m mockSvcClient) GetSubscriptionManager() apic.SubscriptionManager {
	return nil
}

func (m mockSvcClient) GetCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	return "", nil
}

func (m mockSvcClient) DeleteAPIServiceInstance(name string) error {
	return nil
}

func (m mockSvcClient) DeleteConsumerInstance(name string) error {
	return nil
}

func (m mockSvcClient) DeleteServiceByName(name string) error {
	return nil
}

func (m mockSvcClient) GetConsumerInstanceByID(id string) (*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}

func (m mockSvcClient) GetConsumerInstancesByExternalAPIID(externalAPIID string) ([]*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}

func (m mockSvcClient) UpdateConsumerInstanceSubscriptionDefinition(externalAPIID, subscriptionDefinitionName string) error {
	return nil
}

func (m mockSvcClient) GetUserEmailAddress(ID string) (string, error) {
	return "", nil
}

func (m mockSvcClient) GetUserName(ID string) (string, error) {
	return "", nil
}

func (m mockSvcClient) GetSubscriptionsForCatalogItem(states []string, catalogItemID string) ([]apic.CentralSubscription, error) {
	return nil, nil
}

func (m mockSvcClient) GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (apic.SubscriptionSchema, error) {
	return nil, nil
}

func (m mockSvcClient) UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, schema apic.SubscriptionSchema) error {
	return nil
}

func (m mockSvcClient) GetCatalogItemName(ID string) (string, error) {
	return "", nil
}

func (m mockSvcClient) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	return nil, nil
}

func (m mockSvcClient) Healthcheck(name string) *hc.Status {
	return nil
}

func (m mockSvcClient) GetAPIRevisions(query map[string]string, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m mockSvcClient) GetAPIServiceRevisions(query map[string]string, URL, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m mockSvcClient) GetAPIServiceInstances(query map[string]string, URL string) ([]*v1alpha1.APIServiceInstance, error) {
	return nil, nil
}

func (m mockSvcClient) GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*v1.ResourceInstance, error) {
	return nil, nil
}

func (m mockSvcClient) GetAPIV1ResourceInstancesWithPageSize(query map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error) {
	return nil, nil
}

func (m mockSvcClient) GetAPIServiceByName(name string) (*v1alpha1.APIService, error) {
	return nil, nil
}

func (m mockSvcClient) GetAPIServiceInstanceByName(name string) (*v1alpha1.APIServiceInstance, error) {
	return nil, nil
}

func (m mockSvcClient) GetAPIRevisionByName(name string) (*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m mockSvcClient) CreateCategory(name string) (*catalog.Category, error) {
	return nil, nil
}

func (m mockSvcClient) GetOrCreateCategory(category string) string {
	return ""
}

func (m mockSvcClient) GetEnvironment() (*v1alpha1.Environment, error) {
	return nil, nil
}

func (m mockSvcClient) GetCentralTeamByName(name string) (*definitions.PlatformTeam, error) {
	return nil, nil
}

func (m mockSvcClient) GetTeam(query map[string]string) ([]definitions.PlatformTeam, error) {
	return nil, nil
}

func (m mockSvcClient) GetAccessControlList(aclName string) (*v1alpha1.AccessControlList, error) {
	return nil, nil
}

func (m mockSvcClient) UpdateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error) {
	return nil, nil
}

func (m mockSvcClient) CreateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error) {
	return nil, nil
}

func (m mockSvcClient) UpdateAPIV1ResourceInstance(url string, ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	return nil, nil
}

func (m mockSvcClient) CreateSubResourceScoped(scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{}) error {
	return nil
}

func (m mockSvcClient) CreateSubResourceUnscoped(kindPlural, name, group, version string, subs map[string]interface{}) error {
	return nil
}

func (m mockSvcClient) GetResource(url string) (*v1.ResourceInstance, error) {
	return nil, nil
}

func (m mockSvcClient) CreateResource(url string, bts []byte) (*v1.ResourceInstance, error) {
	return nil, nil
}

func (m mockSvcClient) UpdateResource(url string, bts []byte) (*v1.ResourceInstance, error) {
	return nil, nil
}

func TestDiscoveryCache(t *testing.T) {
	dcj := newDiscoveryCache(nil, true, &sync.Mutex{}, nil)
	dcj.getHCStatus = func(_ string) hc.StatusLevel {
		return hc.OK
	}
	attributeKey := "Attr1"
	attributeValue := "testValue"
	emptyAPISvc := []v1.ResourceInstance{}
	apiSvc1 := v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             "testAPIService1",
			SubResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{
					definitions.AttrExternalAPIID:         "1111",
					definitions.AttrExternalAPIPrimaryKey: "1234",
					definitions.AttrExternalAPIName:       "NAME",
					attributeKey:                          attributeValue,
				},
			},
		},
	}
	apiSvc2 := v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             "testAPIService2",
			SubResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{
					definitions.AttrExternalAPIID: "2222",
				},
			},
		},
	}
	var serverAPISvcResponse []v1.ResourceInstance
	environmentRes := &v1alpha1.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "test",
			Title:    "test",
		},
	}
	teams := []definitions.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			buf, _ := json.Marshal(serverAPISvcResponse)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test") {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/api/v1/platformTeams") {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}
	}))
	defer s.Close()

	cfg := createCentralCfg(s.URL, "test")
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)

	assert.True(t, dcj.Ready())
	assert.Nil(t, dcj.Status())

	serverAPISvcResponse = emptyAPISvc
	dcj.updateAPICache()
	assert.Equal(t, 0, len(agent.cacheManager.GetAPIServiceKeys()))
	assert.False(t, IsAPIPublishedByID("1111"))
	assert.False(t, IsAPIPublishedByID("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	dcj.updateAPICache()
	assert.Equal(t, 1, len(agent.cacheManager.GetAPIServiceKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.False(t, IsAPIPublishedByID("2222"))
	assert.Equal(t, "1111", GetAttributeOnPublishedAPIByID("1111", definitions.AttrExternalAPIID))
	assert.Equal(t, "", GetAttributeOnPublishedAPIByID("2222", definitions.AttrExternalAPIID))
	assert.Equal(t, attributeValue, GetAttributeOnPublishedAPIByPrimaryKey("1234", attributeKey))
	assert.Equal(t, attributeValue, GetAttributeOnPublishedAPIByName("NAME", attributeKey))

	apicClient := agent.apicClient
	var apiSvc v1alpha1.APIService
	apiSvc.FromInstance(&apiSvc2)
	agent.apicClient = &mockSvcClient{apiSvc: &apiSvc}
	StartAgentStatusUpdate()
	PublishAPI(apic.ServiceBody{})
	agent.apicClient = apicClient
	assert.Equal(t, 2, len(agent.cacheManager.GetAPIServiceKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.True(t, IsAPIPublishedByID("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	dcj.updateAPICache()
	assert.Equal(t, 1, len(agent.cacheManager.GetAPIServiceKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.True(t, IsAPIPublishedByPrimaryKey("1234"))
	assert.False(t, IsAPIPublishedByID("2222"))
}
