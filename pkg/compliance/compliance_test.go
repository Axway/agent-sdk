package compliance

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
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
	reqCount := 0

	apicMockCli.CreateOrUpdateResourceMock = func(ri v1.Interface) (*v1.ResourceInstance, error) {
		resInst, _ := ri.AsInstance()
		crr := management.NewComplianceRuntimeResult("", "")
		crr.FromInstance(resInst)
		reqCount++

		agent.GetCacheManager().AddComplianceRuntimeResult(resInst)
		return resInst, nil
	}

	tests := []struct {
		name            string
		runtimeResults  []RuntimeResult
		existingResults []interface{}
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqCount = 0
			agent.InitializeForTest(apicMockCli)
			processor := &mockProcessor{
				results: tt.runtimeResults,
			}
			for _, result := range tt.runtimeResults {
				crr := management.NewComplianceRuntimeResult(result.ComplianceRuntimeResult, "test")
				crr.Spec.Results = tt.existingResults
				ri, _ := crr.AsInstance()
				agent.GetCacheManager().AddComplianceRuntimeResult(ri)
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
			for _, result := range tt.runtimeResults {
				cacheManager := agent.GetCacheManager()
				ri, _ := cacheManager.GetComplianceRuntimeResultByName(result.ComplianceRuntimeResult)
				crr := &management.ComplianceRuntimeResult{}
				crr.FromInstance(ri)

				assert.Len(t, crr.Spec.Results, 1)
				assert.Equal(t, result.RiskScore, crr.Spec.Results[0].(map[string]interface{})["runtime"].(map[string]interface{})["riskScore"])
			}

			assert.Equal(t, len(tt.runtimeResults), reqCount)
		})
	}
}
