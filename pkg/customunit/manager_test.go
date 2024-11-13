package customunit

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
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

func Test_HandleQuotaEnforcementInfo(t *testing.T) {
	metricServicesConfigs := []config.MetricServiceConfiguration{
		{
			Enable:       true,
			URL:          "https://mockserver:8080",
			RejectOnFail: true,
		},
	}
	cm := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
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

	manager := NewCustomUnitHandler(metricServicesConfigs, cm, config.DiscoveryAgent)
	err := manager.HandleQuotaEnforcement(accessReq, managedAppForTest)

	assert.Nil(t, err)
}
