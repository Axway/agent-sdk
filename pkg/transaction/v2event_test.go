package transaction

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// compile-time assertions. Both data types must satisfy the V4Data interface.
var _ metric.V4Data = (*TransactionLegV2Data)(nil)
var _ metric.V4Data = (*TransactionSummaryV2Data)(nil)

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
	testTxnOutbound1  = "txn-outbound-1"
	testTxnReporter   = "txn-reporter-1"
	testTxnAsset      = "txn-asset-1"
	testTxnRevision   = "txn-revision-1"
	testTxnExclLeg    = "txn-excl-leg"
	testTxnExclSum    = "txn-excl-sum"
	testTxnIfaceLeg   = "txn-iface-leg"
	testTxnProduct1   = "txn-product-1"
	testTxnProduct2   = "txn-product-2"
	testTxnProduct3   = "txn-product-3"
	testTxnConsumer1  = "txn-consumer-1"
	testTxnConsumer2  = "txn-consumer-2"
	testConsumerOrgID = "consumer-org-1"
	testTxnProxyRev    = "txn-proxy-rev"
	testTxnObsDelta    = "txn-obs-delta"
	testRevisionUUID1  = "revision-uuid-abc123"
	testRevisionUUID2  = "revision-uuid-def456"
	testRevisionUUID3  = "revision-uuid-ghi789"
	testProxyName      = "proxy-name"
	testProxyID        = "proxy-id"
	testEntryPointHost = "host.example.com"
	testAPIName        = "my-api"
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
				assert.Equal(t, "leg0", data.ID)
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
				assert.Equal(t, "leg1", data.ID)
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
				assert.Equal(t, "leg0", data.ID)
			},
		},
		"leg with protocol fields are nested under protocol object": {
			logEvent: LogEvent{
				Type:          TypeTransactionEvent,
				TransactionID: "txn-proto-1",
				TransactionEvent: &Event{
					ID:     "0",
					Status: "Pass",
					Protocol: &Protocol{
						URI:    "/v1/resource",
						Method: "POST",
						Status: 201,
					},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				require.NotNil(t, data.Protocol)
				assert.Equal(t, "/v1/resource", data.Protocol.URI)
				assert.Equal(t, "POST", data.Protocol.Method)
				assert.Equal(t, 201, data.Protocol.Status)

				b, err := json.Marshal(data)
				require.NoError(t, err)
				var top map[string]interface{}
				require.NoError(t, json.Unmarshal(b, &top))
				assert.Contains(t, top, "protocol")
				assert.NotContains(t, top, "uri")
				assert.NotContains(t, top, "method")
				assert.NotContains(t, top, "statusCode")
			},
		},
		"leg without protocol has nil protocol field": {
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    "txn-no-proto",
				TransactionEvent: &Event{ID: "0", Status: "Pass"},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				assert.Nil(t, data.Protocol)
			},
		},
		"outbound leg parentId source and destination are populated": {
			logEvent: LogEvent{
				Type:          TypeTransactionEvent,
				TransactionID: testTxnOutbound1,
				TransactionEvent: &Event{
					ID:          "1",
					ParentID:    testTxnOutbound1,
					Source:      "client",
					Destination: "backend-service",
					Status:      "Pass",
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				assert.Equal(t, testTxnOutbound1, data.ParentID)
				assert.Equal(t, "client", data.Source)
				assert.Equal(t, "backend-service", data.Destination)
			},
		},
		"entry leg has no parentId": {
			logEvent: LogEvent{
				Type:          TypeTransactionEvent,
				TransactionID: "txn-entry-parent",
				TransactionEvent: &Event{
					ID:     "0",
					Status: "Pass",
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				assert.Empty(t, data.ParentID)

				b, err := json.Marshal(data)
				require.NoError(t, err)
				assert.NotContains(t, string(b), `"parentId"`)
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
					Proxy:    &Proxy{ID: "proxy-1", Name: testAPIName},
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
				require.NotNil(t, data.Proxy)
				assert.Equal(t, "proxy-id-2", data.Proxy.ID)
				assert.Equal(t, "proxy-name-2", data.Proxy.Name)
			},
		},
		"summary without proxy has no proxy block": {
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
				assert.Nil(t, data.Proxy)
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
						Host:   testEntryPointHost,
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
				assert.Equal(t, testEntryPointHost, data.EntryPoint.Host)
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
					Status:          "Success",
					AppOwnerInfo:    nil,
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
							ConsumerOrgID: testConsumerOrgID,
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
				assert.Equal(t, testConsumerOrgID, data.ConsumerDetails.ConsumerOrgID)
				require.NotNil(t, data.ConsumerDetails.Marketplace)
				assert.Equal(t, "mp-guid-1", data.ConsumerDetails.Marketplace.GUID)
			},
		},
		"summary with product populates product block": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnProduct1,
				TransactionSummary: &Summary{
					Status: "Success",
					Product: &models.Product{
						ID:          "prod-id-1",
						Name:        "my-product",
						VersionID:   "ver-id-1",
						VersionName: "1.0.0",
						Owner:       &models.OwnerBlock{Type: "team", TeamGUID: testTeamGUID},
					},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.Product)
				assert.Equal(t, "prod-id-1", data.Product.ID)
				assert.Equal(t, "my-product", data.Product.Name)
				assert.Equal(t, "ver-id-1", data.Product.VersionID)
				assert.Equal(t, "1.0.0", data.Product.VersionName)
				require.NotNil(t, data.Product.Owner)
				assert.Equal(t, testTeamGUID, data.Product.Owner.TeamGUID)
			},
		},
		"summary with productPlan and quota populates those fields": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnProduct2,
				TransactionSummary: &Summary{
					Status:      "Success",
					ProductPlan: &models.ProductPlan{ID: "plan-id-1"},
					Quota:       &models.Quota{ID: "quota-id-1"},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.ProductPlan)
				assert.Equal(t, "plan-id-1", data.ProductPlan.ID)
				require.NotNil(t, data.Quota)
				assert.Equal(t, "quota-id-1", data.Quota.ID)
			},
		},
		"summary with empty product ID omits product block": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnProduct3,
				TransactionSummary: &Summary{
					Status:  "Success",
					Product: &models.Product{ID: ""},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				assert.Nil(t, data.Product)
			},
		},
		"summary consumerDetails application fields are populated": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnConsumer1,
				TransactionSummary: &Summary{
					Status: "Success",
					ConsumerDetails: &models.ConsumerDetails{
						Application: &models.AppDetails{
							ID:            "app-id-1",
							Name:          "my-app",
							ConsumerOrgID: testConsumerOrgID,
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
				require.NotNil(t, data.ConsumerDetails.Application)
				assert.Equal(t, "app-id-1", data.ConsumerDetails.Application.ID)
				assert.Equal(t, "my-app", data.ConsumerDetails.Application.Name)
				assert.Equal(t, testConsumerOrgID, data.ConsumerDetails.Application.ConsumerOrgID)
			},
		},
		"summary consumerDetails publishedProduct and subscription are populated": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnConsumer2,
				TransactionSummary: &Summary{
					Status: "Success",
					ConsumerDetails: &models.ConsumerDetails{
						PublishedProduct: &models.Product{ID: "pp-id-1", Name: "published-product"},
						Subscription:     &models.Subscription{ID: "sub-id-1", Name: "my-sub"},
					},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.ConsumerDetails)
				require.NotNil(t, data.ConsumerDetails.PublishedProduct)
				assert.Equal(t, "pp-id-1", data.ConsumerDetails.PublishedProduct.ID)
				assert.Equal(t, "published-product", data.ConsumerDetails.PublishedProduct.Name)
				require.NotNil(t, data.ConsumerDetails.Subscription)
				assert.Equal(t, "sub-id-1", data.ConsumerDetails.Subscription.ID)
				assert.Equal(t, "my-sub", data.ConsumerDetails.Subscription.Name)
			},
		},
		// proxy path picks up apiServiceRevision from summary.API when both are set
		"summary with proxy and API populates apiServiceRevision from API": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnProxyRev,
				TransactionSummary: &Summary{
					Status: "Success",
					Proxy:  &Proxy{ID: "proxy-id-rev", Name: testAPIName},
					API: &models.APIDetails{
						ID:                 "api-id-rev",
						APIServiceInstance: testRevisionUUID1,
					},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.APIServiceRevision)
				assert.Equal(t, testRevisionUUID1, data.APIServiceRevision.ID)
			},
		},
		// leg event data must not contain fields reserved for summary
		// ProxyID takes priority over Source for api.id when both are set
		"leg api.id uses ProxyID when set, not Source": {
			logEvent: LogEvent{
				Type:          TypeTransactionEvent,
				TransactionID: "txn-proxyid-priority",
				TransactionEvent: &Event{
					ID:      "0",
					Status:  "Pass",
					Source:  "10.0.0.1",
					ProxyID: "remoteApiId_abc123",
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				require.NotNil(t, data.API)
				assert.Equal(t, "remoteApiId_abc123", data.API.ID)
				assert.Equal(t, "10.0.0.1", data.Source)
			},
		},
		"leg api.id falls back to Source when ProxyID is empty": {
			logEvent: LogEvent{
				Type:          TypeTransactionEvent,
				TransactionID: "txn-source-fallback",
				TransactionEvent: &Event{
					ID:     "0",
					Status: "Pass",
					Source: "remoteApiId_abc123",
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				require.NotNil(t, data.API)
				assert.Equal(t, "remoteApiId_abc123", data.API.ID)
			},
		},
		// leg proxy is populated from SetProxy call
		"leg proxy populated from SetProxy": {
			logEvent: LogEvent{
				Type:          TypeTransactionEvent,
				TransactionID: "txn-leg-proxy",
				TransactionEvent: &Event{
					ID:        "0",
					Status:    "Pass",
					Source:    "remoteApiId_abc123",
					ProxyName: testAPIName,
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				require.NotNil(t, data.Proxy)
				assert.Equal(t, "remoteApiId_abc123", data.Proxy.ID)
				assert.Equal(t, testAPIName, data.Proxy.Name)

				b, err := json.Marshal(data)
				require.NoError(t, err)
				assert.Contains(t, string(b), `"proxy":`)
			},
		},
		// direction is lowercased regardless of what the agent passes
		"leg direction is lowercased from Inbound": {
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    "txn-dir-lower",
				TransactionEvent: &Event{ID: "0", Status: "Pass", Direction: "Inbound"},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				assert.Equal(t, "inbound", data.Direction)
			},
		},
		"leg direction is lowercased from Outbound": {
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    "txn-dir-lower-2",
				TransactionEvent: &Event{ID: "1", Status: "Pass", Direction: "Outbound"},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionLegV2Data)
				require.True(t, ok)
				assert.Equal(t, "outbound", data.Direction)
			},
		},
		// summary proxy falls back to API details when Proxy is nil (controller path)
		"summary proxy falls back to API when Proxy is nil": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: "txn-proxy-api-fallback",
				TransactionSummary: &Summary{
					Status: "Success",
					API: &models.APIDetails{
						ID:   "remoteApiId_abc123",
						Name: "my-api-name",
					},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.Proxy)
				assert.Equal(t, "abc123", data.Proxy.ID)
				assert.Equal(t, "my-api-name", data.Proxy.Name)
			},
		},
		// summary proxy serializes as nested object not flat dot-notation keys
		"summary proxy is nested object": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: "txn-proxy-nested",
				TransactionSummary: &Summary{
					Status: "Success",
					Proxy:  &Proxy{ID: "proxy-nested-id", Name: "proxy-nested-name"},
				},
			},
			orgID:         testOrgID,
			environmentID: testEnvID,
			check: func(t *testing.T, ie *InsightsEvent) {
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.Proxy)
				assert.Equal(t, "proxy-nested-id", data.Proxy.ID)
				assert.Equal(t, "proxy-nested-name", data.Proxy.Name)

				b, err := json.Marshal(data)
				require.NoError(t, err)
				s := string(b)
				assert.Contains(t, s, `"proxy":{"id"`)
				assert.NotContains(t, s, `"proxy.id"`)
				assert.NotContains(t, s, `"proxy.name"`)
			},
		},
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
				assert.NotContains(t, s, `"apiServiceId"`)
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

