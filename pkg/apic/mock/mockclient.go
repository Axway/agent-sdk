package mock

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

type Client struct {
	SetTokenGetterMock                                       func(tokenRequester auth.PlatformTokenGetter)
	SetConfigMock                                            func(cfg corecfg.CentralConfig)
	PublishServiceMock                                       func(serviceBody *apic.ServiceBody) (*v1alpha1.APIService, error)
	RegisterSubscriptionWebhookMock                          func() error
	RegisterSubscriptionSchemaMock                           func(subscriptionSchema apic.SubscriptionSchema, update bool) error
	UpdateSubscriptionSchemaMock                             func(subscriptionSchema apic.SubscriptionSchema) error
	GetSubscriptionManagerMock                               func() apic.SubscriptionManager
	GetCatalogItemIDForConsumerInstanceMock                  func(instanceID string) (string, error)
	DeleteAPIServiceInstanceMock                             func(name string) error
	DeleteAPIServiceInstanceWithFinalizersMock               func(ri *v1.ResourceInstance) error
	DeleteConsumerInstanceMock                               func(name string) error
	DeleteServiceByNameMock                                  func(name string) error
	GetConsumerInstanceByIDMock                              func(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error)
	GetConsumerInstancesByExternalAPIIDMock                  func(externalAPIID string) ([]*v1alpha1.ConsumerInstance, error)
	UpdateConsumerInstanceSubscriptionDefinitionMock         func(externalAPIID, subscriptionDefinitionName string) error
	GetUserEmailAddressMock                                  func(ID string) (string, error)
	GetUserNameMock                                          func(ID string) (string, error)
	GetSubscriptionsForCatalogItemMock                       func(states []string, catalogItemID string) ([]apic.CentralSubscription, error)
	GetSubscriptionDefinitionPropertiesForCatalogItemMock    func(catalogItemID, propertyKey string) (apic.SubscriptionSchema, error)
	UpdateSubscriptionDefinitionPropertiesForCatalogItemMock func(catalogItemID, propertyKey string, subscriptionSchema apic.SubscriptionSchema) error
	GetCatalogItemNameMock                                   func(ID string) (string, error)
	ExecuteAPIMock                                           func(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	HealthcheckMock                                          func(name string) *hc.Status
	GetAPIRevisionsMock                                      func(queryParams map[string]string, stage string) ([]*v1alpha1.APIServiceRevision, error)
	GetAPIServiceRevisionsMock                               func(queryParams map[string]string, URL, stage string) ([]*v1alpha1.APIServiceRevision, error)
	GetAPIServiceInstancesMock                               func(queryParams map[string]string, URL string) ([]*v1alpha1.APIServiceInstance, error)
	GetAPIV1ResourceInstancesMock                            func(queryParams map[string]string, URL string) ([]*v1.ResourceInstance, error)
	GetAPIV1ResourceInstancesWithPageSizeMock                func(queryParams map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error)
	GetAPIServiceByNameMock                                  func(serviceName string) (*v1alpha1.APIService, error)
	GetAPIServiceInstanceByNameMock                          func(serviceInstanceName string) (*v1alpha1.APIServiceInstance, error)
	GetAPIRevisionByNameMock                                 func(serviceRevisionName string) (*v1alpha1.APIServiceRevision, error)
	CreateCategoryMock                                       func(categoryName string) (*catalog.Category, error)
	GetOrCreateCategoryMock                                  func(category string) string
	GetEnvironmentMock                                       func() (*v1alpha1.Environment, error)
	GetCentralTeamByNameMock                                 func(teamName string) (*definitions.PlatformTeam, error)
	GetTeamMock                                              func(queryParams map[string]string) ([]definitions.PlatformTeam, error)
	GetAccessControlListMock                                 func(aclName string) (*v1alpha1.AccessControlList, error)
	UpdateAccessControlListMock                              func(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error)
	CreateAccessControlListMock                              func(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error)
	RegisterCredentialRequestDefinitionMock                  func(data *v1alpha1.CredentialRequestDefinition, update bool) (*v1alpha1.CredentialRequestDefinition, error)
	RegisterAccessRequestDefinitionMock                      func(data *v1alpha1.AccessRequestDefinition, update bool) (*v1alpha1.AccessRequestDefinition, error)
	UpdateAPIV1ResourceInstanceMock                          func(url string, ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
	CreateSubResourceScopedMock                              func(scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{}) error
	CreateSubResourceUnscopedMock                            func(kindPlural, name, group, version string, subs map[string]interface{}) error
	GetResourceMock                                          func(url string) (*v1.ResourceInstance, error)
	CreateResourceMock                                       func(url string, bts []byte) (*v1.ResourceInstance, error)
	UpdateResourceMock                                       func(url string, bts []byte) (*v1.ResourceInstance, error)
	UpdateResourceFinalizerMock                              func(res *v1.ResourceInstance, finalizer, description string, addAction bool) (*v1.ResourceInstance, error)
}

func (m *Client) GetEnvironment() (*v1alpha1.Environment, error) {
	if m.GetEnvironmentMock != nil {
		return m.GetEnvironmentMock()
	}
	return nil, nil
}

func (m *Client) GetCentralTeamByName(teamName string) (*definitions.PlatformTeam, error) {
	if m.GetCentralTeamByNameMock != nil {
		return m.GetCentralTeamByNameMock(teamName)
	}
	return nil, nil
}

func (m *Client) GetAPIRevisions(queryParams map[string]string, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	if m.GetAPIRevisionsMock != nil {
		return m.GetAPIRevisionsMock(queryParams, stage)
	}
	return nil, nil
}

func (m *Client) GetAPIServiceInstances(queryParams map[string]string, URL string) ([]*v1alpha1.APIServiceInstance, error) {
	if m.GetAPIServiceInstancesMock != nil {
		return m.GetAPIServiceInstancesMock(queryParams, URL)
	}
	return nil, nil
}

func (m *Client) GetAPIServiceRevisions(queryParams map[string]string, URL, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	if m.GetAPIServiceRevisionsMock != nil {
		return m.GetAPIServiceRevisionsMock(queryParams, URL, stage)
	}
	return nil, nil
}

func (m *Client) GetAPIV1ResourceInstancesWithPageSize(queryParams map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error) {
	if m.GetAPIV1ResourceInstancesWithPageSizeMock != nil {
		return m.GetAPIV1ResourceInstancesWithPageSizeMock(queryParams, URL, pageSize)
	}
	return nil, nil
}

func (m *Client) GetAPIV1ResourceInstances(queryParams map[string]string, URL string) ([]*v1.ResourceInstance, error) {
	if m.GetAPIV1ResourceInstancesMock != nil {
		return m.GetAPIV1ResourceInstancesMock(queryParams, URL)
	}
	return nil, nil
}

func (m *Client) CreateCategory(categoryName string) (*catalog.Category, error) {
	if m.CreateCategoryMock != nil {
		return m.CreateCategoryMock(categoryName)
	}
	return nil, nil
}

func (m *Client) GetOrCreateCategory(category string) string {
	if m.GetOrCreateCategoryMock != nil {
		return m.GetOrCreateCategoryMock(category)
	}
	return ""
}

func (m *Client) GetAPIServiceByName(serviceName string) (*v1alpha1.APIService, error) {
	if m.GetAPIServiceByNameMock != nil {
		return m.GetAPIServiceByNameMock(serviceName)
	}
	return nil, nil
}

func (m *Client) GetAPIRevisionByName(revisionName string) (*v1alpha1.APIServiceRevision, error) {
	if m.GetAPIRevisionByNameMock != nil {
		return m.GetAPIRevisionByNameMock(revisionName)
	}
	return nil, nil
}

func (m *Client) GetAPIServiceInstanceByName(instanceName string) (*v1alpha1.APIServiceInstance, error) {
	if m.GetAPIServiceInstanceByNameMock != nil {
		return m.GetAPIServiceInstanceByNameMock(instanceName)
	}
	return nil, nil
}

func (m *Client) SetTokenGetter(tokenGetter auth.PlatformTokenGetter) {
	if m.SetTokenGetterMock != nil {
		m.SetTokenGetterMock(tokenGetter)
	}
}

func (m *Client) PublishService(serviceBody *apic.ServiceBody) (*v1alpha1.APIService, error) {
	if m.PublishServiceMock != nil {
		return m.PublishServiceMock(serviceBody)
	}
	return nil, nil
}
func (m *Client) RegisterSubscriptionWebhook() error {
	if m.RegisterSubscriptionWebhookMock != nil {
		return m.RegisterSubscriptionWebhookMock()
	}
	return nil
}

func (m *Client) RegisterSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema, update bool) error {
	if m.RegisterSubscriptionSchemaMock != nil {
		return m.RegisterSubscriptionSchemaMock(subscriptionSchema, update)
	}
	return nil
}

func (m *Client) UpdateSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema) error {
	if m.UpdateSubscriptionSchemaMock != nil {
		return m.UpdateSubscriptionSchemaMock(subscriptionSchema)
	}
	return nil
}

