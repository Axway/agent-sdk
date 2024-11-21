package customunit

import (
	"log"
	"math/rand"
	"testing"
	"time"

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

func Test_NewCustomUnitHandler(t *testing.T) {
	metricServicesConfigs := config.NewAgentFeaturesConfiguration().GetMetricServicesConfigs()
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	handler := NewCustomUnitHandler(metricServicesConfigs, cm, config.DiscoveryAgent)

	assert.NotNil(t, handler)
}

type fakeMetricCollector struct {
}

type mockAgentCache struct {
}

func (c *mockAgentCache) GetAPIServiceWithAPIID(id string) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: "service",
			Metadata: v1.Metadata{
				ID: "svc-" + "fsdfsf2342r2ferge",
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID:         "fsdfsf2342r2ferge",
					defs.AttrExternalAPIPrimaryKey: "primary-" + "fsdfsf2342r2ferge",
					defs.AttrExternalAPIName:       "test",
				},
			},
		},
	}
}

func (c *mockAgentCache) GetAPIServiceWithName(name string) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: "service",
			Metadata: v1.Metadata{
				ID: "svc-" + "fsdfsf2342r2ferge",
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID:         "fsdfsf2342r2ferge",
					defs.AttrExternalAPIPrimaryKey: "primary-" + "fsdfsf2342r2ferge",
					defs.AttrExternalAPIName:       "test",
				},
			},
		},
	}
}

func (c *mockAgentCache) GetAPIServiceInstanceByID(string) (*v1.ResourceInstance, error) {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name: "instance",
			Metadata: v1.Metadata{
				ID: "instance-" + "fsdfsf2342r2ferge",
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID:         "fsdfsf2342r2ferge",
					defs.AttrExternalAPIPrimaryKey: "primary-" + "fsdfsf2342r2ferge",
					defs.AttrExternalAPIName:       "test",
				},
			},
		},
	}, nil
}

func (c *mockAgentCache) GetAPIServiceKeys() []string {
	keys := make([]string, 0)
	return keys
}
func (c *mockAgentCache) GetManagedApplication(string) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: "fsdfsdfsf234235fgdgd",
			},
			Name: "app",
		},
	}
}
func (c *mockAgentCache) GetManagedApplicationByName(string) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: "fsdfsdfsf234235fgdgd",
			},
			Name: "app",
		},
	}
}

func (c *mockAgentCache) GetManagedApplicationCacheKeys() []string {
	keys := make([]string, 0)
	return keys
}

func (c *fakeMetricCollector) AddCustomMetricDetail(detail models.CustomMetricDetail) {

}

func GetMetricReport() *customunits.MetricReport {
	apiID, appID, unit := "fsdfsf2342r2ferge", "fsdfsdfsf234235fgdgd", "x-ai-tokens"
	count := rand.Int63n(50)
	metricReport := &customunits.MetricReport{
		ApiService: &customunits.APIServiceLookup{
			Type:  customunits.APIServiceLookupType_ExternalAPIID,
			Value: apiID,
		},
		ManagedApp: &customunits.AppLookup{
			Type:  customunits.AppLookupType_ManagedAppID,
			Value: appID,
		},
		PlanUnit: &customunits.UnitLookup{
			UnitName: unit,
		},
		Count: count,
	}

	return metricReport
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

	metricServicesConfigs := []config.MetricServiceConfiguration{
		{
			Enable:       true,
			URL:          "bufnet",
			RejectOnFail: false,
		},
	}
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

	handler := NewCustomUnitHandler(metricServicesConfigs, cm, config.DiscoveryAgent)
	err := handler.HandleQuotaEnforcement(accessReq, managedAppForTest)

	assert.Nil(t, err)
}

func Test_HandleMetricReporting(t *testing.T) {

	metricCollector := &fakeMetricCollector{}
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	metricServicesConfigs := []config.MetricServiceConfiguration{
		{
			Enable:       true,
			URL:          "bufnet",
			RejectOnFail: true,
		},
	}

	handler := NewCustomUnitHandler(metricServicesConfigs, cm, config.TraceabilityAgent)

	go handler.HandleMetricReporting(metricCollector)
	time.Sleep(10 * time.Second)
	handler.Stop()
}

func Test_ReceiveMetrics(t *testing.T) {
	metricReport := GetMetricReport()
	metricServicesConfigs := []config.MetricServiceConfiguration{
		{
			Enable:       true,
			URL:          "bufnet",
			RejectOnFail: true,
		},
	}
	handler := NewCustomUnitHandler(metricServicesConfigs, &mockAgentCache{}, config.TraceabilityAgent)

	go handler.receiveMetrics(&fakeMetricCollector{})
	handler.metricReportChan <- metricReport
	time.Sleep(15 * time.Second)
	handler.Stop()
}

func Test_BuildCustomMetricDetail(t *testing.T) {
	metricReport := GetMetricReport()
	metricServicesConfigs := []config.MetricServiceConfiguration{
		{
			Enable:       true,
			URL:          "bufnet",
			RejectOnFail: true,
		},
	}
	handler := NewCustomUnitHandler(metricServicesConfigs, &mockAgentCache{}, config.TraceabilityAgent)

	customMetricDetail, err := handler.buildCustomMetricDetail(metricReport)

	assert.Nil(t, err)
	assert.NotNil(t, customMetricDetail)
	assert.Equal(t, customMetricDetail.APIDetails.ID, "remoteApiId_fsdfsf2342r2ferge")
	assert.Equal(t, customMetricDetail.AppDetails.ID, "fsdfsdfsf234235fgdgd")
	assert.Equal(t, customMetricDetail.UnitDetails.Name, "x-ai-tokens")
}
