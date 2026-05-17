package transaction

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
