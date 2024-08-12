package mock

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

type Client struct {
	SetTokenGetterMock                        func(tokenRequester auth.PlatformTokenGetter)
	SetConfigMock                             func(cfg corecfg.CentralConfig)
	PublishServiceMock                        func(serviceBody *apic.ServiceBody) (*management.APIService, error)
	DeleteAPIServiceInstanceMock              func(name string) error
	DeleteServiceByNameMock                   func(name string) error
	GetUserEmailAddressMock                   func(ID string) (string, error)
	GetUserNameMock                           func(ID string) (string, error)
	ExecuteAPIMock                            func(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	HealthcheckMock                           func(name string) *hc.Status
	GetAPIRevisionsMock                       func(queryParams map[string]string, stage string) ([]*management.APIServiceRevision, error)
	GetAPIServiceRevisionsMock                func(queryParams map[string]string, URL, stage string) ([]*management.APIServiceRevision, error)
	GetAPIServiceInstancesMock                func(queryParams map[string]string, URL string) ([]*management.APIServiceInstance, error)
	GetAPIV1ResourceInstancesMock             func(queryParams map[string]string, URL string) ([]*v1.ResourceInstance, error)
	GetAPIV1ResourceInstancesWithPageSizeMock func(queryParams map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error)
	GetAPIServiceByNameMock                   func(serviceName string) (*management.APIService, error)
	GetAPIServiceInstanceByNameMock           func(serviceInstanceName string) (*management.APIServiceInstance, error)
	GetAPIRevisionByNameMock                  func(serviceRevisionName string) (*management.APIServiceRevision, error)
	GetEnvironmentMock                        func() (*management.Environment, error)
	GetCentralTeamByNameMock                  func(teamName string) (*definitions.PlatformTeam, error)
	GetTeamMock                               func(queryParams map[string]string) ([]definitions.PlatformTeam, error)
	GetAccessControlListMock                  func(aclName string) (*management.AccessControlList, error)
	UpdateAccessControlListMock               func(acl *management.AccessControlList) (*management.AccessControlList, error)
	CreateAccessControlListMock               func(acl *management.AccessControlList) (*management.AccessControlList, error)
	UpdateResourceInstanceMock                func(ri v1.Interface) (*v1.ResourceInstance, error)
	CreateResourceInstanceMock                func(ri v1.Interface) (*v1.ResourceInstance, error)
	PatchSubResourceMock                      func(ri v1.Interface, subResourceName string, patches []map[string]interface{}) (*v1.ResourceInstance, error)
	DeleteResourceInstanceMock                func(ri v1.Interface) error
	CreateSubResourceMock                     func(rm v1.ResourceMeta, subs map[string]interface{}) error
	GetResourceMock                           func(url string) (*v1.ResourceInstance, error)
	GetResourcesMock                          func(ri v1.Interface) ([]v1.Interface, error)
	CreateResourceMock                        func(url string, bts []byte) (*v1.ResourceInstance, error)
	UpdateResourceMock                        func(url string, bts []byte) (*v1.ResourceInstance, error)
	UpdateResourceFinalizerMock               func(res *v1.ResourceInstance, finalizer, description string, addAction bool) (*v1.ResourceInstance, error)
	CreateOrUpdateResourceMock                func(v1.Interface) (*v1.ResourceInstance, error)
}

// GetEnvironment -
func (m *Client) GetEnvironment() (*management.Environment, error) {
	if m.GetEnvironmentMock != nil {
		return m.GetEnvironmentMock()
	}
	return nil, nil
}

// GetCentralTeamByName -
func (m *Client) GetCentralTeamByName(teamName string) (*definitions.PlatformTeam, error) {
	if m.GetCentralTeamByNameMock != nil {
		return m.GetCentralTeamByNameMock(teamName)
	}
	return nil, nil
}

// GetAPIRevisions -
func (m *Client) GetAPIRevisions(queryParams map[string]string, stage string) ([]*management.APIServiceRevision, error) {
	if m.GetAPIRevisionsMock != nil {
		return m.GetAPIRevisionsMock(queryParams, stage)
	}
	return nil, nil
}

// GetAPIServiceInstances -
func (m *Client) GetAPIServiceInstances(queryParams map[string]string, URL string) ([]*management.APIServiceInstance, error) {
	if m.GetAPIServiceInstancesMock != nil {
		return m.GetAPIServiceInstancesMock(queryParams, URL)
	}
	return nil, nil
}

// GetAPIServiceRevisions -
func (m *Client) GetAPIServiceRevisions(queryParams map[string]string, URL, stage string) ([]*management.APIServiceRevision, error) {
	if m.GetAPIServiceRevisionsMock != nil {
		return m.GetAPIServiceRevisionsMock(queryParams, URL, stage)
	}
	return nil, nil
}

// GetAPIV1ResourceInstancesWithPageSize -
func (m *Client) GetAPIV1ResourceInstancesWithPageSize(queryParams map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error) {
	if m.GetAPIV1ResourceInstancesWithPageSizeMock != nil {
		return m.GetAPIV1ResourceInstancesWithPageSizeMock(queryParams, URL, pageSize)
	}
	return nil, nil
}

// GetAPIV1ResourceInstances -
func (m *Client) GetAPIV1ResourceInstances(queryParams map[string]string, URL string) ([]*v1.ResourceInstance, error) {
	if m.GetAPIV1ResourceInstancesMock != nil {
		return m.GetAPIV1ResourceInstancesMock(queryParams, URL)
	}
	return nil, nil
}

// GetAPIServiceByName -
func (m *Client) GetAPIServiceByName(serviceName string) (*management.APIService, error) {
	if m.GetAPIServiceByNameMock != nil {
		return m.GetAPIServiceByNameMock(serviceName)
	}
	return nil, nil
}

// GetAPIRevisionByName -
func (m *Client) GetAPIRevisionByName(revisionName string) (*management.APIServiceRevision, error) {
	if m.GetAPIRevisionByNameMock != nil {
		return m.GetAPIRevisionByNameMock(revisionName)
	}
	return nil, nil
}

// GetAPIServiceInstanceByName -
func (m *Client) GetAPIServiceInstanceByName(instanceName string) (*management.APIServiceInstance, error) {
	if m.GetAPIServiceInstanceByNameMock != nil {
		return m.GetAPIServiceInstanceByNameMock(instanceName)
	}
	return nil, nil
}

// SetTokenGetter -
func (m *Client) SetTokenGetter(tokenGetter auth.PlatformTokenGetter) {
	if m.SetTokenGetterMock != nil {
		m.SetTokenGetterMock(tokenGetter)
	}
}

// PublishService -
func (m *Client) PublishService(serviceBody *apic.ServiceBody) (*management.APIService, error) {
	if m.PublishServiceMock != nil {
		return m.PublishServiceMock(serviceBody)
	}
	return nil, nil
}

// DeleteServiceByName -
func (m *Client) DeleteServiceByName(serviceName string) error {
	if m.DeleteServiceByNameMock != nil {
		return m.DeleteServiceByNameMock(serviceName)
	}
	return nil
}

// DeleteAPIServiceInstance -
func (m *Client) DeleteAPIServiceInstance(instanceName string) error {
	if m.DeleteAPIServiceInstanceMock != nil {
		return m.DeleteAPIServiceInstanceMock(instanceName)
	}
	return nil
}

// GetUserName -
func (m *Client) GetUserName(ID string) (string, error) {
	if m.GetUserNameMock != nil {
		return m.GetUserNameMock(ID)
	}
	return "", nil
}

// GetUserEmailAddress -
func (m *Client) GetUserEmailAddress(ID string) (string, error) {
	if m.GetUserEmailAddressMock != nil {
		return m.GetUserEmailAddressMock(ID)
	}
	return "", nil
}

// Healthcheck -
func (m *Client) Healthcheck(name string) *hc.Status {
	if m.HealthcheckMock != nil {
		return m.HealthcheckMock(name)
	}
	return &hc.Status{Result: hc.OK}
}

// ExecuteAPI -
func (m *Client) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	if m.ExecuteAPIMock != nil {
		return m.ExecuteAPIMock(method, url, queryParam, buffer)
	}
	return nil, nil
}

