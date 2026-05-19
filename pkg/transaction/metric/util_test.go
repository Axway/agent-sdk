package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
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
				Version:     "3",
				Environment: &EnvironmentInfo{RuntimeType: runtimeTypeUnmanaged},
				EventID:     "id-1",
				Observation: &models.ObservationDetails{
					Start: 10,
					End:   15,
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
						Duration: 5,
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
				Version:     "3",
				Environment: &EnvironmentInfo{RuntimeType: runtimeTypeUnmanaged},
				EventID:     "id-1",
				Reporter: &Reporter{
					AgentVersion:     cmd.BuildVersion,
					AgentType:        cmd.BuildAgentName,
					AgentSDKVersion:  cmd.SDKBuildVersion,
					AgentName:        agent.GetCentralConfig().GetAgentName(),
					ObservationDelta: 5,
				},
				Observation: &models.ObservationDetails{
					Start: 10,
					End:   15,
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
					Owner:        &models.OwnerBlock{Type: "unknown"},
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
					Owner:         &models.OwnerBlock{Type: "unknown"},
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

func makeAppRI(name string, owner *v1.Owner) *v1.ResourceInstance {
	app := management.NewManagedApplication(name, "env1")
	app.Marketplace = management.ManagedApplicationMarketplace{
		Name: "mp1",
		Resource: management.ManagedApplicationMarketplaceResource{Owner: owner},
	}
	ri, _ := app.AsInstance()
	return ri
}

func newUtilCacheConfig() *config.CentralConfiguration { return &config.CentralConfiguration{} }

func TestIsKnownID(t *testing.T) {
	cases := map[string]struct {
		id   string
		want bool
	}{
		"empty string is not known":    {id: "", want: false},
		"unknown literal is not known": {id: "unknown", want: false},
		"valid id is known":            {id: "abc-123", want: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, isKnownID(tc.id))
		})
	}
}

func TestCentralConfigFields(t *testing.T) {
	cases := map[string]struct {
		axwayManaged  bool
		agentName     string
		wantRuntime   string
		wantAgentName string
	}{
		"non-managed config returns unmanaged": {
			axwayManaged:  false,
			agentName:     "agent-connected",
			wantRuntime:   runtimeTypeUnmanaged,
			wantAgentName: "agent-connected",
		},
		"axway-managed config returns managed": {
			axwayManaged:  true,
			agentName:     "agent-managed",
			wantRuntime:   runtimeTypeManaged,
			wantAgentName: "agent-managed",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := &config.CentralConfiguration{AgentName: tc.agentName}
			cfg.SetAxwayManaged(tc.axwayManaged)
			agent.InitializeForTest(nil, agent.TestWithCentralConfig(cfg))

			_, agentName, runtimeType := centralConfigFields()
			assert.Equal(t, tc.wantRuntime, runtimeType)
			assert.Equal(t, tc.wantAgentName, agentName)
		})
	}
}

func TestResolveAppOwnerFromCache(t *testing.T) {
	cases := map[string]struct {
		appRI    *v1.ResourceInstance
		appID    string
		wantType string
		wantGUID string
	}{
		"app not found in cache returns unknown": {
			appRI:    nil,
			appID:    "missing-app",
			wantType: "unknown",
		},
		"app found with team owner returns team block": {
			appRI:    makeAppRI("app-team", &v1.Owner{Type: v1.TeamOwner, ID: "team-guid-1"}),
			appID:    "app-team",
			wantType: "team",
			wantGUID: "team-guid-1",
		},
		"app with nil owner returns none": {
			appRI:    makeAppRI("app-no-owner", nil),
			appID:    "app-no-owner",
			wantType: "none",
		},
		"app with empty team GUID returns unknown": {
			appRI:    makeAppRI("app-empty-guid", &v1.Owner{Type: v1.TeamOwner, ID: ""}),
			appID:    "app-empty-guid",
			wantType: "unknown",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			agent.InitializeForTest(nil, agent.TestWithCentralConfig(newUtilCacheConfig()))
			if tc.appRI != nil {
				agent.GetCacheManager().AddManagedApplication(tc.appRI)
			}

			owner := resolveAppOwnerFromCache(tc.appID)
			require.NotNil(t, owner)
			assert.Equal(t, tc.wantType, owner.Type)
			if tc.wantGUID != "" {
				assert.Equal(t, tc.wantGUID, owner.TeamGUID)
			}
		})
	}
}
