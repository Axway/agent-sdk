package transaction

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/stretchr/testify/assert"
)

const (
	testTargetPath   = "/targetPath"
	testResourcePath = "/resourcePath"
	testHost         = "somehost.com"

	prefixedCatFactAPI      = "remoteApiId_cat-fact-api"
	fallbackAPIName         = "fallback-api"
	prefixedFallbackAPIName = SummaryEventAPINamePrefix + fallbackAPIName
)

func TestTransactionEventBuilder(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
		resp.Write([]byte(token))
	}))
	defer s.Close()

	cfg := createMapperTestConfig(s.URL, "1111", "aaa", "env1", "1111")
	agent.Initialize(cfg.Central)
	timeStamp := time.Now().Unix()

	config := redaction.Config{
		Path: redaction.Path{
			Allowed: []redaction.Show{},
		},
		Args: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		RequestHeaders: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		ResponseHeaders: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		MaskingCharacters: "{*}",
		JMSProperties: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
	}

	redactionConfig, _ := config.SetupRedactions()
	httpProtocol, _ := createHTTPProtocol("/testuri", "GET", "{}", "{}", 200, 10, 10, redactionConfig)

	logEvent, err := NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "id property not set in transaction event", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "direction property not set in transaction event", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetDirection("Inbound").
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "status property not set in transaction event", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetDirection("Inbound").
		SetStatus("Success").
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "invalid transaction event status", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetParentID("0000").
		SetSource("source").
		SetDestination("destination").
		SetDuration(10).
		SetDirection("Inbound").
		SetStatus(TxEventStatusPass).
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "protocol details not set in transaction event", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetParentID("0000").
		SetSource("source").
		SetDestination("destination").
		SetDuration(10).
		SetDirection("Inbound").
		SetStatus(TxEventStatusPass).
		SetProtocolDetail("").
		Build()

	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "unsupported protocol type", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetParentID("0000").
		SetSource("source").
		SetDestination("destination").
		SetDuration(10).
		SetDirection("Inbound").
		SetStatus(TxEventStatusPass).
		SetProtocolDetail(httpProtocol).
		SetRedactionConfig(redactionConfig).
		Build()
	assert.NotNil(t, logEvent)
	assert.Nil(t, err)

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetParentID("0000").
		SetSource("source").
		SetDestination("destination").
		SetDuration(10).
		SetDirection("Inbound").
		SetStatus(TxEventStatusPass).
		SetProtocolDetail(httpProtocol).
		Build()

	assert.Nil(t, err)
	assert.Equal(t, "1.0", logEvent.Version)
	assert.Equal(t, "1111", logEvent.TenantID)
	assert.Equal(t, "1111", logEvent.TrcbltPartitionID)
	assert.Equal(t, "env1", logEvent.EnvironmentName)
	assert.Equal(t, "1111", logEvent.EnvironmentID)
	assert.Equal(t, "aaa", logEvent.APICDeployment)
	assert.Equal(t, "", logEvent.Environment)
	assert.Equal(t, timeStamp, logEvent.Stamp)
	assert.Equal(t, TypeTransactionEvent, logEvent.Type)

	assert.Nil(t, logEvent.TransactionSummary)
	assert.NotNil(t, logEvent.TransactionEvent)

	assert.Equal(t, "1111", logEvent.TransactionEvent.ID)
	assert.Equal(t, "0000", logEvent.TransactionEvent.ParentID)
	assert.Equal(t, "source", logEvent.TransactionEvent.Source)
	assert.Equal(t, "destination", logEvent.TransactionEvent.Destination)
	assert.Equal(t, 10, logEvent.TransactionEvent.Duration)
	assert.Equal(t, "Inbound", logEvent.TransactionEvent.Direction)
	assert.Equal(t, string(TxEventStatusPass), logEvent.TransactionEvent.Status)
	assert.NotNil(t, logEvent.TransactionEvent.Protocol)
	_, ok := logEvent.TransactionEvent.Protocol.(*Protocol)
	assert.True(t, ok)

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTenantID("2222").
		SetTrcbltPartitionID("2222").
		SetEnvironmentName("env2").
		SetEnvironmentID("2222").
		SetAPICDeployment("bbb").
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetStatus(TxEventStatusPass).
		SetDirection("Inbound").
		SetProtocolDetail(httpProtocol).
		Build()

	assert.Nil(t, err)
	assert.Equal(t, "1.0", logEvent.Version)
	assert.Equal(t, "2222", logEvent.TenantID)
	assert.Equal(t, "2222", logEvent.TrcbltPartitionID)
	assert.Equal(t, "env2", logEvent.EnvironmentName)
	assert.Equal(t, "2222", logEvent.EnvironmentID)
	assert.Equal(t, "bbb", logEvent.APICDeployment)
	assert.Equal(t, "", logEvent.Environment)
}

