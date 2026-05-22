package transaction

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/assert"
)

type Config struct {
	Central corecfg.CentralConfig `config:"central"`
}

func createMapperTestConfig(authURL, tenantID, apicDeployment, envName, envID string) *Config {
	cfg := &Config{
		Central: &corecfg.CentralConfiguration{
			AgentType:                 corecfg.TraceabilityAgent,
			URL:                       "https://xxx.axway.com",
			PlatformURL:               "https://platform.xxx.com",
			TenantID:                  tenantID,
			APICDeployment:            apicDeployment,
			Environment:               envName,
			APIServerVersion:          "v1alpha1",
			UsageReporting:            corecfg.NewUsageReporting("https://platform.xxx.com"),
			MetricReporting:           corecfg.NewMetricReporting(),
			ReportActivityFrequency:   2 * time.Minute,
			APIValidationCronSchedule: "@daily",
			ClientTimeout:             1 * time.Minute,
			Auth: &corecfg.AuthConfiguration{
				URL:        authURL,
				ClientID:   "test",
				Realm:      "Broker",
				PrivateKey: "testdata/private_key.pem",
				PublicKey:  "testdata/public_key",
				Timeout:    10 * time.Second,
			},
		},
	}
	cfg.Central.SetEnvironmentID(envID)
	return cfg
}

func createOfflineMapperTestConfig(envID string) *Config {
	cfg := &Config{
		Central: &corecfg.CentralConfiguration{
			EnvironmentID: envID,
			UsageReporting: &corecfg.UsageReportingConfiguration{
				Offline: true,
			},
		},
	}
	cfg.Central.SetEnvironmentID(envID)
	sampling.SetupSampling(sampling.DefaultConfig(), true, "")
	return cfg
}

func TestCreateEventWithValidTokenRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
		resp.Write([]byte(token))
	}))
	defer s.Close()

	cfg := createMapperTestConfig(s.URL, "1111", "aaa", "env1", "1111")
	err := agent.Initialize(cfg.Central)
	assert.Nil(t, err)

	eventGenerator := NewEventGenerator()
	dummyLogEvent := LogEvent{
		TenantID:      cfg.Central.GetTenantID(),
		Environment:   cfg.Central.GetAPICDeployment(),
		EnvironmentID: cfg.Central.GetEnvironmentID(),
		Type:          TypeTransactionEvent,
		TransactionID: "txn-test-1",
		TransactionEvent: &Event{
			ID:     "0",
			Status: "Pass",
		},
	}
	eventFields := make(common.MapStr)
	eventFields["someKey.1"] = "someVal.1"
	eventFields["someKey.2"] = "someVal.2"
	eventFields["message"] = "existingMessage"

	events, _ := eventGenerator.CreateEvents(LogEvent{}, []LogEvent{dummyLogEvent}, time.Now(), nil, eventFields, nil)
	assert.NotNil(t, events)
	event := events[0]

	// Forwarded fields must be present
	assert.Equal(t, "someVal.1", event.Fields["someKey.1"])
	assert.Equal(t, "someVal.2", event.Fields["someKey.2"])

	msg := fmt.Sprintf("%v", event.Fields["message"])
	fields := event.Fields["fields"].(map[string]string)
	assert.NotNil(t, fields)

	// Original "message" field must not leak through
	assert.NotEqual(t, "existingMessage", event.Fields["message"])

	// Message must now be an InsightsEvent envelope (version "4").
	// Use map to avoid interface{} unmarshaling issues with the Data field.
	var envelope map[string]interface{}
	err = json.Unmarshal([]byte(msg), &envelope)
	assert.Nil(t, err)
	assert.Equal(t, insightsEventVersion, envelope["version"])
	assert.Equal(t, "api.transaction.event", envelope["event"])
	assert.Equal(t, cfg.Central.GetTenantID(), envelope["org"])
	assert.NotEmpty(t, envelope["id"])

	assert.Equal(t, "somevalue", fields["token"])
	assert.Equal(t, traceability.TransactionFlow, fields[traceability.FlowHeader])
}

func TestCreateEventWithInvalidTokenRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusForbidden)
	}))
	defer s.Close()

	cfg := createMapperTestConfig(s.URL, "1111", "aaa", "env1", "1111")
	agent.Initialize(cfg.Central)
	eventGenerator := NewEventGenerator()
	dummyLogEvent := LogEvent{
		TenantID:      cfg.Central.GetTenantID(),
		Environment:   cfg.Central.GetAPICDeployment(),
		EnvironmentID: cfg.Central.GetEnvironmentID(),
		Type:          TypeTransactionEvent,
		TransactionID: "txn-invalid-token",
		TransactionEvent: &Event{
			ID:     "0",
			Status: "Pass",
		},
	}

	_, err := eventGenerator.CreateEvents(LogEvent{}, []LogEvent{dummyLogEvent}, time.Now(), nil, nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "bad response from AxwayId: 403 Forbidden", err.Error())
}

func TestCreateEvent(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(`{"access_token":"tok","expires_in":99999}`))
	}))
	defer s.Close()

	const tenantID = "tenant-create-event"
	const envID = "env-create-event"

	cases := map[string]struct {
		tenantID  string
		envID     string
		logEvent  LogEvent
		wantErr   string
		wantEvent string
	}{
		"missing environmentID returns error": {
			tenantID: "tenant-123",
			envID:    "",
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    "txn-guard",
				TransactionEvent: &Event{ID: "0", Status: "Pass"},
			},
			wantErr: "distribution.environment",
		},
		"leg event beat message is InsightsEvent JSON": {
			tenantID: tenantID,
			envID:    envID,
			logEvent: LogEvent{
				Type:             TypeTransactionEvent,
				TransactionID:    "txn-beat-leg",
				TransactionEvent: &Event{ID: "0", Status: "Pass"},
			},
			wantEvent: "api.transaction.event",
		},
		"summary event beat message is InsightsEvent JSON": {
			tenantID: tenantID,
			envID:    envID,
			logEvent: LogEvent{
				Type:               TypeTransactionSummary,
				TransactionID:      "txn-beat-sum",
				TransactionSummary: &Summary{Status: "Success"},
			},
			wantEvent: "api.transaction.summary",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := createMapperTestConfig(s.URL, tc.tenantID, "prod", "env-name", tc.envID)
			err := agent.Initialize(cfg.Central)
			assert.Nil(t, err)

			gen := &Generator{
				shouldAddFields:                false,
				shouldUseTrafficForAggregation: false,
				logger:                         log.NewFieldLogger(),
			}

			beatEvent, err := gen.createEvent(tc.logEvent, time.Now(), nil, nil, nil)
			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}

			assert.NoError(t, err)
			msg, ok := beatEvent.Fields["message"]
			assert.True(t, ok)

			var envelope map[string]interface{}
			err = json.Unmarshal([]byte(msg.(string)), &envelope)
			assert.NoError(t, err)
			assert.Equal(t, insightsEventVersion, envelope["version"])
			assert.Equal(t, tc.wantEvent, envelope["event"])
			assert.Equal(t, tc.tenantID, envelope["org"])
			assert.NotEmpty(t, envelope["id"])
		})
	}
}

func TestCreateEventsInOfflineMode(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusForbidden)
	}))
	defer s.Close()

	cfg := createOfflineMapperTestConfig("1111")
	agent.Initialize(cfg.Central)
	eventGenerator := NewEventGenerator()
	eventGenerator.SetUseTrafficForAggregation(false)
	dummySummaryEvent := LogEvent{
		EnvironmentID: cfg.Central.GetEnvironmentID(),
	}

	_, err := eventGenerator.CreateEvents(dummySummaryEvent, []LogEvent{}, time.Now(), nil, nil, nil)
	assert.Nil(t, err)
}