func (m *Client) GetSubscriptionManager() apic.SubscriptionManager {
	if m.GetSubscriptionManagerMock != nil {
		return m.GetSubscriptionManagerMock()
	}
	return nil
}

func (m *Client) GetCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	if m.GetCatalogItemIDForConsumerInstanceMock != nil {
		return m.GetCatalogItemIDForConsumerInstanceMock(instanceID)
	}
	return "", nil
}

func (m *Client) DeleteServiceByName(serviceName string) error {
	if m.DeleteServiceByNameMock != nil {
		return m.DeleteServiceByNameMock(serviceName)
	}
	return nil
}

func (m *Client) DeleteConsumerInstance(instanceName string) error {
	if m.DeleteConsumerInstanceMock != nil {
		return m.DeleteConsumerInstanceMock(instanceName)
	}
	return nil
}

func (m *Client) DeleteAPIServiceInstance(instanceName string) error {
	if m.DeleteAPIServiceInstanceMock != nil {
		return m.DeleteAPIServiceInstanceMock(instanceName)
	}
	return nil
}

func (m *Client) DeleteAPIServiceInstanceWithFinalizers(ri *v1.ResourceInstance) error {
	if m.DeleteAPIServiceInstanceWithFinalizersMock != nil {
		return m.DeleteAPIServiceInstanceWithFinalizersMock(ri)
	}
	return nil
}

