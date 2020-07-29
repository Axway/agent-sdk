package transaction

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

type Config struct {
	Central corecfg.CentralConfig `config:"central"`
}

func createMapperTestConfig(authURL, tenantID, env, envName string) *Config {
	return &Config{
		Central: &corecfg.CentralConfiguration{
			TenantID:       tenantID,
			APICDeployment: env,
			Environment:    envName,
			Auth: &corecfg.AuthConfiguration{
				URL:        authURL,
				ClientID:   "test",
				Realm:      "Broker",
				PrivateKey: "testdata/private_key.pem",
				PublicKey:  "testdata/public_key",
				Timeout:    10 * time.Second,
			},
			SubscriptionApprovalWebhook: corecfg.NewWebhookConfig(),
		},
	}
}

func TestCreateEventWithValidTokenRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
		resp.Write([]byte(token))
	}))
	defer s.Close()

	cfg := createMapperTestConfig(s.URL, "1111", "aaa", "1111")
	authCfg := cfg.Central.GetAuthConfig()
	eventGenerator := NewEventGenerator(authCfg.GetTokenURL(), authCfg.GetAudience(), authCfg.GetPrivateKey(), authCfg.GetPublicKey(), "", authCfg.GetClientID(), authCfg.GetTimeout())
	dummyLogEvent := LogEvent{
		TenantID:      cfg.Central.GetTenantID(),
		Environment:   cfg.Central.GetAPICDeployment(),
		EnvironmentID: cfg.Central.GetEnvironmentID(),
	}
	event, _ := eventGenerator.CreateEvent(dummyLogEvent, time.Now(), nil, nil)
	assert.NotNil(t, event)
	msg := fmt.Sprintf("%v", event.Fields["message"])
	fields := event.Fields["fields"].(map[string]string)
	assert.NotNil(t, fields)
	assert.NotNil(t, msg)
	var logEvent LogEvent
	json.Unmarshal([]byte(msg), &logEvent)
	assert.Equal(t, dummyLogEvent, logEvent)
	assert.Equal(t, "somevalue", fields["token"])
	assert.Equal(t, "api-central-v8", fields["axway-target-flow"])
}

func TestCreateEventWithInvalidTokenRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusForbidden)
	}))
	defer s.Close()

	cfg := createMapperTestConfig(s.URL, "1111", "aaa", "1111")
	authCfg := cfg.Central.GetAuthConfig()
	eventGenerator := NewEventGenerator(authCfg.GetTokenURL(), authCfg.GetAudience(), authCfg.GetPrivateKey(), authCfg.GetPublicKey(), "", authCfg.GetClientID(), authCfg.GetTimeout())
	dummyLogEvent := LogEvent{
		TenantID:      cfg.Central.GetTenantID(),
		Environment:   cfg.Central.GetAPICDeployment(),
		EnvironmentID: cfg.Central.GetEnvironmentID(),
	}

	_, err := eventGenerator.CreateEvent(dummyLogEvent, time.Now(), nil, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "bad response from AxwayId: 403 Forbidden", err.Error())
}
