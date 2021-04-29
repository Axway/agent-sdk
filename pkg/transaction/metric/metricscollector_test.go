package metric

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/stretchr/testify/assert"
)

func createCentralCfg(url, env string) *config.CentralConfiguration {

	cfg := config.NewCentralConfig(config.DiscoveryAgent).(*config.CentralConfiguration)
	cfg.URL = url
	cfg.TenantID = "123456"
	cfg.Environment = env
	authCfg := cfg.Auth.(*config.AuthConfiguration)
	authCfg.URL = url + "/auth"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "DOSA_1111"
	authCfg.PrivateKey = "../../transaction/testdata/private_key.pem"
	authCfg.PublicKey = "../../transaction/testdata/public_key"
	cfg.PublisUsageEvents = true
	cfg.PublishMetricEvents = true
	return cfg
}

var accessToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0IiwiaWF0IjoxNjE0NjA0NzE0LCJleHAiOjE2NDYxNDA3MTQsImF1ZCI6InRlc3RhdWQiLCJzdWIiOiIxMjM0NTYiLCJvcmdfZ3VpZCI6IjEyMzQtMTIzNC0xMjM0LTEyMzQifQ.5Uqt0oFhMgmI-sLQKPGkHwknqzlTxv-qs9I_LmZ18LQ"

func TestMetricCollector(t *testing.T) {
	gatekeeperEventCount := 0
	lighthouseEventCount := 0
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"" + accessToken + "\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/gatekeeper") {
			gatekeeperEventCount++
		}
		if strings.Contains(req.RequestURI, "/lighthouse") {
			lighthouseEventCount++
		}
	}))

	defer s.Close()

	testCases := []struct {
		name              string
		lighthouse        bool
		expectedGKEvents1 int
		expectedLHEvents1 int
		expectedGKEvents2 int
		expectedLHEvents2 int
	}{
		{
			name:              "WithLighthouse",
			lighthouse:        true,
			expectedGKEvents1: 0,
			expectedLHEvents1: 1,
			expectedGKEvents2: 0,
			expectedLHEvents2: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gatekeeperEventCount = 0
			lighthouseEventCount = 0

			cfg := createCentralCfg(s.URL, "demo")
			cfg.GatekeeperURL = s.URL + "/gatekeeper"
			if test.lighthouse {
				cfg.LighthouseURL = s.URL + "/lighthouse"
			}
			cfg.PlatformEnvironmentID = "267bd671-e5e2-4679-bcc3-bbe7b70f30fd"
			cfg.DataplaneType = "Azure"
			agent.Initialize(cfg)
			eventChannel := make(chan interface{}, 1028)
			myCollector := NewMetricCollector(eventChannel)
			metricCollector := myCollector.(*collector)
			jobs.GetJob(metricCollector.jobID)
			NewMetricPublisher(eventChannel)

			metricCollector.orgGUID = metricCollector.getOrgGUID()

			metricCollector.AddMetric("111", "111", "200", 10, "", "")
			metricCollector.AddMetric("111", "111", "200", 20, "", "")
			metricCollector.AddMetric("111", "111", "200", 30, "", "")
			metricCollector.AddMetric("111", "111", "401", 10, "", "")
			metricCollector.AddMetric("111", "111", "401", 20, "", "")

			metricCollector.AddMetric("222", "222", "200", 5, "", "")
			metricCollector.AddMetric("222", "222", "200", 5, "", "")

			metricCollector.endTime = time.Now()
			metricCollector.generateEvents()
			metricCollector.startTime = time.Now()
			time.Sleep(1 * time.Second)
			assert.Equal(t, test.expectedGKEvents1, gatekeeperEventCount)
			assert.Equal(t, test.expectedLHEvents1, lighthouseEventCount)

			gatekeeperEventCount = 0
			lighthouseEventCount = 0
			metricCollector.AddMetric("111", "111", "200", 5, "", "")
			metricCollector.AddMetric("111", "111", "200", 15, "", "")
			metricCollector.AddMetric("111", "111", "401", 15, "", "")
			metricCollector.AddMetric("111", "111", "401", 5, "", "")
			metricCollector.AddMetric("111", "111", "401", 120, "", "")

			metricCollector.AddMetric("222", "222", "200", 5, "", "")
			metricCollector.AddMetric("222", "222", "200", 50, "", "")
			metricCollector.AddMetric("222", "222", "400", 15, "", "")
			metricCollector.endTime = time.Now()
			metricCollector.generateEvents()
			time.Sleep(1 * time.Second)
			assert.Equal(t, test.expectedGKEvents2, gatekeeperEventCount)
			assert.Equal(t, test.expectedLHEvents2, lighthouseEventCount)
		})
	}
}