// SetConfig -
func (m *Client) SetConfig(cfg corecfg.CentralConfig) {
	if m.SetConfigMock != nil {
		m.SetConfigMock(cfg)
	}
}

// GetTeam -
func (m *Client) GetTeam(queryParams map[string]string) ([]definitions.PlatformTeam, error) {
	if m.GetTeamMock != nil {
		return m.GetTeamMock(queryParams)
	}
	return nil, nil
}

// GetAccessControlList -
func (m *Client) GetAccessControlList(aclName string) (*management.AccessControlList, error) {
	if m.GetAccessControlListMock != nil {
		return m.GetAccessControlListMock(aclName)
	}
	return nil, nil
}

// UpdateAccessControlList -
func (m *Client) UpdateAccessControlList(acl *management.AccessControlList) (*management.AccessControlList, error) {
	if m.UpdateAccessControlListMock != nil {
		return m.UpdateAccessControlListMock(acl)
	}
	return nil, nil
}

// CreateAccessControlList -
func (m *Client) CreateAccessControlList(acl *management.AccessControlList) (*management.AccessControlList, error) {
	if m.CreateAccessControlListMock != nil {
		return m.CreateAccessControlListMock(acl)
	}
	return nil, nil
}

// UpdateResourceInstance -
func (m *Client) UpdateResourceInstance(ri v1.Interface) (*v1.ResourceInstance, error) {
	if m.UpdateResourceInstanceMock != nil {
		return m.UpdateResourceInstanceMock(ri)
	}
	return nil, nil
}