func (m *Client) UpdateConsumerInstanceSubscriptionDefinition(externalAPIID, subscriptionDefinitionName string) error {
	if m.UpdateConsumerInstanceSubscriptionDefinitionMock != nil {
		return m.UpdateConsumerInstanceSubscriptionDefinitionMock(externalAPIID, subscriptionDefinitionName)
	}
	return nil
}

func (m *Client) GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error) {
	if m.GetConsumerInstanceByIDMock != nil {
		return m.GetConsumerInstanceByIDMock(consumerInstanceID)
	}
	return nil, nil
}
func (m *Client) GetConsumerInstancesByExternalAPIID(consumerInstanceID string) ([]*v1alpha1.ConsumerInstance, error) {
	if m.GetConsumerInstancesByExternalAPIIDMock != nil {
		return m.GetConsumerInstancesByExternalAPIIDMock(consumerInstanceID)
	}
	return nil, nil
}

func (m *Client) GetUserName(ID string) (string, error) {
	if m.GetUserNameMock != nil {
		return m.GetUserNameMock(ID)
	}
	return "", nil
}

func (m *Client) GetUserEmailAddress(ID string) (string, error) {
	if m.GetUserEmailAddressMock != nil {
		return m.GetUserEmailAddressMock(ID)
	}
	return "", nil
}

func (m *Client) GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]apic.CentralSubscription, error) {
	if m.GetSubscriptionsForCatalogItemMock != nil {
		return m.GetSubscriptionsForCatalogItemMock(states, instanceID)
	}
	return nil, nil
}

func (m *Client) GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (apic.SubscriptionSchema, error) {
	if m.GetSubscriptionDefinitionPropertiesForCatalogItemMock != nil {
		return m.GetSubscriptionDefinitionPropertiesForCatalogItemMock(catalogItemID, propertyKey)
	}
	return nil, nil
}

func (m *Client) Healthcheck(name string) *hc.Status {
	if m.HealthcheckMock != nil {
		return m.HealthcheckMock(name)
	}
	return &hc.Status{Result: hc.OK}
}

