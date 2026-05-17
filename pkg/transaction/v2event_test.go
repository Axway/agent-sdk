package transaction

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	testOrgID         = "org-1"
	testEnvID         = "env-1"
	testOrgABC        = "org-abc"
	testEnvABC        = "env-abc"
	testOrgXYZ        = "org-xyz"
	testEnvXYZ        = "env-xyz"
	testTxnLeg1       = "txn-leg-1"
	testTxnLeg2       = "txn-leg-2"
	testTxnLeg3       = "txn-leg-3"
	testTxnSum1       = "txn-sum-1"
	testTxnSum2       = "txn-sum-2"
	testTxnSum3       = "txn-sum-3"
	testTxnSum4       = "txn-sum-4"
	testTxnSumJSON    = "txn-sum-json"
	testTxnErrLeg     = "txn-err-1"
	testTxnErrSummary = "txn-err-2"
	testTxnUnknown    = "txn-1"
	testEventLegName  = "api.transaction.event"
	testEventSumName  = "api.transaction.summary"
	testTxnOwner1     = "txn-owner-1"
	testTxnOwner2     = "txn-owner-2"
	testTxnOwner3     = "txn-owner-3"
	testTxnOwner4     = "txn-owner-4"
	testTxnOwner5     = "txn-owner-5"
	testTxnNoFields1  = "txn-nofields-leg"
	testTxnNoFields2  = "txn-nofields-sum"
	testTeamGUID      = "team-guid-123"
)

