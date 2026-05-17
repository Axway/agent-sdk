package metric

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

const (
	contractAgentName = "contract-agent"
	contractAppID     = "app-ctr-3"
)

func contractAPIServiceRI(apiID string, owner *v1.Owner) *v1.ResourceInstance {
	svc := management.NewAPIService("svc-"+apiID, "env-contract")
	svc.SubResources = map[string]interface{}{
		"x-agent-details": map[string]interface{}{
			"externalAPIID": apiID,
		},
	}
	svc.Owner = owner
	ri, _ := svc.AsInstance()
	return ri
}

func contractManagedAppRI(name string, owner *v1.Owner) *v1.ResourceInstance {
	app := management.NewManagedApplication(name, "env-contract")
	app.Marketplace = management.ManagedApplicationMarketplace{
		Name: "mp-contract",
		Resource: management.ManagedApplicationMarketplaceResource{
			Owner: owner,
		},
	}
	ri, _ := app.AsInstance()
	return ri
}

// TestContractMetricV3 validates the metric v3 JSON shape, owner resolution, and removed fields.
func TestContractMetricV3(t *testing.T) {
	cases := map[string]struct {
		setup func()
		input *APIMetric
		check func(t *testing.T, cm *centralMetric, raw string)
	}{
		"version and top-level org fields": {
			setup: func() {
				agent.InitializeForTest(nil, agent.TestWithCentralConfig(&config.CentralConfiguration{AgentName: contractAgentName}))
			},
			input: &APIMetric{
				EventID:    "contract-metric-1",
				StatusCode: "200",
				Count:      10,
				Observation: models.ObservationDetails{
					Start: 1000,
					End:   1500,
				},
				API: models.APIDetails{
					ID:   "contract-api-1",
					Name: "contract-api",
				},
			},
			check: func(t *testing.T, cm *centralMetric, raw string) {
				assert.Equal(t, "3", cm.Version)
				require.NotNil(t, cm.Environment)
				assert.Equal(t, "connected", cm.Environment.RuntimeType)

				event := V4Event{
					Version: "4",
					Event:   "api.transaction.status.metric",
					Org:     "contract-org-guid",
					Data:    cm,
				}
				b, err := json.Marshal(event)
				require.NoError(t, err)
				s := string(b)
				assert.Contains(t, s, `"version":"4"`)
				assert.Contains(t, s, `"api.transaction.status.metric"`)
				assert.Contains(t, s, `"org":"contract-org-guid"`)
				assert.NotContains(t, s, `"app":`)
				assert.Contains(t, s, `"version":"3"`)
			},
		},
		"duration equals observation end minus start": {
			setup: func() {
				agent.InitializeForTest(nil, agent.TestWithCentralConfig(&config.CentralConfiguration{AgentName: contractAgentName}))
			},
			input: &APIMetric{
				EventID:    "contract-metric-dur",
				StatusCode: "200",
				Count:      5,
				Observation: models.ObservationDetails{
					Start: 2000,
					End:   3500,
				},
			},
			check: func(t *testing.T, cm *centralMetric, raw string) {
				require.NotNil(t, cm.Units)
				require.NotNil(t, cm.Units.Transactions)
				assert.Equal(t, int64(1500), cm.Units.Transactions.Duration)
				assert.Contains(t, raw, `"duration":1500`)
			},
		},
		"api owner populated from cache": {
			setup: func() {
				agent.InitializeForTest(nil, agent.TestWithCentralConfig(&config.CentralConfiguration{AgentName: contractAgentName}))
				agent.GetCacheManager().AddAPIService(contractAPIServiceRI("ctr-api-2", &v1.Owner{Type: v1.TeamOwner, ID: "team-ctr-2"}))
			},
			input: &APIMetric{
				EventID:    "contract-metric-owner",
				StatusCode: "200",
				Count:      1,
				Observation: models.ObservationDetails{Start: 100, End: 200},
				API: models.APIDetails{
					ID:   "ctr-api-2",
					Name: "owner-api",
				},
			},
			check: func(t *testing.T, cm *centralMetric, raw string) {
				require.NotNil(t, cm.API)
				require.NotNil(t, cm.API.Owner, "api.owner must be populated when cache has an APIService with an owner")
				assert.Equal(t, "team", cm.API.Owner.Type)
				assert.Equal(t, "team-ctr-2", cm.API.Owner.TeamGUID)
			},
		},
		"api owner falls back to unknown on cache miss": {
			setup: func() {
				agent.InitializeForTest(nil, agent.TestWithCentralConfig(&config.CentralConfiguration{AgentName: contractAgentName}))
			},
			input: &APIMetric{
				EventID:    "contract-metric-miss",
				StatusCode: "200",
				Count:      1,
				Observation: models.ObservationDetails{Start: 100, End: 200},
				API: models.APIDetails{
					ID:   "not-in-cache",
					Name: "miss-api",
				},
			},
			check: func(t *testing.T, cm *centralMetric, raw string) {
				require.NotNil(t, cm.API)
				require.NotNil(t, cm.API.Owner)
				assert.Equal(t, "unknown", cm.API.Owner.Type)
			},
		},
		"application owner populated from cache": {
			setup: func() {
				agent.InitializeForTest(nil, agent.TestWithCentralConfig(&config.CentralConfiguration{AgentName: contractAgentName}))
				agent.GetCacheManager().AddManagedApplication(contractManagedAppRI(contractAppID, &v1.Owner{Type: v1.TeamOwner, ID: "app-team-3"}))
			},
			input: &APIMetric{
				EventID:    "contract-metric-app-owner",
				StatusCode: "200",
				Count:      1,
				Observation: models.ObservationDetails{Start: 100, End: 200},
				App: models.AppDetails{
					ID:   contractAppID,
					Name: contractAppID,
				},
			},
			check: func(t *testing.T, cm *centralMetric, raw string) {
				require.NotNil(t, cm.App)
				require.NotNil(t, cm.App.Owner, "application.owner must be populated when cache has a managed application with an owner")
				assert.Equal(t, "team", cm.App.Owner.Type)
				assert.Equal(t, "app-team-3", cm.App.Owner.TeamGUID)
			},
		},
		"removed fields absent from serialized JSON": {
			setup: func() {
				agent.InitializeForTest(nil, agent.TestWithCentralConfig(&config.CentralConfiguration{AgentName: contractAgentName}))
			},
			input: &APIMetric{
				EventID:    "contract-metric-removed",
				StatusCode: "200",
				Count:      1,
				Observation: models.ObservationDetails{Start: 100, End: 200},
				API: models.APIDetails{
					ID:       "ctr-api-r",
					Name:     "removed-api",
					Revision: 2,
				},
			},
			check: func(t *testing.T, cm *centralMetric, raw string) {
				assert.NotContains(t, raw, `"team":{`)
				assert.NotContains(t, raw, `"apiServiceInstance"`)
				assert.NotContains(t, raw, `"proxy.apiServiceInstance"`)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.setup()

			cm := centralMetricFromAPIMetric(tc.input)
			require.NotNil(t, cm)

			b, err := json.Marshal(cm)
			require.NoError(t, err)

			tc.check(t, cm, string(b))
		})
	}
}
