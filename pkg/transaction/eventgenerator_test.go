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
			SubscriptionConfiguration: corecfg.NewSubscriptionConfig(),
			UsageReporting:            corecfg.NewUsageReporting("https://platform.xxx.com"),
			ReportActivityFrequency:   2 * time.Minute,
			APIValidationCronSchedule: "0 0 * * *",
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
	sampling.SetupSampling(sampling.DefaultConfig(), true)
	return cfg
}

func TestCreateEventWithValidTokenRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
		resp.Write([]byte(token))
	}))
	defer s.Close()

	cfg := createMapperTestConfig(s.URL, "1111", "aaa", "env1", "1111")
	// authCfg := cfg.Central.GetAuthConfig()
	err := agent.Initialize(cfg.Central)
	assert.Nil(t, err)

	eventGenerator := NewEventGenerator()
	dummyLogEvent := LogEvent{
		TenantID:      cfg.Central.GetTenantID(),
		Environment:   cfg.Central.GetAPICDeployment(),
		EnvironmentID: cfg.Central.GetEnvironmentID(),
	}
	eventFields := make(common.MapStr)
	eventFields["someKey.1"] = "someVal.1"
	eventFields["someKey.2"] = "someVal.2"
	eventFields["message"] = "existingMessage"

	event, _ := eventGenerator.CreateEvent(dummyLogEvent, time.Now(), nil, eventFields, nil)
	assert.NotNil(t, event)
	// Validate that existing fields are added to generated event
	assert.Equal(t, "someVal.1", event.Fields["someKey.1"])
	assert.Equal(t, "someVal.2", event.Fields["someKey.2"])

	msg := fmt.Sprintf("%v", event.Fields["message"])
	fields := event.Fields["fields"].(map[string]string)
	assert.NotNil(t, fields)
	assert.NotNil(t, msg)
	// Validate if message field from orgincal event fields is not included
	assert.NotEqual(t, "existingMessage", event.Fields["message"])
	var logEvent LogEvent
	json.Unmarshal([]byte(msg), &logEvent)
	assert.Equal(t, dummyLogEvent, logEvent)
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
	}

	_, err := eventGenerator.CreateEvent(dummyLogEvent, time.Now(), nil, nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "bad response from AxwayId: 403 Forbidden", err.Error())
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
