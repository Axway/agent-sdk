package metric

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd"
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
	// cfg.PublishMetricEvents = true
	return cfg
}

var accessToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0IiwiaWF0IjoxNjE0NjA0NzE0LCJleHAiOjE2NDYxNDA3MTQsImF1ZCI6InRlc3RhdWQiLCJzdWIiOiIxMjM0NTYiLCJvcmdfZ3VpZCI6IjEyMzQtMTIzNC0xMjM0LTEyMzQifQ.5Uqt0oFhMgmI-sLQKPGkHwknqzlTxv-qs9I_LmZ18LQ"

func TestMetricCollector(t *testing.T) {
	lighthouseEventCount := 0
	transactionCount := 0
	failUsageEvent := false
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"" + accessToken + "\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/lighthouse") {
			if failUsageEvent {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
			lighthouseEventCount++
			req.ParseMultipartForm(1 << 20)
			for _, fileHeaders := range req.MultipartForm.File {
				for _, fileHeader := range fileHeaders {
					file, err := fileHeader.Open()
					if err != nil {
						return
					}
					body, _ := ioutil.ReadAll(file)
					var usageEvent LighthouseUsageEvent
					json.Unmarshal(body, &usageEvent)
					for _, report := range usageEvent.Report {
						for _, usage := range report.Usage {
							transactionCount = transactionCount + int(usage)
						}
					}
				}
			}
		}
	}))

	defer s.Close()
	cfg := createCentralCfg(s.URL, "demo")
	cfg.LighthouseURL = s.URL + "/lighthouse"
	cfg.SetEnvironmentID("267bd671-e5e2-4679-bcc3-bbe7b70f30fd")
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	myCollector := createMetricCollector()
	metricCollector := myCollector.(*collector)
	metricCollector.orgGUID = metricCollector.getOrgGUID()
	jobs.GetJob(metricCollector.jobID)
	jobs.UnregisterJob(metricCollector.jobID)

	testCases := []struct {
		name                     string
		loopCount                int
		apiTransactionCount      []int
		failUsageEventOnServer   []bool
		expectedLHEvents         []int
		expectedTransactionCount []int
	}{
		// Success case
		{
			name:                     "WithLighthouse",
			loopCount:                1,
			apiTransactionCount:      []int{5},
			failUsageEventOnServer:   []bool{false},
			expectedLHEvents:         []int{1},
			expectedTransactionCount: []int{5},
		},
		// Success case with no usage report
		{
			name:                     "WithLighthouseNoUsageReport",
			loopCount:                1,
			apiTransactionCount:      []int{0},
			failUsageEventOnServer:   []bool{false},
			expectedLHEvents:         []int{0},
			expectedTransactionCount: []int{0},
		},
		// Test case with failing request to LH, the subsequent successful request should contain the total count since initial failure
		{
			name:                     "WithLighthouseWithFailure",
			loopCount:                3,
			apiTransactionCount:      []int{5, 10, 2},
			failUsageEventOnServer:   []bool{false, true, false},
			expectedLHEvents:         []int{1, 1, 2},
			expectedTransactionCount: []int{5, 5, 17},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			for l := 0; l < test.loopCount; l++ {
				for i := 0; i < test.apiTransactionCount[l]; i++ {
					metricCollector.AddMetric("111", "111", "200", 10, "", "")
				}
				failUsageEvent = test.failUsageEventOnServer[l]
				metricCollector.Execute()
				assert.Equal(t, test.expectedLHEvents[l], lighthouseEventCount)
				assert.Equal(t, test.expectedTransactionCount[l], transactionCount)
			}
			lighthouseEventCount = 0
			transactionCount = 0
			failUsageEvent = false
		})
	}
}
