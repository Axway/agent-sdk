package transaction

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/stretchr/testify/assert"
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
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "id property not set in transaction event", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "direction property not set in transaction event", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		SetTransactionID("11111").
		SetTimestamp(timeStamp).
		SetID("1111").
		SetDirection("Inbound").
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "status property not set in transaction event", err.Error())

	logEvent, err = NewTransactionEventBuilder().
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
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
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
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
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
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
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
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
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
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
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
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
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		SetTimestamp(timeStamp).
		SetStatus(TxSummaryStatusSuccess, "200").
		SetDuration(10).
		SetApplication("1111", "TestApp").
		SetTeam("1111").
		SetProxy("", "proxy", 1).
		SetEntryPoint("http", "GET", "/test", "somehost.com").
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
	assert.Equal(t, "unknown", logEvent.TransactionSummary.Proxy.ID)
	assert.Equal(t, "proxy", logEvent.TransactionSummary.Proxy.Name)
	assert.Equal(t, 1, logEvent.TransactionSummary.Proxy.Revision)

	assert.NotNil(t, logEvent.TransactionSummary.EntryPoint)
	assert.Equal(t, "http", logEvent.TransactionSummary.EntryPoint.Type)
	assert.Equal(t, "GET", logEvent.TransactionSummary.EntryPoint.Method)
	assert.Equal(t, "/{*}", logEvent.TransactionSummary.EntryPoint.Path, "Path was not redacted as it should have been")
	assert.Equal(t, "somehost.com", logEvent.TransactionSummary.EntryPoint.Host)
	assert.Equal(t, true, logEvent.TransactionSummary.IsInMetricEvent)

	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		SetDuration(10).
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "transaction entry point details are not set in transaction summary event", err.Error())

	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		SetEntryPoint("http", "GET", "/test", "somehost.com").
		SetDuration(10).
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "status property not set in transaction summary event", err.Error())

	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		SetEntryPoint("http", "GET", "/test", "somehost.com").
		SetDuration(10).
		SetStatus("Pass", "200").
		Build()
	assert.Nil(t, logEvent)
	assert.NotNil(t, err)
	assert.Equal(t, "invalid transaction summary status", err.Error())

	// Test with explicitly setting properties that are set thru agent config by default
	logEvent, err = NewTransactionSummaryBuilder().
		SetRedactionConfig(redactionConfig).
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		SetEntryPoint("http", "GET", "/test", "somehost.com").
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
	assert.Equal(t, "unknown", logEvent.TransactionSummary.Proxy.ID)
	assert.Equal(t, "", logEvent.TransactionSummary.Proxy.Name)
	assert.Equal(t, 0, logEvent.TransactionSummary.Proxy.Revision)

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
	assert.Equal(t, "somehost.com", logEvent.TransactionSummary.EntryPoint.Host)
	assert.Equal(t, false, logEvent.TransactionSummary.IsInMetricEvent)
}

func TestLogRedactionOverride(t *testing.T) {

	redactionConfig := &redactionTest{}

	timeStamp := time.Now().Unix()

	httpProtocol, _ := createHTTPProtocol("/testuri", "GET", "{}", "{}", 200, 10, 10, redactionConfig)

	logEvent, err := NewTransactionEventBuilder().
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
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
		SetTargetPath("/targetPath").
		SetResourcePath("/resourcePath").
		SetEntryPoint("http", "GET", "/test", "somehost.com").
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