func TestBuildTransactionV2DataReporter(t *testing.T) {
	reporter := ReporterInfo{
		AgentVersion:    "2.1.0",
		AgentType:       "MyAgent",
		AgentSDKVersion: "1.0.5",
		AgentName:       "my-agent",
	}

	cases := map[string]struct {
		logEvent LogEvent
	}{
		"reporter fields populated on leg event": {
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    testTxnReporter,
				TransactionEvent: &Event{ID: "0", Status: "Pass"},
			},
		},
		"reporter fields populated on summary event": {
			logEvent: LogEvent{
				Type:               TypeTransactionSummary,
				TransactionID:      testTxnReporter + "-sum",
				TransactionSummary: &Summary{Status: "Success"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ie, err := BuildTransactionV2Data(log.NewFieldLogger(), tc.logEvent, testOrgID, testEnvID, nil, reporter)
			require.NoError(t, err)
			require.NotNil(t, ie)

			b, err := json.Marshal(ie)
			require.NoError(t, err)
			s := string(b)
			assert.Contains(t, s, `"2.1.0"`)
			assert.Contains(t, s, `"MyAgent"`)
			assert.Contains(t, s, `"1.0.5"`)
			assert.Contains(t, s, `"my-agent"`)
		})
	}

	t.Run("observationDelta propagates to summary reporter", func(t *testing.T) {
		r := ReporterInfo{ObservationDelta: 60000}
		logEvent := LogEvent{
			Type:               TypeTransactionSummary,
			TransactionID:      testTxnObsDelta,
			TransactionSummary: &Summary{Status: "Success"},
		}
		ie, err := BuildTransactionV2Data(log.NewFieldLogger(), logEvent, testOrgID, testEnvID, nil, r)
		require.NoError(t, err)
		data, ok := ie.Data.(*TransactionSummaryV2Data)
		require.True(t, ok)
		require.NotNil(t, data.Reporter)
		assert.Equal(t, int64(60000), data.Reporter.ObservationDelta)
	})
}