func TestSummaryBuilder(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
		resp.Write([]byte(token))
	}))
	defer s.Close()

	cfg := createMapperTestConfig(s.URL, "1111", "aaa", "env1", "1111")
	// authCfg := cfg.Central.GetAuthConfig()
	agent.Initialize(cfg.Central)
	timeStamp := time.Now().Unix()
	config := redaction.Config{
		Path: redaction.Path{
			Allowed: []redaction.Show{},
		},
		Args: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		RequestHeaders: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		ResponseHeaders: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		MaskingCharacters: "{*}",
		JMSProperties: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
	}

	redactionConfig, _ := config.SetupRedactions()

	logEvent, err := NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTransactionID("11111").
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTimestamp(timeStamp).
		SetStatus(TxSummaryStatusSuccess, "200").
		SetDuration(10).
		SetApplication("1111", "TestApp").
		SetTeam("1111").
		SetProxy("", "proxy", 1).
		SetEntryPoint("http", "GET", "/test", testHost).
		SetIsInMetricEvent(true).
		Build()

	assert.Nil(t, err)
	assert.Equal(t, "1.0", logEvent.Version)
	assert.Equal(t, "1111", logEvent.TenantID)
	assert.Equal(t, "1111", logEvent.TrcbltPartitionID)
	assert.Equal(t, "env1", logEvent.EnvironmentName)
	assert.Equal(t, "1111", logEvent.EnvironmentID)
	assert.Equal(t, "aaa", logEvent.APICDeployment)
	assert.Equal(t, "", logEvent.Environment)
	assert.Equal(t, timeStamp, logEvent.Stamp)
	assert.Equal(t, TypeTransactionSummary, logEvent.Type)

	assert.Nil(t, logEvent.TransactionEvent)
	assert.NotNil(t, logEvent.TransactionSummary)

	assert.Equal(t, string(TxSummaryStatusSuccess), logEvent.TransactionSummary.Status)
	assert.Equal(t, "200", logEvent.TransactionSummary.StatusDetail)
	assert.Equal(t, 10, logEvent.TransactionSummary.Duration)

	assert.NotNil(t, logEvent.TransactionSummary.Application)
	assert.Equal(t, "1111", logEvent.TransactionSummary.Application.ID)
	assert.Equal(t, "TestApp", logEvent.TransactionSummary.Application.Name)

	assert.NotNil(t, logEvent.TransactionSummary.Team)
	assert.Equal(t, "1111", logEvent.TransactionSummary.Team.ID)

	assert.NotNil(t, logEvent.TransactionSummary.Proxy)
	assert.Equal(t, "remoteApiName_proxy", logEvent.TransactionSummary.Proxy.ID)
	assert.Equal(t, "proxy", logEvent.TransactionSummary.Proxy.Name)
	assert.Equal(t, 1, logEvent.TransactionSummary.Proxy.Revision)

	assert.NotNil(t, logEvent.TransactionSummary.EntryPoint)
	assert.Equal(t, "http", logEvent.TransactionSummary.EntryPoint.Type)
	assert.Equal(t, "GET", logEvent.TransactionSummary.EntryPoint.Method)
	assert.Equal(t, "/{*}", logEvent.TransactionSummary.EntryPoint.Path, "Path was not redacted as it should have been")
	assert.Equal(t, testHost, logEvent.TransactionSummary.EntryPoint.Host)
	assert.Equal(t, true, logEvent.TransactionSummary.IsInMetricEvent)

	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetDuration(10).
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "transaction entry point details are not set in transaction summary event", err.Error())

	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetEntryPoint("http", "GET", "/test", testHost).
		SetDuration(10).
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "status property not set in transaction summary event", err.Error())

	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetEntryPoint("http", "GET", "/test", testHost).
		SetDuration(10).
		SetStatus("Pass", "200").
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "invalid transaction summary status", err.Error())

	// Test with explicitly setting properties that are set thru agent config by default
	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetEntryPoint("http", "GET", "/test", testHost).
		SetTenantID("2222").
		SetTrcbltPartitionID("2222").
		SetEnvironmentName("env2").
		SetEnvironmentID("2222").
		SetAPICDeployment("bbb").
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetStatus(TxSummaryStatusSuccess, "200").
		SetDuration(10).
		SetProduct("2222", "productname", "1.0").
		SetRunTime("1111", "runtime1").
		SetIsInMetricEvent(false).
		Build()

	assert.Nil(t, err)
	assert.Equal(t, "1.0", logEvent.Version)
	assert.Equal(t, "2222", logEvent.TenantID)
	assert.Equal(t, "2222", logEvent.TrcbltPartitionID)
	assert.Equal(t, "env2", logEvent.EnvironmentName)
	assert.Equal(t, "2222", logEvent.EnvironmentID)
	assert.Equal(t, "bbb", logEvent.APICDeployment)
	assert.Equal(t, timeStamp, logEvent.Stamp)

	assert.Equal(t, string(TxSummaryStatusSuccess), logEvent.TransactionSummary.Status)
	assert.Equal(t, "200", logEvent.TransactionSummary.StatusDetail)
	assert.Equal(t, 10, logEvent.TransactionSummary.Duration)

	assert.Nil(t, logEvent.TransactionSummary.Application)
	assert.Nil(t, logEvent.TransactionSummary.Team)

	assert.NotNil(t, logEvent.TransactionSummary.Proxy)
	assert.Equal(t, "remoteApiId_unknown", logEvent.TransactionSummary.Proxy.ID)
	assert.Equal(t, "", logEvent.TransactionSummary.Proxy.Name)
	assert.Equal(t, 1, logEvent.TransactionSummary.Proxy.Revision)

	assert.NotNil(t, logEvent.TransactionSummary.Product)
	assert.Equal(t, "2222", logEvent.TransactionSummary.Product.ID)
	assert.Equal(t, "1.0", logEvent.TransactionSummary.Product.VersionID)

	assert.NotNil(t, logEvent.TransactionSummary.Runtime)
	assert.Equal(t, "1111", logEvent.TransactionSummary.Runtime.ID)
	assert.Equal(t, "runtime1", logEvent.TransactionSummary.Runtime.Name)

	assert.NotNil(t, logEvent.TransactionSummary.EntryPoint)
	assert.Equal(t, "http", logEvent.TransactionSummary.EntryPoint.Type)
	assert.Equal(t, "GET", logEvent.TransactionSummary.EntryPoint.Method)
	assert.Equal(t, "/{*}", logEvent.TransactionSummary.EntryPoint.Path, "Path was not redacted as it should have been")
	assert.Equal(t, testHost, logEvent.TransactionSummary.EntryPoint.Host)
	assert.Equal(t, false, logEvent.TransactionSummary.IsInMetricEvent)
}