func (m *Client) UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, subscriptionSchema apic.SubscriptionSchema) error {
	if m.UpdateSubscriptionDefinitionPropertiesForCatalogItemMock != nil {
		return m.UpdateSubscriptionDefinitionPropertiesForCatalogItemMock(catalogItemID, propertyKey, subscriptionSchema)
	}
	return nil
}

func (m *Client) GetCatalogItemName(ID string) (string, error) {
	if m.GetCatalogItemNameMock != nil {
		return m.GetCatalogItemNameMock(ID)
	}
	return "", nil
}

func (m *Client) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	if m.ExecuteAPIMock != nil {
		return m.ExecuteAPIMock(method, url, queryParam, buffer)
	}
	return nil, nil
}

func (m *Client) SetConfig(cfg corecfg.CentralConfig) {
	if m.SetConfigMock != nil {
		m.SetConfigMock(cfg)
	}
}

func (m *Client) GetTeam(queryParams map[string]string) ([]definitions.PlatformTeam, error) {
	if m.GetTeamMock != nil {
		return m.GetTeamMock(queryParams)
	}
	return nil, nil
}

func (m *Client) GetAccessControlList(aclName string) (*v1alpha1.AccessControlList, error) {
	if m.GetAccessControlListMock != nil {
		return m.GetAccessControlListMock(aclName)
	}
	return nil, nil
}

func (m *Client) UpdateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error) {
	if m.UpdateAccessControlListMock != nil {
		return m.UpdateAccessControlListMock(acl)
	}
	return nil, nil
}

func (m *Client) CreateAccessControlList(acl *v1alpha1.AccessControlList) (*v1alpha1.AccessControlList, error) {
	if m.CreateAccessControlListMock != nil {
		return m.CreateAccessControlListMock(acl)
	}
	return nil, nil
}

func (m *Client) RegisterCredentialRequestDefinition(data *v1alpha1.CredentialRequestDefinition, update bool) (*v1alpha1.CredentialRequestDefinition, error) {
	if m.RegisterCredentialRequestDefinitionMock != nil {
		return m.RegisterCredentialRequestDefinitionMock(data, update)
	}
	return nil, nil
}

func (m *Client) RegisterAccessRequestDefinition(data *v1alpha1.AccessRequestDefinition, update bool) (*v1alpha1.AccessRequestDefinition, error) {
	if m.RegisterAccessRequestDefinitionMock != nil {
		return m.RegisterAccessRequestDefinitionMock(data, update)
	}
	return nil, nil
}

func (m *Client) UpdateAPIV1ResourceInstance(url string, ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	if m.UpdateAPIV1ResourceInstanceMock != nil {
		return m.UpdateAPIV1ResourceInstanceMock(url, ri)
	}
	return nil, nil
}

func (m *Client) CreateSubResourceScoped(scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{}) error {
	if m.CreateSubResourceScopedMock != nil {
		return m.CreateSubResourceScopedMock(scopeKindPlural, scopeName, resKindPlural, name, group, version, subs)
	}
	return nil
}

func (m *Client) CreateSubResourceUnscoped(kindPlural, name, group, version string, subs map[string]interface{}) error {
	if m.CreateSubResourceUnscopedMock != nil {
		return m.CreateSubResourceUnscopedMock(kindPlural, name, group, version, subs)
	}
	return nil
}

func (m *Client) GetResource(url string) (*v1.ResourceInstance, error) {
	if m.GetResourceMock != nil {
		return m.GetResourceMock(url)
	}
	return nil, nil
}

func (m *Client) CreateResource(url string, bts []byte) (*v1.ResourceInstance, error) {
	if m.CreateResourceMock != nil {
		return m.CreateResourceMock(url, bts)
	}
	return nil, nil
}

func (m *Client) UpdateResource(url string, bts []byte) (*v1.ResourceInstance, error) {
	if m.UpdateResourceMock != nil {
		return m.UpdateResourceMock(url, bts)
	}
	return nil, nil
}

func (m *Client) UpdateResourceFinalizer(res *v1.ResourceInstance, finalizer, description string, addAction bool) (*v1.ResourceInstance, error) {
	if m.UpdateResourceFinalizerMock != nil {
		return m.UpdateResourceFinalizerMock(res, finalizer, description, addAction)
	}
	return nil, nil
}