func TestBuildTransactionV2DataSummaryOptionalFields(t *testing.T) {
	reporter := ReporterInfo{AgentVersion: "1.0.0", AgentType: "T", AgentSDKVersion: "0.1", AgentName: "n"}

	cases := map[string]struct {
		logEvent LogEvent
		check    func(t *testing.T, data *TransactionSummaryV2Data)
	}{
		"assetResource populated when present": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnAsset,
				TransactionSummary: &Summary{
					Status:        "Success",
					AssetResource: &models.AssetResource{ID: "asset-abc"},
				},
			},
			check: func(t *testing.T, data *TransactionSummaryV2Data) {
				require.NotNil(t, data.AssetResource)
				assert.Equal(t, "asset-abc", data.AssetResource.ID)
			},
		},
		"assetResource omitted when empty": {
			logEvent: LogEvent{
				Type:               TypeTransactionSummary,
				TransactionID:      testTxnAsset + "-empty",
				TransactionSummary: &Summary{Status: "Success"},
			},
			check: func(t *testing.T, data *TransactionSummaryV2Data) {
				assert.Nil(t, data.AssetResource)
			},
		},
		"apiServiceRevision populated from API.APIServiceInstance": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnRevision,
				TransactionSummary: &Summary{
					Status: "Success",
					API: &models.APIDetails{
						ID:                 "api-id",
						Name:               "api-name",
						APIServiceInstance: "rev-id-123",
					},
				},
			},
			check: func(t *testing.T, data *TransactionSummaryV2Data) {
				require.NotNil(t, data.APIServiceRevision)
				assert.Equal(t, "rev-id-123", data.APIServiceRevision.ID)
			},
		},
		"apiServiceRevision omitted when no revision id": {
			logEvent: LogEvent{
				Type:               TypeTransactionSummary,
				TransactionID:      testTxnRevision + "-empty",
				TransactionSummary: &Summary{Status: "Success"},
			},
			check: func(t *testing.T, data *TransactionSummaryV2Data) {
				assert.Nil(t, data.APIServiceRevision)
				b, err := json.Marshal(data)
				require.NoError(t, err)
				assert.NotContains(t, string(b), `"apiServiceRevision"`)
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ie, err := BuildTransactionV2Data(log.NewFieldLogger(), tc.logEvent, testOrgID, testEnvID, nil, reporter)
			require.NoError(t, err)
			require.NotNil(t, ie)
			data, ok := ie.Data.(*TransactionSummaryV2Data)
			require.True(t, ok)
			tc.check(t, data)
		})
	}
}