func TestLogRedactionOverride(t *testing.T) {

	redactionConfig := &redactionTest{}

	timeStamp := time.Now().Unix()

	httpProtocol, _ := createHTTPProtocol("/testuri", "GET", "{}", "{}", 200, 10, 10, redactionConfig)

	logEvent, err := NewTransactionEventBuilder().
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetTenantID("2222").
		SetTrcbltPartitionID("2222").
		SetEnvironmentName("env2").
		SetEnvironmentID("2222").
		SetAPICDeployment("bbb").
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetStatus(TxEventStatusPass).
		SetDirection("Inbound").
		SetProtocolDetail(httpProtocol).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, logEvent)
	assert.True(t, redactionConfig.uriRedactionCalled)
	assert.False(t, redactionConfig.pathRedactionCalled)
	assert.True(t, redactionConfig.queryArgsRedactionCalled)
	assert.False(t, redactionConfig.queryArgsRedactionStringCalled)
	assert.False(t, redactionConfig.requestHeadersRedactionCalled)
	assert.False(t, redactionConfig.responseHeadersRedactionCalled)
	assert.False(t, redactionConfig.jmsPropertiesRedactionCalled)

	redactionConfig = &redactionTest{}

	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath(testTargetPath).
		SetResourcePath(testResourcePath).
		SetEntryPoint("http", "GET", "/test", testHost).
		SetTenantID("2222").
		SetTrcbltPartitionID("2222").
		SetEnvironmentName("env2").
		SetEnvironmentID("2222").
		SetAPICDeployment("bbb").
		SetTimestamp(timeStamp).
		SetStatus(TxSummaryStatusSuccess, "200").
		SetDuration(10).
		SetProduct("2222", "productname", "1.0").
		SetRunTime("1111", "runtime1").
		SetIsInMetricEvent(false).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, logEvent)
	assert.True(t, redactionConfig.uriRedactionCalled)
	assert.False(t, redactionConfig.pathRedactionCalled)
	assert.False(t, redactionConfig.queryArgsRedactionCalled)
	assert.False(t, redactionConfig.queryArgsRedactionStringCalled)
	assert.False(t, redactionConfig.requestHeadersRedactionCalled)
	assert.False(t, redactionConfig.responseHeadersRedactionCalled)
	assert.False(t, redactionConfig.jmsPropertiesRedactionCalled)
}

