package compliance

import (
	"errors"
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

type mockProcessor struct {
	results       []RuntimeResult
	collectCalled bool
}

func (m *mockProcessor) CollectRuntimeResult(results RuntimeResults) error {
	m.collectCalled = true
	for _, result := range m.results {
		results.AddRuntimeResult(result)
	}
	return nil
}

func TestCompliance(t *testing.T) {
	apicMockCli := &mock.Client{}
	resReqCount := 0
	subResReqCount := 0

	apicMockCli.CreateSubResourceMock = func(rm v1.ResourceMeta, subs map[string]interface{}) error {
		if rm.GetGroupVersionKind().Kind == management.APIServiceInstanceGVK().Kind {
			ri, err := agent.GetCacheManager().GetAPIServiceInstanceByName(rm.Name)
			if ri == nil {
				return err
			}
			ri.SubResources = subs
			agent.GetCacheManager().AddAPIServiceInstance(ri)
			subResReqCount++
		} else if rm.GetGroupVersionKind().Kind == management.EnvironmentGVK().Kind {
			ri := agent.GetCacheManager().GetWatchResourceByName(management.EnvironmentGVK().Group, management.EnvironmentGVK().Kind, rm.Name)
			if ri == nil {
				return errors.New("err")
			}
			ri.SubResources = subs
			agent.GetCacheManager().AddWatchResource(ri)
			subResReqCount++
		} else if rm.GetGroupVersionKind().Kind == management.APIServiceGVK().Kind {
			ri := agent.GetCacheManager().GetWatchResourceByName(management.APIServiceGVK().Group, management.APIServiceGVK().Kind, rm.Name)
			if ri == nil {
				return errors.New("err")
			}
			ri.SubResources = subs
			agent.GetCacheManager().AddWatchResource(ri)
			subResReqCount++
		}

		return nil
	}

	apicMockCli.CreateOrUpdateResourceMock = func(ri v1.Interface) (*v1.ResourceInstance, error) {
		resInst, _ := ri.AsInstance()
		crr := management.NewComplianceRuntimeResult("", "")
		crr.FromInstance(resInst)
		resReqCount++

		agent.GetCacheManager().AddComplianceRuntimeResult(resInst)
		return resInst, nil
	}

	tests := []struct {
		name              string
		runtimeResults    []RuntimeResult
		existingResults   []interface{}
		expectedAgentName string
		expectedAgentType string
		expectedTimestamp string
		skipAddingCRR     bool
	}{
		{
			name: "no collection",
		},
		{
			name: "collect and publish, 1 result",
			runtimeResults: []RuntimeResult{
				{
					ComplianceRuntimeResult: "test-1",
					RiskScore:               10,
				},
			},
		},
		{
			name: "collect and publish, 1 result, cached CRR not found",
			runtimeResults: []RuntimeResult{
				{
					ComplianceRuntimeResult: "test-1",
					RiskScore:               10,
				},
			},
			skipAddingCRR: true,
		},
		{
			name: "collect and publish, 1 result, link CRR to API Service Instance",
			runtimeResults: []RuntimeResult{
				{
					ComplianceRuntimeResult: "test-1",
					RiskScore:               10,
					ApiServiceInstance:      management.NewAPIServiceInstance("apisi1", "env"),
					ComplianceScopedEnv:     "env",
				},
			},
		},
		{
			name: "collect and publish, 1 result, link CRR to Environment",
			runtimeResults: []RuntimeResult{
				{
					ComplianceRuntimeResult: "test-1",
					RiskScore:               10,
					Environment:             management.NewEnvironment("env"),
					ComplianceScopedEnv:     "env",
				},
			},
		},
		{
			name: "collect and publish, 1 result, link CRR to API Service",
			runtimeResults: []RuntimeResult{
				{
					ComplianceRuntimeResult: "test-1",
					RiskScore:               10,
					ApiService:              management.NewAPIService("apis1", "env"),
					ComplianceScopedEnv:     "env",
				},
			},
		},
		{
			name: "collect and publish, 1 result, with existing results, without runtime",
			runtimeResults: []RuntimeResult{
				{
					ComplianceRuntimeResult: "test-1",
					RiskScore:               10,
				},
			},
			existingResults: []interface{}{
				map[string]interface{}{
					"type": "Graylog",
				},
			},
		},
		{
			name: "collect and publish, 1 result, with existing results, with runtime",
			runtimeResults: []RuntimeResult{
				{
					ComplianceRuntimeResult: "test-1",
					RiskScore:               10,
				},
			},
			existingResults: []interface{}{
				map[string]interface{}{
					"type": "Graylog",
					"runtime": map[string]interface{}{
						"riskScore": 5,
					},
				},
			},
		},
		{
			name: "collect and publish, 1 result, with existing agentName and agentType",
			runtimeResults: []RuntimeResult{
				{
					ComplianceRuntimeResult: "test-1",
					RiskScore:               10,
					ComplianceAgentName:     "agent-name",
					ComplianceAgentType:     "agent-type",
				},
			},
			existingResults: []interface{}{
				map[string]interface{}{
					"type": "Graylog",
					"runtime": map[string]interface{}{
						"riskScore": 5,
					},
				},
			},
			expectedAgentName: "agent-name",
			expectedAgentType: "agent-type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resReqCount = 0
			subResReqCount = 0
			agent.InitializeForTest(apicMockCli)
			processor := &mockProcessor{
				results: tt.runtimeResults,
			}
			for _, result := range tt.runtimeResults {
				crr := management.NewComplianceRuntimeResult(result.ComplianceRuntimeResult, "test")
				crr.Spec.Results = tt.existingResults
				ri, _ := crr.AsInstance()
				if !tt.skipAddingCRR {
					agent.GetCacheManager().AddComplianceRuntimeResult(ri)
				}
				if result.ApiServiceInstance != nil {
					apisiRI, _ := result.ApiServiceInstance.AsInstance()
					agent.GetCacheManager().AddAPIServiceInstance(apisiRI)
				} else if result.Environment != nil {
					envRI, _ := result.Environment.AsInstance()
					agent.GetCacheManager().AddWatchResource(envRI)
				} else if result.ApiService != nil {
					apisRI, _ := result.ApiService.AsInstance()
					agent.GetCacheManager().AddWatchResource(apisRI)
				}
			}
			cm := GetManager()
			manager, ok := cm.(*complianceManager)
			assert.True(t, ok)
			manager.job = &runtimeComplianceJob{
				logger:    manager.logger,
				processor: processor,
			}
			manager.job.Execute()
			assert.True(t, processor.collectCalled)
			expectedSubResCount := 0

			for _, result := range tt.runtimeResults {
				cacheManager := agent.GetCacheManager()
				ri, _ := cacheManager.GetComplianceRuntimeResultByName(result.ComplianceRuntimeResult)
				crr := &management.ComplianceRuntimeResult{}
				crr.FromInstance(ri)

				assert.Len(t, crr.Spec.Results, 1)
				assert.Equal(t, result.RiskScore, crr.Spec.Results[0].(map[string]interface{})["runtime"].(map[string]interface{})["riskScore"])
				assert.Equal(t, tt.expectedAgentName, crr.Spec.ComplianceAgent)
				assert.Equal(t, tt.expectedAgentType, crr.Spec.Type)
				specHash, _ := util.GetAgentDetailsValue(crr, definitions.AttrSpecHash)
				assert.NotEmpty(t, specHash)
				if result.ApiServiceInstance != nil {
					expectedSubResCount++
					assert.Equal(t, result.ComplianceRuntimeResult, crr.Name)
				}
				if result.ApiService != nil {
					expectedSubResCount++
					assert.Equal(t, result.ComplianceRuntimeResult, crr.Name)
				}
				if result.Environment != nil {
					expectedSubResCount++
					assert.Equal(t, result.ComplianceRuntimeResult, crr.Name)
				}
			}
			assert.Equal(t, expectedSubResCount, subResReqCount)
			assert.Equal(t, len(tt.runtimeResults), resReqCount)
		})
	}
}
