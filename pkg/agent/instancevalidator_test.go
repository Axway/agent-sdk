package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/util"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

type info struct {
	instanceName            string
	serviceName             string
	externalAPIIDidInstance string
	externalAPINameInstance string
	externalAPIIDidService  string
	externalAPINameService  string
}

func setupCache(apiInfos []info) ([]*v1.ResourceInstance, []*v1.ResourceInstance) {
	agent.cacheManager = agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	services := []*v1.ResourceInstance{}
	instances := []*v1.ResourceInstance{}
	for _, info := range apiInfos {
		svc := &v1.ResourceInstance{
			ResourceMeta: v1.ResourceMeta{
				GroupVersionKind: management.APIServiceGVK(),
				Name:             info.serviceName,
				SubResources: map[string]interface{}{
					definitions.XAgentDetails: map[string]interface{}{},
				},
			},
		}
		instance := &v1.ResourceInstance{
			ResourceMeta: v1.ResourceMeta{
				GroupVersionKind: management.APIServiceInstanceGVK(),
				Name:             info.instanceName,
				SubResources: map[string]interface{}{
					definitions.XAgentDetails: map[string]interface{}{},
				},
			},
		}

		if info.externalAPIIDidService != "" {
			svc.SubResources[definitions.XAgentDetails].(map[string]interface{})[definitions.AttrExternalAPIID] = info.externalAPIIDidService
			svc.SubResources[definitions.XAgentDetails].(map[string]interface{})[definitions.AttrExternalAPIPrimaryKey] = "primary-" + info.externalAPIIDidService
			svc.Metadata.ID = fmt.Sprintf("svc-%s", info.externalAPIIDidService)
		}
		if info.externalAPINameService != "" {
			svc.SubResources[definitions.XAgentDetails].(map[string]interface{})[definitions.AttrExternalAPIName] = info.externalAPINameService
		}
		if info.externalAPIIDidInstance != "" {
			instance.SubResources[definitions.XAgentDetails].(map[string]interface{})[definitions.AttrExternalAPIID] = info.externalAPIIDidInstance
			instance.SubResources[definitions.XAgentDetails].(map[string]interface{})[definitions.AttrExternalAPIPrimaryKey] = "primary-" + info.externalAPIIDidInstance
			instance.Metadata.ID = fmt.Sprintf("svc-%s", info.externalAPIIDidInstance)
		}
		if info.externalAPINameInstance != "" {
			instance.SubResources[definitions.XAgentDetails].(map[string]interface{})[definitions.AttrExternalAPIName] = info.externalAPINameInstance
		}

		agent.cacheManager.AddAPIService(svc)
		agent.cacheManager.AddAPIServiceInstance(instance)
		services = append(services, svc)
		instances = append(instances, instance)
	}
	return services, instances
}

func setupAPIValidator(apiValidation bool) {
	setAPIValidator(func(apiID, stageName string) bool {
		return apiValidation
	})
}