func TestEventBuilderSetProxy(t *testing.T) {
	cases := map[string]struct {
		proxyID        string
		proxyName      string
		expectedSource string
	}{
		"already-prefixed ID is preserved": {
			proxyID:        prefixedCatFactAPI,
			proxyName:      "Cat Fact API",
			expectedSource: prefixedCatFactAPI,
		},
		"empty proxyID falls back to proxyName with name prefix": {
			proxyID:        "",
			proxyName:      fallbackAPIName,
			expectedSource: prefixedFallbackAPIName,
		},
		"both empty produces unknown with prefix": {
			proxyID:        "",
			proxyName:      "",
			expectedSource: "remoteApiId_unknown",
		},
		"only-prefix ID falls back to proxyName with name prefix": {
			proxyID:        "remoteApiId_",
			proxyName:      fallbackAPIName,
			expectedSource: prefixedFallbackAPIName,
		},
		"prefixed ID content equal to proxyName is treated as not a real ID": {
			proxyID:        SummaryEventProxyIDPrefix + fallbackAPIName,
			proxyName:      fallbackAPIName,
			expectedSource: prefixedFallbackAPIName,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewTransactionEventBuilder().(*transactionEventBuilder)
			result := b.SetProxy(tc.proxyID, tc.proxyName)
			assert.Equal(t, b, result)
			assert.Equal(t, tc.expectedSource, b.logEvent.TransactionEvent.Source)
		})
	}
}