// CreateResourceInstance -
func (m *Client) CreateResourceInstance(ri v1.Interface) (*v1.ResourceInstance, error) {
	if m.CreateResourceInstanceMock != nil {
		return m.CreateResourceInstanceMock(ri)
	}
	return nil, nil
}

// PatchSubResource -
func (m *Client) PatchSubResource(ri v1.Interface, subResourceName string, patches []map[string]interface{}) (*v1.ResourceInstance, error) {
	if m.PatchSubResourceMock != nil {
		return m.PatchSubResourceMock(ri, subResourceName, patches)
	}
	return nil, nil
}

// DeleteResourceInstance -
func (m *Client) DeleteResourceInstance(ri v1.Interface) error {
	if m.DeleteResourceInstanceMock != nil {
		return m.DeleteResourceInstanceMock(ri)
	}
	return nil
}

// CreateSubResource -
func (m *Client) CreateSubResource(rm v1.ResourceMeta, subs map[string]interface{}) error {
	if m.CreateSubResourceMock != nil {
		return m.CreateSubResourceMock(rm, subs)
	}
	return nil
}

// GetResource -
func (m *Client) GetResource(url string) (*v1.ResourceInstance, error) {
	if m.GetResourceMock != nil {
		return m.GetResourceMock(url)
	}
	return nil, nil
}

// GetResources -
func (m *Client) GetResources(ri v1.Interface) ([]v1.Interface, error) {
	if m.GetResourcesMock != nil {
		return m.GetResourcesMock(ri)
	}
	return nil, nil
}

// CreateResource -
func (m *Client) CreateResource(url string, bts []byte) (*v1.ResourceInstance, error) {
	if m.CreateResourceMock != nil {
		return m.CreateResourceMock(url, bts)
	}
	return nil, nil
}

// UpdateResource -
func (m *Client) UpdateResource(url string, bts []byte) (*v1.ResourceInstance, error) {
	if m.UpdateResourceMock != nil {
		return m.UpdateResourceMock(url, bts)
	}
	return nil, nil
}

// UpdateResourceFinalizer -
func (m *Client) UpdateResourceFinalizer(res *v1.ResourceInstance, finalizer, description string, addAction bool) (*v1.ResourceInstance, error) {
	if m.UpdateResourceFinalizerMock != nil {
		return m.UpdateResourceFinalizerMock(res, finalizer, description, addAction)
	}
	return nil, nil
}

// CreateOrUpdateResource -
func (m *Client) CreateOrUpdateResource(iface v1.Interface) (*v1.ResourceInstance, error) {
	if m.CreateOrUpdateResourceMock != nil {
		return m.CreateOrUpdateResourceMock(iface)
	}
	return nil, nil
}
