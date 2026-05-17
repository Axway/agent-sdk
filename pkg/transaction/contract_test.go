package transaction

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	contractSDKVersion = "1.0.0-sdk"
	contractAgentName  = "test-agent"
	contractLegTxnID   = "contract-leg-1"
	contractAPICDeploy = "prod-deploy"
	contractOrg        = "contract-org"
	contractEnv        = "contract-env"
)

// TestContractTransactionV2Data validates the JSON shape of both api.transaction.event and
// api.transaction.summary envelopes against the schema contract.
func TestContractTransactionV2Data(t *testing.T) {
	reporter := ReporterInfo{
		AgentVersion:    "1.0.0",
		AgentType:       "TestAgent",
		AgentSDKVersion: contractSDKVersion,
		AgentName:       contractAgentName,
	}

	cases := map[string]struct {
		logEvent LogEvent
		orgID    string
		envID    string
		check    func(t *testing.T, ie *InsightsEvent, raw string)
	}{
		"leg event v2 has correct envelope and data shape": {
			logEvent: LogEvent{
				Type:           TypeTransactionEvent,
				TransactionID:  contractLegTxnID,
				APICDeployment: contractAPICDeploy,
				TransactionEvent: &Event{
					ID:        "0",
					Status:    "Pass",
					Duration:  340,
					Direction: "Inbound",
				},
			},
			orgID: contractOrg,
			envID: contractEnv,
			check: func(t *testing.T, ie *InsightsEvent, raw string) {
				assert.Equal(t, insightsEventVersion, ie.Version)
				assert.Equal(t, "api.transaction.event", ie.Event)
				assert.Equal(t, contractOrg, ie.Org)
				assert.NotEmpty(t, ie.ID)
				require.NotNil(t, ie.Distribution)
				assert.Equal(t, contractEnv, ie.Distribution.Environment)
				require.NotNil(t, ie.Session)
				assert.Equal(t, contractLegTxnID, ie.Session.ID)

				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok, "data must be *TransactionLegV2Data")
				assert.Equal(t, "2", data.Version)
				assert.Equal(t, contractAPICDeploy, data.APICDeployment)
				assert.Equal(t, contractLegTxnID, data.TransactionID)
				assert.Equal(t, 0, data.LegID)

				assert.Contains(t, raw, `"api.transaction.event"`)
				assert.Contains(t, raw, `"version":"4"`)
				assert.Contains(t, raw, `"version":"2"`)
				assert.NotContains(t, raw, `"isInMetricEvent"`)
				assert.NotContains(t, raw, `"team":{`)
				assert.NotContains(t, raw, `"apiServiceInstance"`)
				assert.NotContains(t, raw, `"statusDetail"`)
				assert.NotContains(t, raw, `"entryPoint"`)
				assert.NotContains(t, raw, `"assetResource"`)
				assert.NotContains(t, raw, `"apiServiceRevision"`)
			},
		},
		"summary event v2 has correct envelope and data shape": {
			logEvent: LogEvent{
				Type:           TypeTransactionSummary,
				TransactionID:  "contract-sum-1",
				APICDeployment: contractAPICDeploy,
				TransactionSummary: &Summary{
					Status:       "Success",
					StatusDetail: "200",
					Duration:     340,
					OwnerInfo:    &models.OwnerBlock{Type: "team", TeamGUID: "team-contract"},
					EntryPoint: &EntryPoint{
						Method: "GET",
						Path:   "/pets/123",
						Host:   "api.example.com",
					},
					ConsumerDetails: &models.ConsumerDetails{
						Marketplace: &models.MarketplaceReference{
							GUID:          "mp-guid-contract",
							ConsumerOrgID: "consumer-org-contract",
						},
					},
					AppOwnerInfo: &models.OwnerBlock{Type: "team", TeamGUID: "app-team-contract"},
				},
			},
			orgID: contractOrg,
			envID: contractEnv,
			check: func(t *testing.T, ie *InsightsEvent, raw string) {
				assert.Equal(t, insightsEventVersion, ie.Version)
				assert.Equal(t, "api.transaction.summary", ie.Event)
				assert.Equal(t, contractOrg, ie.Org)
				assert.NotEmpty(t, ie.ID)
				require.NotNil(t, ie.Session)
				assert.Equal(t, "contract-sum-1", ie.Session.ID)

				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok, "data must be *TransactionSummaryV2Data")
				assert.Equal(t, "2", data.Version)
				assert.Equal(t, contractAPICDeploy, data.APICDeployment)
				assert.Equal(t, "Success", data.Status)
				assert.Equal(t, "200", data.StatusDetail)
				assert.Equal(t, 340, data.Duration)

				require.NotNil(t, data.API)
				require.NotNil(t, data.API.Owner)
				assert.Equal(t, "team", data.API.Owner.Type)
				assert.Equal(t, "team-contract", data.API.Owner.TeamGUID)

				require.NotNil(t, data.EntryPoint)
				assert.Equal(t, "GET", data.EntryPoint.Method)
				assert.Equal(t, "/pets/123", data.EntryPoint.Path)
				assert.Equal(t, "api.example.com", data.EntryPoint.Host)

				require.NotNil(t, data.ConsumerDetails)
				assert.Equal(t, "consumer-org-contract", data.ConsumerDetails.ConsumerOrgID)
				require.NotNil(t, data.ConsumerDetails.Marketplace)
				assert.Equal(t, "mp-guid-contract", data.ConsumerDetails.Marketplace.GUID)

				require.NotNil(t, data.ConsumerDetails.Application)
				require.NotNil(t, data.ConsumerDetails.Application.Owner)
				assert.Equal(t, "team", data.ConsumerDetails.Application.Owner.Type)
				assert.Equal(t, "app-team-contract", data.ConsumerDetails.Application.Owner.TeamGUID)

				require.NotNil(t, data.Reporter)
				assert.Equal(t, "1.0.0", data.Reporter.Version)
				assert.Equal(t, "TestAgent", data.Reporter.Type)
				assert.Equal(t, contractSDKVersion, data.Reporter.AgentSDKVersion)
				assert.Equal(t, contractAgentName, data.Reporter.AgentName)

				assert.Contains(t, raw, `"api.transaction.summary"`)
				assert.Contains(t, raw, `"version":"4"`)
				assert.Contains(t, raw, `"version":"2"`)
				assert.NotContains(t, raw, `"isInMetricEvent"`)
				assert.NotContains(t, raw, `"legId"`)
				assert.NotContains(t, raw, `"direction"`)
				assert.NotContains(t, raw, `"uri"`)
				assert.NotContains(t, raw, `"application.id"`)
				assert.NotContains(t, raw, `"api.revision"`)
				assert.NotContains(t, raw, `"api.teamId"`)
				assert.NotContains(t, raw, `"proxy.apiServiceInstance"`)
				assert.NotContains(t, raw, `"proxy.revision"`)
			},
		},
		"deprecated proxy fields present when proxy is set": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: "contract-dep-1",
				TransactionSummary: &Summary{
					Status: "Success",
					Proxy:  &Proxy{ID: "proxy-deprecated-id", Name: "proxy-deprecated-name"},
				},
			},
			orgID: "org-dep",
			envID: "env-dep",
			check: func(t *testing.T, ie *InsightsEvent, raw string) {
				assert.Contains(t, raw, "proxy-deprecated-id")
				assert.Contains(t, raw, "proxy-deprecated-name")
			},
		},
		"deprecated proxy fields absent via omitempty when proxy is nil": {
			logEvent: LogEvent{
				Type:               TypeTransactionSummary,
				TransactionID:      "contract-dep-2",
				TransactionSummary: &Summary{Status: "Success"},
			},
			orgID: "org-dep",
			envID: "env-dep",
			check: func(t *testing.T, ie *InsightsEvent, raw string) {
				assert.NotContains(t, raw, `"proxy.id"`)
				assert.NotContains(t, raw, `"proxy.name"`)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ie, err := BuildTransactionV2Data(log.NewFieldLogger(), tc.logEvent, tc.orgID, tc.envID, nil, reporter)
			require.NoError(t, err)
			require.NotNil(t, ie)

			b, err := json.Marshal(ie)
			require.NoError(t, err)

			tc.check(t, ie, string(b))
		})
	}
}