func testDefaultRedaction(t *testing.T) redaction.Redactions {
	t.Helper()
	cfg := redaction.DefaultConfig()
	r, err := cfg.SetupRedactions()
	require.NoError(t, err)
	return r
}

func TestSetAPIServiceRevision(t *testing.T) {
	rc := testDefaultRedaction(t)

	cases := map[string]struct {
		revisionID      string
		withProxy       bool
		wantAPIInstance string
		wantAPIName     string
		wantV2Revision  string
	}{
		"sets APIServiceInstance without proxy": {
			revisionID:      testRevisionUUID1,
			withProxy:       false,
			wantAPIInstance: testRevisionUUID1,
		},
		"sets APIServiceInstance after SetProxy preserves proxy name": {
			revisionID:      testRevisionUUID2,
			withProxy:       true,
			wantAPIInstance: testRevisionUUID2,
			wantAPIName:     testProxyName,
		},
		"propagates through to v2 event apiServiceRevision.id": {
			revisionID:     testRevisionUUID3,
			withProxy:      true,
			wantV2Revision: testRevisionUUID3,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewTransactionSummaryBuilder().
				SetTransactionID("txn-rev").
				SetStatus(TxSummaryStatusSuccess, "200").
				SetDuration(100).
				SetEntryPoint("http", "GET", "/path", testEntryPointHost).
				SetRedactionConfig(rc)
			if tc.withProxy {
				b = b.SetProxy(testProxyID, testProxyName, 1)
			}
			le, err := b.SetAPIServiceRevision(tc.revisionID).Build()
			require.NoError(t, err)

			require.NotNil(t, le.TransactionSummary.API)
			if tc.wantAPIInstance != "" {
				assert.Equal(t, tc.wantAPIInstance, le.TransactionSummary.API.APIServiceInstance)
			}
			if tc.wantAPIName != "" {
				assert.Equal(t, tc.wantAPIName, le.TransactionSummary.API.Name)
			}
			if tc.wantV2Revision != "" {
				ie, err := BuildTransactionV2Data(log.NewFieldLogger(), *le, testOrgID, testEnvID, nil, ReporterInfo{})
				require.NoError(t, err)
				data, ok := ie.Data.(*TransactionSummaryV2Data)
				require.True(t, ok)
				require.NotNil(t, data.APIServiceRevision)
				assert.Equal(t, tc.wantV2Revision, data.APIServiceRevision.ID)
			}
		})
	}
}

