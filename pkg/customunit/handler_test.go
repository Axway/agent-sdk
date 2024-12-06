package customunit

import (
	"log"
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const instRefID = "inst-id-1"
const instRefName = "inst-name-1"
const managedAppRefName = "managed-app-name"

var testServiceConfig = []config.MetricServiceConfiguration{
	{
		Enable:       true,
		URL:          "bufnet",
		RejectOnFail: false,
	},
}

func Test_NewCustomUnitHandler(t *testing.T) {
	metricServicesConfigs := config.NewAgentFeaturesConfiguration().GetMetricServicesConfigs()
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewCustomUnitHandler(metricServicesConfigs, cm, config.DiscoveryAgent)

	assert.NotNil(t, handler)
}

type fakeMetricCollector struct {
	expected int
	details  []models.CustomMetricDetail
	done     chan struct{}
}

func (c *fakeMetricCollector) AddCustomMetricDetail(detail models.CustomMetricDetail) {
	if c.details == nil {
		c.details = []models.CustomMetricDetail{}
	}
	c.details = append(c.details, detail)
	if len(c.details) == c.expected {
		c.done <- struct{}{}
	}
}

type mockAgentCache struct {
	apiNameMap    map[string]*v1.ResourceInstance
	apiIDMap      map[string]*v1.ResourceInstance
	instanceIDMap map[string]*v1.ResourceInstance
	apiExtIDMap   map[string]*v1.ResourceInstance
	apiKeys       []string
	appNameMap    map[string]*v1.ResourceInstance
	appIDMap      map[string]*v1.ResourceInstance
	appKeys       []string
}

func (c *mockAgentCache) addAPISvc(name, id, extID, extPK, extName string) {
	apis := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: "svc-" + name,
			Metadata: v1.Metadata{
				ID: id,
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID:         id,
					defs.AttrExternalAPIPrimaryKey: extPK,
					defs.AttrExternalAPIName:       extName,
				},
			},
		},
	}

	if len(c.apiKeys) == 0 {
		c.apiNameMap = map[string]*v1.ResourceInstance{}
		c.apiIDMap = map[string]*v1.ResourceInstance{}
		c.apiExtIDMap = map[string]*v1.ResourceInstance{}
		c.instanceIDMap = map[string]*v1.ResourceInstance{}
		c.apiKeys = []string{}
	}

	c.apiNameMap[name] = apis
	c.apiIDMap[id] = apis
	c.apiExtIDMap[extID] = apis
	c.apiKeys = append(c.apiKeys, id)

	c.instanceIDMap[id] = &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: "instance",
			Metadata: v1.Metadata{
				ID: "inst-" + id,
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID:         id,
					defs.AttrExternalAPIPrimaryKey: "primary-" + id,
					defs.AttrExternalAPIName:       "ext" + name,
				},
			},
		},
	}
}

func (c *mockAgentCache) GetAPIServiceWithAPIID(id string) *v1.ResourceInstance {
	if ri, ok := c.apiExtIDMap[id]; ok {
		return ri
	}
	return c.apiIDMap[id]
}

func (c *mockAgentCache) GetAPIServiceWithName(name string) *v1.ResourceInstance {
	return c.apiNameMap[name]
}

func (c *mockAgentCache) GetAPIServiceInstanceByID(id string) (*v1.ResourceInstance, error) {
	return c.apiExtIDMap[id], nil
}

func (c *mockAgentCache) GetAPIServiceKeys() []string {
	return c.apiKeys
}

func (c *mockAgentCache) addApp(name, id, extID, extName string) {
	app := &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAppID: extID,
					"applicationName":      extName,
				},
			},
		},
	}

	if len(c.appKeys) == 0 {
		c.appNameMap = map[string]*v1.ResourceInstance{}
		c.appIDMap = map[string]*v1.ResourceInstance{}
		c.appKeys = []string{}
	}

	c.appNameMap[name] = app
	c.appIDMap[id] = app
	c.appKeys = append(c.appKeys, id)
}

