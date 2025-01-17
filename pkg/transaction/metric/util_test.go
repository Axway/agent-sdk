package metric

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/stretchr/testify/assert"
)

func TestCentralMetricFromAPIMetric(t *testing.T) {
	agent.InitializeForTest(
		nil,
		agent.TestWithCentralConfig(&config.CentralConfiguration{AgentName: "agent"}),
	)

	testCases := map[string]struct {
		skip           bool
		input          *APIMetric
		expectedOutput *centralMetric
	}{
		"expected nil output when nil input": {},
		"expect simple metric with transactions": {
			input: &APIMetric{
				EventID:    "id-1",
				StatusCode: "200",
				Count:      100,
				Response: ResponseMetrics{
					Max: 100,
					Min: 10,
					Avg: 50,
				},
				Observation: models.ObservationDetails{
					Start: 10,
					End:   15,
				},
				Quota: models.Quota{
					ID: "quota",
				},
			},
			expectedOutput: &centralMetric{
				EventID: "id-1",
				Observation: &models.ObservationDetails{
					Start: 10,
				},
				Reporter: &Reporter{
					AgentVersion:     cmd.BuildVersion,
					AgentType:        cmd.BuildAgentName,
					AgentSDKVersion:  cmd.SDKBuildVersion,
					AgentName:        agent.GetCentralConfig().GetAgentName(),
					ObservationDelta: 5,
				},
				Units: &Units{
					Transactions: &Transactions{
						UnitCount: UnitCount{
							Count: 100,
							Quota: &models.ResourceReference{
								ID: "quota",
							},
						},
						Response: &ResponseMetrics{
							Max: 100,
							Min: 10,
							Avg: 50,
						},
						Status: "Success",
					},
				},
			},
		},
		"expect full marketplace context to work with a custom unit": {
			input: &APIMetric{
				EventID: "id-1",
				Count:   100,
				Observation: models.ObservationDetails{
					Start: 10,
					End:   15,
				},
				Subscription: models.Subscription{
					ID:   "sub",
					Name: "sub",
				},
				App: models.AppDetails{
					ID:            "app",
					Name:          "app",
					ConsumerOrgID: "org",
				},
				Product: models.Product{
					ID:          "product",
					Name:        "product",
					VersionName: "version",
					VersionID:   "version",
				},
				API: models.APIDetails{
					ID:       "api",
					Name:     "api",
					Revision: 1,
				},
				AssetResource: models.AssetResource{
					ID:   "asset",
					Name: "asset",
				},
				ProductPlan: models.ProductPlan{
					ID: "productplan",
				},
				Quota: models.Quota{
					ID: "quota",
				},
				Unit: &models.Unit{
					Name: "custom",
				},
			},
			expectedOutput: &centralMetric{
				EventID: "id-1",
				Reporter: &Reporter{
					AgentVersion:     cmd.BuildVersion,
					AgentType:        cmd.BuildAgentName,
					AgentSDKVersion:  cmd.SDKBuildVersion,
					AgentName:        agent.GetCentralConfig().GetAgentName(),
					ObservationDelta: 5,
				},
				Observation: &models.ObservationDetails{
					Start: 10,
				},
				Units: &Units{
					CustomUnits: map[string]*UnitCount{
						"custom": {
							Count: 100,
							Quota: &models.ResourceReference{
								ID: "quota",
							},
						},
					},
				},
				ProductPlan: &models.ResourceReference{
					ID: "productplan",
				},
				AssetResource: &models.ResourceReference{
					ID: "asset",
				},
				API: &models.APIResourceReference{
					ResourceReference: models.ResourceReference{
						ID: "api",
					},
					Name:         "api",
					APIServiceID: "",
				},
				Product: &models.ProductResourceReference{
					ResourceReference: models.ResourceReference{
						ID: "product",
					},
					VersionID: "version",
				},
				App: &models.ApplicationResourceReference{
					ResourceReference: models.ResourceReference{
						ID: "app",
					},
					ConsumerOrgID: "org",
				},
				Subscription: &models.ResourceReference{
					ID: "sub",
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				return
			}
			output := centralMetricFromAPIMetric(tc.input)
			assert.Equal(t, tc.expectedOutput, output)
		})
	}
}