func TestEventBuilderSetProxyWithStage(t *testing.T) {
	cases := map[string]struct {
		proxyID        string
		proxyName      string
		proxyStage     string
		expectedSource string
	}{
		"already-prefixed ID preserved regardless of stage": {
			proxyID:        "remoteApiId_ext-api-001",
			proxyName:      "My API",
			proxyStage:     "prod",
			expectedSource: "remoteApiId_ext-api-001",
		},
		"empty stage does not affect source resolution": {
			proxyID:        "remoteApiId_ext-api-002",
			proxyName:      "My API",
			proxyStage:     "",
			expectedSource: "remoteApiId_ext-api-002",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewTransactionEventBuilder().(*transactionEventBuilder)
			result := b.SetProxyWithStage(tc.proxyID, tc.proxyName, tc.proxyStage)
			assert.Equal(t, b, result)
			assert.Equal(t, tc.expectedSource, b.logEvent.TransactionEvent.Source)
		})
	}
}

func TestSummaryBuilderResolveProxyID(t *testing.T) {
	cases := map[string]struct {
		proxyID   string
		proxyName string
		expected  string
	}{
		"proxyID with content after prefix ignores proxyName": {
			proxyID:   "remoteApiId_dwight",
			proxyName: "schrute",
			expected:  "remoteApiId_dwight",
		},
		"proxyID with content after prefix and empty proxyName": {
			proxyID:   "remoteApiId_dwight",
			proxyName: "",
			expected:  "remoteApiId_dwight",
		},
		"proxyID is just prefix falls back to proxyName": {
			proxyID:   "remoteApiId_",
			proxyName: "schrute",
			expected:  "remoteApiName_schrute",
		},
		"both empty produces unknown": {
			proxyID:   "",
			proxyName: "",
			expected:  "remoteApiId_unknown",
		},
		"proxyID is just prefix and proxyName empty produces unknown": {
			proxyID:   "remoteApiId_",
			proxyName: "",
			expected:  "remoteApiId_unknown",
		},
		"empty proxyID uses proxyName with name prefix": {
			proxyID:   "",
			proxyName: "schrute",
			expected:  "remoteApiName_schrute",
		},
		"proxyID without prefix preserved as-is": {
			proxyID:   "dwight",
			proxyName: "schrute",
			expected:  "dwight",
		},
		"proxyID with different prefix preserved as-is": {
			proxyID:   "differentPrefix_dwight",
			proxyName: "schrute",
			expected:  "differentPrefix_dwight",
		},
		"proxyID with multiple underscores preserved": {
			proxyID:   "remoteApiId_dwight_test_api",
			proxyName: "schrute",
			expected:  "remoteApiId_dwight_test_api",
		},
		"proxyName with special characters": {
			proxyID:   "",
			proxyName: "proxy-name.with.dots",
			expected:  "remoteApiName_proxy-name.with.dots",
		},
		"proxyID equals exactly the prefix falls back to proxyName": {
			proxyID:   SummaryEventProxyIDPrefix,
			proxyName: "fallback",
			expected:  SummaryEventAPINamePrefix + "fallback",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := transutil.ResolveIDWithPrefix(tc.proxyID, tc.proxyName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSummaryBuilderSetProxyWithStageVersion(t *testing.T) {
	cases := map[string]struct {
		proxyID      string
		proxyName    string
		proxyStage   string
		proxyVersion string
		revision     int
		expectedID   string
	}{
		"complete proxy information with content after prefix": {
			proxyID:      "remoteApiId_dwight",
			proxyName:    "schrute",
			proxyStage:   "prod",
			proxyVersion: "v1.0",
			revision:     1,
			expectedID:   "remoteApiId_dwight",
		},
		"proxy ID is just prefix falls back to proxyName": {
			proxyID:      "remoteApiId_",
			proxyName:    "schrute",
			proxyStage:   "test",
			proxyVersion: "v2.0",
			revision:     2,
			expectedID:   "remoteApiName_schrute",
		},
		"empty proxy information produces unknown": {
			proxyID:      "",
			proxyName:    "",
			proxyStage:   "",
			proxyVersion: "",
			revision:     0,
			expectedID:   "remoteApiId_unknown",
		},
		"both empty with stage and version produces unknown": {
			proxyID:      "",
			proxyName:    "",
			proxyStage:   "stage",
			proxyVersion: "version",
			revision:     1,
			expectedID:   "remoteApiId_unknown",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewTransactionSummaryBuilder().(*transactionSummaryBuilder)
			result := b.SetProxyWithStageVersion(tc.proxyID, tc.proxyName, tc.proxyStage, tc.proxyVersion, tc.revision)
			assert.Equal(t, b, result)
			assert.NotNil(t, b.logEvent.TransactionSummary.Proxy)
			assert.Equal(t, tc.expectedID, b.logEvent.TransactionSummary.Proxy.ID)
			assert.Equal(t, tc.proxyName, b.logEvent.TransactionSummary.Proxy.Name)
			assert.Equal(t, tc.proxyStage, b.logEvent.TransactionSummary.Proxy.Stage)
			assert.Equal(t, tc.proxyVersion, b.logEvent.TransactionSummary.Proxy.Version)
			assert.Equal(t, tc.revision, b.logEvent.TransactionSummary.Proxy.Revision)
		})
	}
}

func TestSummaryBuilderSetProxy(t *testing.T) {
	cases := map[string]struct {
		proxyID    string
		proxyName  string
		revision   int
		expectedID string
		wantStage  string
		wantVer    string
	}{
		"SetProxy delegates with empty stage and version": {
			proxyID:    "remoteApiId_dwight",
			proxyName:  "proxyName",
			revision:   1,
			expectedID: "remoteApiId_dwight",
			wantStage:  "",
			wantVer:    "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewTransactionSummaryBuilder().(*transactionSummaryBuilder)
			result := b.SetProxy(tc.proxyID, tc.proxyName, tc.revision)
			assert.Equal(t, b, result)
			assert.Equal(t, tc.expectedID, b.logEvent.TransactionSummary.Proxy.ID)
			assert.Equal(t, tc.proxyName, b.logEvent.TransactionSummary.Proxy.Name)
			assert.Equal(t, tc.wantStage, b.logEvent.TransactionSummary.Proxy.Stage)
			assert.Equal(t, tc.wantVer, b.logEvent.TransactionSummary.Proxy.Version)
			assert.Equal(t, tc.revision, b.logEvent.TransactionSummary.Proxy.Revision)
		})
	}
}

func TestSummaryBuilderSetProxyWithStage(t *testing.T) {
	cases := map[string]struct {
		proxyID    string
		proxyName  string
		proxyStage string
		revision   int
		expectedID string
		wantVer    string
	}{
		"SetProxyWithStage delegates with empty version": {
			proxyID:    "remoteApiId_dwight",
			proxyName:  "proxyName",
			proxyStage: "prod",
			revision:   1,
			expectedID: "remoteApiId_dwight",
			wantVer:    "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewTransactionSummaryBuilder().(*transactionSummaryBuilder)
			result := b.SetProxyWithStage(tc.proxyID, tc.proxyName, tc.proxyStage, tc.revision)
			assert.Equal(t, b, result)
			assert.Equal(t, tc.expectedID, b.logEvent.TransactionSummary.Proxy.ID)
			assert.Equal(t, tc.proxyName, b.logEvent.TransactionSummary.Proxy.Name)
			assert.Equal(t, tc.proxyStage, b.logEvent.TransactionSummary.Proxy.Stage)
			assert.Equal(t, tc.wantVer, b.logEvent.TransactionSummary.Proxy.Version)
			assert.Equal(t, tc.revision, b.logEvent.TransactionSummary.Proxy.Revision)
		})
	}
}