func TestValidatorAPI(t *testing.T) {
	getCalls := 0
	mockClient := &mock.Client{
		GetAPIServiceInstancesMock: func(queryParam map[string]string, url string) ([]*management.APIServiceInstance, error) {
			getCalls++
			instances := []*management.APIServiceInstance{}
			param := queryParam["query"]
			splits := strings.Split(param, `name=="`)
			for i := 1; i < len(splits); i++ {
				name := strings.Trim(strings.Trim(strings.TrimSpace(splits[i]), "or"), `"`)
				res, _ := agent.cacheManager.GetAPIServiceInstanceByName(name)
				if res != nil {
					apisi := &management.APIServiceInstance{}
					apisi.FromInstance(res)
					instances = append(instances, apisi)
				}
			}
			if len(instances) == 0 {
				return instances, fmt.Errorf("not found. query: %s", param)
			}
			return instances, nil
		},
		GetAPIServicesMock: func(queryParam map[string]string, url string) ([]*management.APIService, error) {
			getCalls++
			services := []*management.APIService{}
			param := queryParam["query"]
			splits := strings.Split(param, `name=="`)
			for i := 1; i < len(splits); i++ {
				name := strings.Trim(strings.Trim(strings.TrimSpace(splits[i]), `" or `), `"`)
				res := agent.cacheManager.GetAPIServiceWithName(name)
				if res != nil {
					api := &management.APIService{}
					api.FromInstance(res)
					services = append(services, api)
				}
			}
			if len(services) == 0 {
				return services, fmt.Errorf("not found. query: %s", param)
			}
			return services, nil
		},
		UpdateResourceInstanceMock: func(ri v1.Interface) (*v1.ResourceInstance, error) {
			switch ri.GetGroupVersionKind().Kind {
			case management.APIServiceGVK().Kind:
				apiRI, _ := ri.AsInstance()
				agent.cacheManager.AddAPIService(apiRI)
				return apiRI, nil
			case management.APIServiceInstanceGVK().Kind:
				instRI, _ := ri.AsInstance()
				agent.cacheManager.AddAPIServiceInstance(instRI)
				return instRI, nil
			}
			return nil, fmt.Errorf("not found id: %s", ri.GetMetadata().ID)
		},
	}
	cases := []struct {
		name                            string
		cachedInfo                      []info
		maxQueryParamLength             int
		apiValidation                   bool
		expectedInstanceNamesToBeTagged []string
		expectedServiceNamesToBeTagged  []string
		expectedGetCalls                int
	}{
		{
			name:          "no queries, no tags, validator always true",
			apiValidation: true,
		},
		{
			name: "1 query, 3 tagged services, 0 tagged instances: missing externalAPIID",
			cachedInfo: []info{
				{
					instanceName:            "exquisite-instance1",
					serviceName:             "exquisite-service1",
					externalAPIIDidService:  "exquisite-service-id1",
					externalAPINameInstance: "test1",
					externalAPINameService:  "test1",
				},
				{
					instanceName:            "exquisite-instance2",
					serviceName:             "exquisite-service2",
					externalAPIIDidService:  "exquisite-service-id2",
					externalAPINameInstance: "test2",
					externalAPINameService:  "test2",
				},
				{
					instanceName:            "exquisite-instance3",
					serviceName:             "exquisite-service3",
					externalAPIIDidService:  "exquisite-service-id3",
					externalAPINameInstance: "test3",
					externalAPINameService:  "test3",
				},
			},
			expectedGetCalls:               1,
			expectedServiceNamesToBeTagged: []string{"exquisite-service1", "exquisite-service2", "exquisite-service3"},
		},
		{
			name: "3 queries, 3 tagged services, 3 tagged instances",
			cachedInfo: []info{
				{
					instanceName:            "exquisite-instance1",
					serviceName:             "exquisite-service1",
					externalAPIIDidInstance: "exquisite-instance-id1",
					externalAPIIDidService:  "exquisite-service-id1",
					externalAPINameInstance: "test1",
					externalAPINameService:  "test1",
				},
				{
					instanceName:            "exquisite-instance2",
					serviceName:             "exquisite-service2",
					externalAPIIDidInstance: "exquisite-instance-id2",
					externalAPIIDidService:  "exquisite-service-id2",
					externalAPINameInstance: "test2",
					externalAPINameService:  "test2",
				},
				{
					instanceName:            "exquisite-instance3",
					serviceName:             "exquisite-service3",
					externalAPIIDidInstance: "exquisite-instance-id3",
					externalAPIIDidService:  "exquisite-service-id3",
					externalAPINameInstance: "test3",
					externalAPINameService:  "test3",
				},
			},
			maxQueryParamLength:             1,
			expectedInstanceNamesToBeTagged: []string{"exquisite-instance1", "exquisite-instance2", "exquisite-instance3"},
			expectedServiceNamesToBeTagged:  []string{"exquisite-service1", "exquisite-service2", "exquisite-service3"},
			expectedGetCalls:                6,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getCalls = 0
			instanceValidator := newInstanceValidator()
			if tc.maxQueryParamLength != 0 {
				instanceValidator.maxQueryParamLength = tc.maxQueryParamLength
			}
			setupCache(tc.cachedInfo)
			setupAPIValidator(tc.apiValidation)
			agent.apicClient = mockClient

			err := instanceValidator.Execute()
			assert.Nil(t, err)

			for _, instName := range tc.expectedInstanceNamesToBeTagged {
				res, _ := agent.cacheManager.GetAPIServiceInstanceByName(instName)
				assert.True(t, util.IsInArray(res.GetTags(), agentWarningTag))
			}
			for _, svcName := range tc.expectedServiceNamesToBeTagged {
				res := agent.cacheManager.GetAPIServiceWithName(svcName)
				assert.True(t, util.IsInArray(res.GetTags(), agentWarningTag))
			}
			assert.Equal(t, tc.expectedGetCalls, getCalls)
		})
	}
}

func getCachedData(id, kind string) string {
	switch kind {
	case management.APIServiceInstanceGVK().Kind:
		d, _ := agent.cacheManager.GetAPIServiceInstanceByID(id)
		b, _ := json.Marshal(d)
		return string(b)
	case management.APIServiceGVK().Kind:
		d := agent.cacheManager.GetAPIServiceWithPrimaryKey(id)
		b, _ := json.Marshal(d)
		return string(b)
	}
	return ""
}