func TestBuildTransactionV2Data(t *testing.T) {
	reporter := ReporterInfo{
		AgentVersion:    "1.0.0",
		AgentType:       "TestAgent",
		AgentSDKVersion: "0.0.1",
		AgentName:       "test-agent",
	}

	tests := map[string]struct {
		logEvent      LogEvent
		orgID         string
		environmentID string
		wantErr       bool
		check         func(t *testing.T, ie *InsightsEvent)
	}{
		"unknown event type returns error": {
			logEvent:      LogEvent{Type: "unknownType", TransactionID: testTxnUnknown},
			orgID:         testOrgID,
			environmentID: testEnvID,
			wantErr:       true,
		},
		"nil TransactionEvent for leg type returns error": {
			logEvent:      LogEvent{Type: TypeTransactionEvent, TransactionID: testTxnErrLeg},
			orgID:         testOrgID,
			environmentID: testEnvID,
			wantErr:       true,
		},
		"nil TransactionSummary for summary type returns error": {
			logEvent:      LogEvent{Type: TypeTransactionSummary, TransactionID: testTxnErrSummary},
			orgID:         testOrgID,
			environmentID: testEnvID,
			wantErr:       true,
		},
		"transaction leg event has correct envelope": {
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    testTxnLeg1,
				Stamp:            time.Now().UnixMilli(),
				TransactionEvent: &Event{ID: "0", Status: "Pass"},
			},
			orgID:         testOrgABC,
			environmentID: testEnvABC,
			check: func(t *testing.T, ie *InsightsEvent) {
				assert.Equal(t, insightsEventVersion, ie.Version)
				assert.Equal(t, testEventLegName, ie.Event)
				assert.Equal(t, testOrgABC, ie.Org)
				assert.NotEmpty(t, ie.ID)
				require.NotNil(t, ie.Distribution)
				assert.Equal(t, testEnvABC, ie.Distribution.Environment)
				require.NotNil(t, ie.Session)
				assert.Equal(t, testTxnLeg1, ie.Session.ID)

				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				assert.Equal(t, legDataVersion, data.Version)
				assert.Equal(t, testTxnLeg1, data.TransactionID)
			},
		},
		"transaction leg apic deployment and non-zero leg id": {
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    testTxnLeg2,
				APICDeployment:   "prod",
				TransactionEvent: &Event{ID: "1", Status: "Pass"},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				assert.Equal(t, legDataVersion, data.Version)
				assert.Equal(t, "prod", data.APICDeployment)
				assert.Equal(t, 1, data.LegID)
			},
		},
		"entry leg id is zero": {
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    testTxnLeg3,
				TransactionEvent: &Event{ID: "0", Status: "Pass"},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				assert.Equal(t, 0, data.LegID)
			},
		},
		"transaction summary event has correct envelope": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnSum1,
				Stamp:         time.Now().UnixMilli(),
				TransactionSummary: &Summary{
					Status:   "Success",
					Duration: 150,
					Proxy:    &Proxy{ID: "proxy-1", Name: "my-api"},
				},
			},
			orgID:         testOrgXYZ,
			environmentID: testEnvXYZ,
			check: func(t *testing.T, ie *InsightsEvent) {
				assert.Equal(t, insightsEventVersion, ie.Version)
				assert.Equal(t, testEventSumName, ie.Event)
				assert.Equal(t, testOrgXYZ, ie.Org)
				assert.NotEmpty(t, ie.ID)
				require.NotNil(t, ie.Session)
				assert.Equal(t, testTxnSum1, ie.Session.ID)

				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				assert.Equal(t, summaryDataVersion, data.Version)
				assert.Equal(t, "Success", data.Status)
				assert.Equal(t, 150, data.Duration)
			},
		},
		"summary with proxy populates deprecated fields": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnSum2,
				TransactionSummary: &Summary{
					Status: "Success",
					Proxy:  &Proxy{ID: "proxy-id-2", Name: "proxy-name-2"},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				assert.Equal(t, "proxy-id-2", data.ProxyID)
				assert.Equal(t, "proxy-name-2", data.ProxyName)
			},
		},
		"summary without proxy has no deprecated fields": {
			logEvent: LogEvent{
				Type:               TypeTransactionSummary,
				TransactionID:      testTxnSum3,
				TransactionSummary: &Summary{Status: "Success"},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				assert.Empty(t, data.ProxyID)
				assert.Empty(t, data.ProxyName)
			},
		},
		"summary with entry point is populated": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnSum4,
				TransactionSummary: &Summary{
					Status: "Success",
					EntryPoint: &EntryPoint{
						Method: "GET",
						Path:   "/path",
						Host:   "host.example.com",
					},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.EntryPoint)
				assert.Equal(t, "GET", data.EntryPoint.Method)
				assert.Equal(t, "/path", data.EntryPoint.Path)
				assert.Equal(t, "host.example.com", data.EntryPoint.Host)
			},
		},
		"summary event is JSON serializable with correct version fields": {
			logEvent: LogEvent{
				Type:               TypeTransactionSummary,
				TransactionID:      testTxnSumJSON,
				TransactionSummary: &Summary{Status: "Success", Duration: 100},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				b, err := json.Marshal(ie)
				assert.Nil(t, err)
				assert.Contains(t, string(b), `"version":"4"`)
				assert.Contains(t, string(b), `"api.transaction.summary"`)
			},
		},
		// summary OwnerInfo pre-populated by agents-controller takes precedence over cache
		"summary with pre-populated OwnerInfo uses it directly": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnOwner1,
				TransactionSummary: &Summary{
					Status: "Success",
					OwnerInfo: &models.OwnerBlock{
						Type:     "team",
						TeamGUID: testTeamGUID,
					},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.API)
				require.NotNil(t, data.API.Owner)
				assert.Equal(t, "team", data.API.Owner.Type)
				assert.Equal(t, testTeamGUID, data.API.Owner.TeamGUID)
			},
		},
		// summary AppOwnerInfo propagates to consumerDetails.application.owner
		"summary AppOwnerInfo propagates to consumerDetails application owner": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnOwner2,
				TransactionSummary: &Summary{
					Status: "Success",
					AppOwnerInfo: &models.OwnerBlock{
						Type:     "team",
						TeamGUID: testTeamGUID,
					},
					ConsumerDetails: &models.ConsumerDetails{},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.ConsumerDetails)
				require.NotNil(t, data.ConsumerDetails.Application)
				require.NotNil(t, data.ConsumerDetails.Application.Owner)
				assert.Equal(t, "team", data.ConsumerDetails.Application.Owner.Type)
				assert.Equal(t, testTeamGUID, data.ConsumerDetails.Application.Owner.TeamGUID)
			},
		},
		// nil OwnerInfo falls through to "unknown" when cacheManager is nil
		"summary nil OwnerInfo with nil cache produces unknown owner": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnOwner3,
				TransactionSummary: &Summary{
					Status:    "Success",
					OwnerInfo: nil,
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.API)
				require.NotNil(t, data.API.Owner)
				assert.Equal(t, "unknown", data.API.Owner.Type)
			},
		},
		// nil AppOwnerInfo produces no owner on consumerDetails application
		"summary nil AppOwnerInfo produces no application owner": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnOwner4,
				TransactionSummary: &Summary{
					Status:       "Success",
					AppOwnerInfo: nil,
					ConsumerDetails: &models.ConsumerDetails{},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.ConsumerDetails)
				require.NotNil(t, data.ConsumerDetails.Application)
				assert.Nil(t, data.ConsumerDetails.Application.Owner)
			},
		},
		// marketplace GUID and consumerOrgId propagate through consumerDetails
		"summary marketplace details propagate to consumerDetails": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnOwner5,
				TransactionSummary: &Summary{
					Status: "Success",
					ConsumerDetails: &models.ConsumerDetails{
						Marketplace: &models.MarketplaceReference{
							GUID:          "mp-guid-1",
							ConsumerOrgID: "consumer-org-1",
						},
					},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.ConsumerDetails)
				assert.Equal(t, "consumer-org-1", data.ConsumerDetails.ConsumerOrgID)
				require.NotNil(t, data.ConsumerDetails.Marketplace)
				assert.Equal(t, "mp-guid-1", data.ConsumerDetails.Marketplace.GUID)
			},
		},
		// leg event data must not contain fields reserved for summary
		"leg event JSON must not contain summary-only fields": {
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    testTxnNoFields1,
				TransactionEvent: &Event{ID: "0", Status: "Pass"},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				b, err := json.Marshal(ie)
				require.NoError(t, err)
				s := string(b)
				assert.NotContains(t, s, `"isInMetricEvent"`)
				assert.NotContains(t, s, `"team"`)
				assert.NotContains(t, s, `"apiServiceInstance"`)
				assert.NotContains(t, s, `"entryPoint"`)
				assert.NotContains(t, s, `"statusDetail"`)
				assert.Contains(t, s, `"api.transaction.event"`)
				assert.Contains(t, s, `"version":"4"`)
			},
		},
		// summary event data must not contain fields reserved for leg or metric
		"summary event JSON must not contain leg-only or metric-only fields": {
			logEvent: LogEvent{
				Type:               TypeTransactionSummary,
				TransactionID:      testTxnNoFields2,
				TransactionSummary: &Summary{Status: "Success"},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				b, err := json.Marshal(ie)
				require.NoError(t, err)
				s := string(b)
				assert.NotContains(t, s, `"isInMetricEvent"`)
				assert.NotContains(t, s, `"legId"`)
				assert.NotContains(t, s, `"direction"`)
				assert.NotContains(t, s, `"uri"`)
				assert.Contains(t, s, `"api.transaction.summary"`)
				assert.Contains(t, s, `"version":"4"`)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ie, err := BuildTransactionV2Data(log.NewFieldLogger(), tc.logEvent, tc.orgID, tc.environmentID, nil, reporter)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, ie)
			if tc.check != nil {
				tc.check(t, ie)
			}
		})
	}
}
