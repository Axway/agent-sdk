package agent

import (
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
	instanceTags            []string
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
		instance.SetTags(append(instance.GetTags(), info.instanceTags...))

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
		GetAPIV1ResourceInstancesMock: func(queryParam map[string]string, url string) ([]*v1.ResourceInstance, error) {
			getCalls++
			ris := []*v1.ResourceInstance{}
			param := queryParam["query"]
			splits := strings.Split(param, `name=="`)
			for i := 1; i < len(splits); i++ {
				name := strings.Trim(strings.Trim(strings.TrimSpace(splits[i]), `" or `), `"`)
				if strings.HasSuffix(url, "apiserviceinstances") {
					res, _ := agent.cacheManager.GetAPIServiceInstanceByName(name)
					if res != nil {
						ris = append(ris, res)
					}
				} else if strings.HasSuffix(url, "apiservices") {
					res := agent.cacheManager.GetAPIServiceWithName(name)
					if res != nil {
						ris = append(ris, res)
					}
				}
			}
			if len(ris) == 0 {
				return ris, fmt.Errorf("not found. query: %s", param)
			}
			return ris, nil
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
			name: "2 queries, 2 tagged services: 2 instances have agent tag, 0 tagged instances: tag already existing for 2, one missing externalAPIID",
			cachedInfo: []info{
				{
					instanceName:            "exquisite-instance1",
					serviceName:             "exquisite-service1",
					externalAPIIDidService:  "exquisite-id1",
					externalAPIIDidInstance: "exquisite-id1",
					instanceTags:            []string{util.AgentWarningTag},
				},
				{
					instanceName:            "exquisite-instance2",
					serviceName:             "exquisite-service2",
					externalAPIIDidService:  "exquisite-id2",
					externalAPIIDidInstance: "exquisite-id2",
					instanceTags:            []string{util.AgentWarningTag},
				},
				{
					instanceName: "exquisite-instance3",
					serviceName:  "exquisite-service3",
				},
			},
			expectedGetCalls:               2,
			maxQueryParamLength:            1,
			expectedServiceNamesToBeTagged: []string{"exquisite-service1", "exquisite-service2"},
		},
		{
			name: "1 query, 0 tagged services, 3 tagged instances",
			cachedInfo: []info{
				{
					instanceName:            "exquisite-instance1",
					serviceName:             "exquisite-service1",
					externalAPIIDidInstance: "exquisite-instance-id1",
				},
				{
					instanceName:            "exquisite-instance2",
					serviceName:             "exquisite-service2",
					externalAPIIDidInstance: "exquisite-instance-id2",
				},
				{
					instanceName:            "exquisite-instance3",
					serviceName:             "exquisite-service3",
					externalAPIIDidInstance: "exquisite-instance-id3",
				},
			},
			expectedGetCalls:                1,
			expectedInstanceNamesToBeTagged: []string{"exquisite-instance1", "exquisite-instance2", "exquisite-instance3"},
		},
		{
			name: "6 queries, 3 tagged services, 3 tagged instances",
			cachedInfo: []info{
				{
					instanceName:            "exquisite-instance1",
					serviceName:             "exquisite-service1",
					externalAPIIDidInstance: "exquisite-api-id1",
					externalAPIIDidService:  "exquisite-api-id1",
				},
				{
					instanceName:            "exquisite-instance2",
					serviceName:             "exquisite-service2",
					externalAPIIDidInstance: "exquisite-api-id2",
					externalAPIIDidService:  "exquisite-api-id2",
				},
				{
					instanceName:            "exquisite-instance3",
					serviceName:             "exquisite-service3",
					externalAPIIDidInstance: "exquisite-api-id3",
					externalAPIIDidService:  "exquisite-api-id3",
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
				assert.True(t, util.IsInArray(res.GetTags(), util.AgentWarningTag))
			}
			for _, svcName := range tc.expectedServiceNamesToBeTagged {
				res := agent.cacheManager.GetAPIServiceWithName(svcName)
				assert.True(t, util.IsInArray(res.GetTags(), util.AgentWarningTag))
			}
			assert.Equal(t, tc.expectedGetCalls, getCalls)
		})
	}
}