// TestBuildTransactionV2DataLegExcludedFields verifies that leg events do not contain
// fields excluded from the leg v2 format.
func TestBuildTransactionV2DataLegExcludedFields(t *testing.T) {
	reporter := ReporterInfo{AgentVersion: "1.0.0", AgentType: "T", AgentSDKVersion: "0.1", AgentName: "n"}

	logEvent := LogEvent{
		Type:          TypeTransactionEvent,
		TransactionID: testTxnExclLeg,
		TransactionEvent: &Event{
			ID:     "0",
			Status: "Pass",
			Protocol: &Protocol{
				URI:    "/api/v1",
				Method: "GET",
				Status: 200,
			},
		},
	}

	ie, err := BuildTransactionV2Data(log.NewFieldLogger(), logEvent, testOrgID, testEnvID, nil, reporter)
	require.NoError(t, err)

	// Unmarshal data as a map so we can inspect only top-level keys of the data object.
	b, err := json.Marshal(ie.Data)
	require.NoError(t, err)
	var dataKeys map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &dataKeys))

	absentTopLevel := []string{"app", "team", "teamId", "apiServiceInstance", "isInMetricEvent", "revision", "uri", "method", "statusCode"}
	for _, key := range absentTopLevel {
		assert.NotContains(t, dataKeys, key, "leg data must not have top-level key %q", key)
	}

	// URI/method/status must be nested inside the protocol object.
	protoRaw, ok := dataKeys["protocol"]
	require.True(t, ok, "leg data must have a protocol field")
	proto, ok := protoRaw.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "/api/v1", proto["uri"])
	assert.Equal(t, "GET", proto["method"])
	assert.Equal(t, float64(200), proto["status"])
	assert.NotContains(t, proto, "statusCode", "protocol must use 'status' not 'statusCode'")
}

// TestBuildTransactionV2DataSummaryExcludedFields verifies that summary events do not
// contain fields excluded from the summary v2 format.
func TestBuildTransactionV2DataSummaryExcludedFields(t *testing.T) {
	reporter := ReporterInfo{AgentVersion: "1.0.0", AgentType: "T", AgentSDKVersion: "0.1", AgentName: "n"}

	cases := map[string]struct {
		logEvent     LogEvent
		absentFields []string
	}{
		"summary event excluded fields": {
			logEvent: LogEvent{
				Type:          TypeTransactionSummary,
				TransactionID: testTxnExclSum,
				TransactionSummary: &Summary{
					Status: "Success",
					Proxy:  &Proxy{ID: "p-1", Name: testAPIName},
				},
			},
			absentFields: []string{
				`"app"`,
				`"isInMetricEvent"`,
				`"team"`,
				`"teamId"`,
				`"apiServiceInstance"`,
				`"proxy.apiServiceInstance"`,
				`"proxy.revision"`,
				`"application.id"`,
				`"legId"`,
				`"direction"`,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ie, err := BuildTransactionV2Data(log.NewFieldLogger(), tc.logEvent, testOrgID, testEnvID, nil, reporter)
			require.NoError(t, err)
			b, err := json.Marshal(ie)
			require.NoError(t, err)
			s := string(b)
			for _, field := range tc.absentFields {
				assert.NotContains(t, s, field, "summary event must not contain field %s", field)
			}
			// proxy nested object must be present when proxy is set
			assert.Contains(t, s, `"proxy":`)
			assert.NotContains(t, s, `"proxy.id"`)
			assert.NotContains(t, s, `"proxy.name"`)
		})
	}
}