func (c *mockAgentCache) GetManagedApplication(id string) *v1.ResourceInstance {
	return c.appIDMap[id]
}

func (c *mockAgentCache) GetManagedApplicationByName(name string) *v1.ResourceInstance {
	return c.appNameMap[name]
}

func (c *mockAgentCache) GetManagedApplicationCacheKeys() []string {
	return c.appKeys
}

func Test_HandleQuotaEnforcementInfo(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	customunits.RegisterQuotaEnforcementServer(s, customunits.UnimplementedQuotaEnforcementServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

	// setup the api service instance
	apisi := management.NewAPIServiceInstance(instRefName, "env-1")
	apisi.Metadata.ID = instRefID
	apisi.Metadata.References = []v1.Reference{
		{
			Name:  instRefName,
			Kind:  management.APIServiceGVK().Kind,
			Group: management.APIServiceGVK().Group,
		},
	}
	apisiRI, _ := apisi.AsInstance()
	cm.AddAPIServiceInstance(apisiRI)

	// setup the api service
	apis := management.NewAPIService(instRefName, "env-1")
	util.SetAgentDetailsKey(apis, definitions.AttrExternalAPIID, instRefID)
	apisRI, _ := apis.AsInstance()
	cm.AddAPIService(apisRI)

	accessReq := &management.AccessRequest{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: "11",
				References: []v1.Reference{
					{
						Group: management.APIServiceInstanceGVK().Group,
						Kind:  management.APIServiceInstanceGVK().Kind,
						ID:    instRefID,
						Name:  instRefName,
					},
				},
				Scope: v1.MetadataScope{
					Kind: management.EnvironmentGVK().Kind,
					Name: "env-1",
				},
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					"sub_access_request_key": "sub_access_request_val",
				},
			},
		},
		Spec: management.AccessRequestSpec{
			ApiServiceInstance: instRefName,
			ManagedApplication: managedAppRefName,
			Data:               map[string]interface{}{},
		},
		Status: &v1.ResourceStatus{
			Level: prov.Pending.String(),
		},
	}
	managedAppForTest := &management.ManagedApplication{
		ResourceMeta: v1.ResourceMeta{
			Name: "app-test",
			Metadata: v1.Metadata{
				ID: "11",
				Scope: v1.MetadataScope{
					Kind: management.EnvironmentGVK().Kind,
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
		},
		Spec: management.ManagedApplicationSpec{},
		Status: &v1.ResourceStatus{
			Level: prov.Pending.String(),
		},
	}

	handler := NewCustomUnitHandler(testServiceConfig, cm, config.DiscoveryAgent)
	err := handler.HandleQuotaEnforcement(accessReq, managedAppForTest)

	assert.Nil(t, err)
}

func Test_HandleMetricReporting(t *testing.T) {
	metricCollector := &fakeMetricCollector{expected: 0, done: make(chan struct{})}
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewCustomUnitHandler(testServiceConfig, cm, config.TraceabilityAgent)

	go handler.HandleMetricReporting(metricCollector)
	handler.Stop()
}

type reportData struct {
	apiLookup     customunits.APIServiceLookupType
	apiLookupVal  string
	apiLookupAttr string
	appLookup     customunits.AppLookupType
	appLookupVal  string
	appLookupAttr string
}

func GetMetricReport(report reportData) *customunits.MetricReport {
	return &customunits.MetricReport{
		ApiService: &customunits.APIServiceLookup{
			Type:            report.apiLookup,
			Value:           report.apiLookupVal,
			CustomAttribute: report.apiLookupAttr,
		},
		ManagedApp: &customunits.AppLookup{
			Type:            report.appLookup,
			Value:           report.appLookupVal,
			CustomAttribute: report.appLookupAttr,
		},
		PlanUnit: &customunits.UnitLookup{
			UnitName: "x-ai-tokens",
		},
		Count: 1,
	}
}

func TestCustomMetricService(t *testing.T) {
	cache := &mockAgentCache{}
	cache.addAPISvc("api1", "id-api1", "ext-id-api1", "pk-api1", "name-api1")
	cache.addAPISvc("api2", "id-api2", "ext-id-api2", "pk-api2", "name-api2")
	cache.addAPISvc("api3", "id-api3", "ext-id-api3", "pk-api3", "name-api3")
	cache.addAPISvc("api4", "id-api4", "ext-id-api4", "pk-api4", "name-api4")
	cache.addAPISvc("api5", "id-api5", "ext-id-api5", "pk-api5", "name-api5")
	cache.addApp("app1", "id-app1", "ext-id-app1", "ext-name-app1")
	cache.addApp("app2", "id-app2", "ext-id-app2", "ext-name-app2")
	cache.addApp("app3", "id-app3", "ext-id-app3", "ext-name-app3")
	cache.addApp("app4", "id-app4", "ext-id-app4", "ext-name-app4")
	cache.addApp("app5", "id-app5", "ext-id-app5", "ext-name-app5")
	handler := NewCustomUnitHandler(testServiceConfig, cache, config.TraceabilityAgent)

	testCases := map[string]struct {
		skip           bool
		reports        []reportData
		waitForReports int
	}{
		"lookup with service name and external id": {
			skip: false,
			reports: []reportData{
				{customunits.APIServiceLookupType_ServiceName, "api1", "", customunits.AppLookupType_ExternalAppID, "ext-id-app1", ""},
			},
			waitForReports: 1,
		},
		"lookup with service id and managed app id": {
			skip: false,
			reports: []reportData{
				{customunits.APIServiceLookupType_ServiceID, "id-api2", "", customunits.AppLookupType_ManagedAppID, "id-app2", ""},
			},
			waitForReports: 1,
		},
		"lookup with external api id and managed app name": {
			skip: false,
			reports: []reportData{
				{customunits.APIServiceLookupType_ExternalAPIID, "ext-id-api3", "", customunits.AppLookupType_ManagedAppName, "app3", ""},
			},
			waitForReports: 1,
		},
		"lookup with custom api attr and custom app attr": {
			skip: false,
			reports: []reportData{
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "pk-api4", definitions.AttrExternalAPIPrimaryKey, customunits.AppLookupType_CustomAppLookup, "ext-name-app4", "applicationName"},
			},
			waitForReports: 1,
		},
		"multiple reports with custom lookups": {
			skip: false,
			reports: []reportData{
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api1", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app1", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api2", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app2", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api3", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app3", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api4", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app4", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api5", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app5", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api1", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app1", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api2", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app2", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api3", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app3", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api4", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app4", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api5", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app5", "applicationName"},
			},
			waitForReports: 10,
		},
		"send 10 reports, 8 with bad values": {
			skip: false,
			reports: []reportData{
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api1", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app1", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api6", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app2", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api3", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app6", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api4", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app4", "applicationName1"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api5", "does-not-exist", customunits.AppLookupType_CustomAppLookup, "ext-name-app5", "applicationName"},
				{customunits.APIServiceLookupType_ExternalAPIID, "name-api1", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app1", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api2", "", customunits.AppLookupType_ManagedAppID, "ext-name-app2", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api3", definitions.AttrExternalAPIName, customunits.AppLookupType_ManagedAppName, "ext-name-app3", "applicationName"},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api4", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app4", ""},
				{customunits.APIServiceLookupType_CustomAPIServiceLookup, "name-api5", definitions.AttrExternalAPIName, customunits.AppLookupType_CustomAppLookup, "ext-name-app5", "applicationName"},
			},
			waitForReports: 2,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				return
			}
			collector := &fakeMetricCollector{expected: tc.waitForReports, done: make(chan struct{})}
			go handler.receiveMetrics(collector)
			defer handler.Stop()

			for _, r := range tc.reports {
				handler.metricReportChan <- GetMetricReport(r)
			}

			<-collector.done
			assert.Len(t, collector.details, tc.waitForReports)
		})
	}
}
