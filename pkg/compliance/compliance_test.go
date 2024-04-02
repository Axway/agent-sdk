package compliance

import (
	"encoding/json"
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
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
	patchReqCount := 0

	apicMockCli.PatchSubResourceMock = func(ri v1.Interface, subResourceName string, patches []map[string]interface{}) (*v1.ResourceInstance, error) {
		patchReqCount++
		resInst, _ := ri.AsInstance()
		instance := management.NewAPIServiceInstance("", "")
		instance.FromInstance(resInst)
		for _, patch := range patches {
			if patch[apic.PatchOperation] != apic.PatchOpBuildObjectTree {
				c := patch[apic.PatchValue]
				buf, _ := json.Marshal(c)
				compliance := &management.ApiServiceInstanceSourceCompliance{}
				json.Unmarshal(buf, compliance)
				instance.Source.Compliance = compliance
				resInst, _ = instance.AsInstance()
				agent.GetCacheManager().AddAPIServiceInstance(resInst)
			}
		}
		return instance.AsInstance()
	}

	tests := []struct {
		name           string
		runtimeResults []RuntimeResult
	}{
		{
			name: "no collection",
		},
		{
			name: "collect and publish",
			runtimeResults: []RuntimeResult{
				{
					APIServiceInstance: "test-1",
					HighCount:          10,
					MediumCount:        10,
					LowCount:           0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchReqCount = 0
			agent.InitializeForTest(apicMockCli)
			processor := &mockProcessor{
				results: tt.runtimeResults,
			}
			for _, result := range tt.runtimeResults {
				inst := management.NewAPIServiceInstance(result.APIServiceInstance, "test")
				inst.Source = &management.ApiServiceInstanceSource{
					DataplaneType: &management.ApiServiceInstanceSourceDataplaneType{
						Managed: string(apic.Unclassified),
					},
				}
				ri, _ := inst.AsInstance()
				agent.GetCacheManager().AddAPIServiceInstance(ri)
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
				ri, _ := cacheManager.GetAPIServiceInstanceByName(result.APIServiceInstance)
				instance := &management.APIServiceInstance{}
				instance.FromInstance(ri)
				assert.NotNil(t, instance.Source)
				assert.NotNil(t, instance.Source.Compliance)
				assert.Equal(t, int32(result.HighCount), instance.Source.Compliance.Runtime.Result.HighCount)
				assert.Equal(t, int32(result.MediumCount), instance.Source.Compliance.Runtime.Result.MediumCount)
				assert.Equal(t, int32(result.LowCount), instance.Source.Compliance.Runtime.Result.LowCount)
			}
			assert.Equal(t, len(tt.runtimeResults), patchReqCount)
		})
	}
}