func TestParseLegID(t *testing.T) {
	tests := map[string]struct {
		input string
		want  int
	}{
		"plain integer zero":       {input: "0", want: 0},
		"plain integer one":        {input: "1", want: 1},
		"plain integer two":        {input: "2", want: 2},
		"prefixed leg0":            {input: "leg0", want: 0},
		"prefixed leg1":            {input: "leg1", want: 1},
		"prefixed leg2":            {input: "leg2", want: 2},
		"empty string":             {input: "", want: 0},
		"non-numeric non-prefixed": {input: testTxnOutbound1, want: 0},
		"negative integer":         {input: "-1", want: 0},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseLegID(tc.input))
		})
	}
}

func TestFormatLegID(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"empty string":             {input: "", want: ""},
		"plain integer zero":       {input: "0", want: "leg0"},
		"plain integer one":        {input: "1", want: "leg1"},
		"already prefixed leg0":    {input: "leg0", want: "leg0"},
		"already prefixed leg1":    {input: "leg1", want: "leg1"},
		"non-numeric arbitrary ID": {input: testTxnOutbound1, want: testTxnOutbound1},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatLegID(tc.input))
		})
	}
}

func TestBuildTransactionV2DataLegIDs(t *testing.T) {
	tests := map[string]struct {
		id       string
		parentID string
		wantID   string
		wantPID  string
	}{
		"plain integer IDs (new agent style)": {
			id: "0", parentID: "", wantID: "leg0", wantPID: "",
		},
		"plain integer IDs outbound (new agent style)": {
			id: "1", parentID: "0", wantID: "leg1", wantPID: "leg0",
		},
		"prefixed IDs (legacy agent style)": {
			id: "leg0", parentID: "", wantID: "leg0", wantPID: "",
		},
		"prefixed IDs outbound (legacy agent style)": {
			id: "leg1", parentID: "leg0", wantID: "leg1", wantPID: "leg0",
		},
		"arbitrary parentID is passed through unchanged": {
			id: "1", parentID: testTxnOutbound1, wantID: "leg1", wantPID: testTxnOutbound1,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			logEvent := LogEvent{
				Type:          TypeTransactionEvent,
				TransactionID: "txn-leg-id-test",
				TransactionEvent: &Event{
					ID:       tc.id,
					ParentID: tc.parentID,
					Status:   "Pass",
				},
			}
			data, err := buildLegV2Data(logEvent, nil, ReporterInfo{})
			require.NoError(t, err)
			assert.Equal(t, tc.wantID, data.ID)
			assert.Equal(t, tc.wantPID, data.ParentID)
		})
	}
}

func TestV4DataInterfaceMethods(t *testing.T) {
	t.Run("TransactionLegV2Data interface methods", func(t *testing.T) {
		d := &TransactionLegV2Data{TransactionID: testTxnIfaceLeg, LegID: 0}
		assert.Equal(t, TypeTransactionEvent, d.GetType())
		assert.Equal(t, testTxnIfaceLeg, d.GetEventID())
		assert.Equal(t, (time.Time{}), d.GetStartTime())
		fields := d.GetLogFields()
		assert.Equal(t, testTxnIfaceLeg, fields["transactionId"])
	})

	t.Run("TransactionSummaryV2Data interface methods", func(t *testing.T) {
		d := &TransactionSummaryV2Data{Status: "Success"}
		assert.Equal(t, TypeTransactionSummary, d.GetType())
		assert.Empty(t, d.GetEventID())
		assert.Equal(t, (time.Time{}), d.GetStartTime())
		fields := d.GetLogFields()
		assert.Equal(t, "Success", fields["status"])
	})
}
