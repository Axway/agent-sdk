package metric

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/stretchr/testify/assert"
)

func createCentralCfg(url, env string) *config.CentralConfiguration {

	cfg := config.NewCentralConfig(config.TraceabilityAgent).(*config.CentralConfiguration)
	cfg.URL = url
	cfg.SingleURL = url
	cfg.TenantID = "123456"
	cfg.Environment = env
	cfg.APICDeployment = "test"
	authCfg := cfg.Auth.(*config.AuthConfiguration)
	authCfg.URL = url + "/auth"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "serviceaccount_1234"
	authCfg.PrivateKey = "../../transaction/testdata/private_key.pem"
	authCfg.PublicKey = "../../transaction/testdata/public_key"
	usgCfg := cfg.UsageReporting.(*config.UsageReportingConfiguration)
	usgCfg.Publish = true
	usgCfg.PublishMetric = true
	return cfg
}

var accessToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0IiwiaWF0IjoxNjE0NjA0NzE0LCJleHAiOjE2NDYxNDA3MTQsImF1ZCI6InRlc3RhdWQiLCJzdWIiOiIxMjM0NTYiLCJvcmdfZ3VpZCI6IjEyMzQtMTIzNC0xMjM0LTEyMzQifQ.5Uqt0oFhMgmI-sLQKPGkHwknqzlTxv-qs9I_LmZ18LQ"

var teamID = "team123"

type testHTTPServer struct {
	lighthouseEventCount int
	transactionCount     int
	transactionVolume    int
	failUsageEvent       bool
	server               *httptest.Server
}

func (s *testHTTPServer) startServer() {
	s.server = httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"" + accessToken + "\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/lighthouse") {
			if s.failUsageEvent {
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
			s.lighthouseEventCount++
			req.ParseMultipartForm(1 << 20)
			for _, fileHeaders := range req.MultipartForm.File {
				for _, fileHeader := range fileHeaders {
					file, err := fileHeader.Open()
					if err != nil {
						return
					}
					body, _ := io.ReadAll(file)
					var usageEvent LighthouseUsageEvent
					json.Unmarshal(body, &usageEvent)
					fmt.Printf("\n\n %+v \n\n", usageEvent)
					for _, report := range usageEvent.Report {
						for usageType, usage := range report.Usage {
							if strings.Index(usageType, "Transactions") > 0 {
								s.transactionCount += int(usage)
							} else if strings.Index(usageType, "Volume") > 0 {
								s.transactionVolume += int(usage)
							}
						}
					}
				}
			}
		}
	}))
}

func (s *testHTTPServer) closeServer() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *testHTTPServer) resetConfig() {
	s.lighthouseEventCount = 0
	s.transactionCount = 0
	s.transactionVolume = 0
	s.failUsageEvent = false
}

func (s *testHTTPServer) resetOffline(myCollector Collector) {
	events := myCollector.(*collector).reports.loadEvents()
	events.Report = make(map[string]LighthouseUsageReport)
	myCollector.(*collector).reports.updateEvents(events)
	s.resetConfig()
}

func cleanUpCachedMetricFile() {
	os.RemoveAll("./cache")
}

func cleanUpReportfiles() {
	os.RemoveAll("./reports")
}

func createRI(group, kind, id, name string, subRes map[string]interface{}) *apiv1.ResourceInstance {
	return &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				ID: id,
			},
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Group: group,
					Kind:  kind,
				},
			},
			SubResources: subRes,
			Name:         name,
		},
	}
}

func createAPIServiceInstance(id, name string, apiID string) *apiv1.ResourceInstance {
	sub := map[string]interface{}{
		defs.XAgentDetails: map[string]interface{}{
			defs.AttrExternalAPIID: apiID,
		},
	}
	return createRI(management.APIServiceInstanceGVK().Group, management.APIServiceInstanceGVK().Kind, id, name, sub)
}

func createManagedApplication(id, name, consumerOrgID string) *apiv1.ResourceInstance {
	var marketplaceSubRes map[string]interface{}
	if consumerOrgID != "" {
		marketplaceSubRes = map[string]interface{}{
			"marketplace": management.ManagedApplicationMarketplace{
				Name: name,
				Resource: management.ManagedApplicationMarketplaceResource{
					Owner: &apiv1.Owner{
						Organization: apiv1.Organization{
							ID: consumerOrgID,
						},
					},
				},
			},
		}
	}
	return createRI(management.ManagedApplicationGVK().Group, management.ManagedApplicationGVK().Kind, id, name, marketplaceSubRes)
}

func createAccessRequest(id, name, appName, instanceID, instanceName, subscriptionName string) *apiv1.ResourceInstance {
	ar := &management.AccessRequest{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				ID: id,
				References: []apiv1.Reference{
					{
						Group: management.APIServiceInstanceGVK().Group,
						Kind:  management.APIServiceInstanceGVK().Kind,
						ID:    instanceID,
						Name:  instanceName,
					},
				},
			},
			Name: name,
		},
		Spec: management.AccessRequestSpec{
			ManagedApplication: appName,
			ApiServiceInstance: instanceName,
		},
		References: []interface{}{
			management.AccessRequestReferencesSubscription{
				Kind: defs.Subscription,
				Name: "catalog/" + subscriptionName,
			},
		},
	}
	ri, _ := ar.AsInstance()
	return ri
}

func TestMetricCollector(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + "/lighthouse"
	cfg.UsageReporting.(*config.UsageReportingConfiguration).PublishMetric = true
	cfg.SetEnvironmentID("267bd671-e5e2-4679-bcc3-bbe7b70f30fd")
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	cm := agent.GetCacheManager()
	cm.AddAPIServiceInstance(createAPIServiceInstance("inst-1", "instance-1", "111"))

	cm.AddManagedApplication(createManagedApplication("app-1", "managed-app-1", ""))
	cm.AddManagedApplication(createManagedApplication("app-2", "managed-app-2", "test-consumer-org"))

	cm.AddAccessRequest(createAccessRequest("ac-1", "access-req-1", "managed-app-1", "inst-1", "instance-1", "subscription-1"))
	cm.AddAccessRequest(createAccessRequest("ac-2", "access-req-2", "managed-app-2", "inst-1", "instance-1", "subscription-2"))

	myCollector := createMetricCollector()
	metricCollector := myCollector.(*collector)

	testCases := []struct {
		name                      string
		loopCount                 int
		retryBatchCount           int
		apiTransactionCount       []int
		failUsageEventOnServer    []bool
		expectedLHEvents          []int
		expectedTransactionCount  []int
		trackVolume               bool
		expectedTransactionVolume []int
		expectedMetricEventsAcked int
		appName                   string
	}{
		// Success case with no app detail
		{
			name:                      "WithLighthouse",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1, // API metric + no Provider subscription metric
		},
		// Success case
		{
			name:                      "WithLighthouse",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1, // API metric + Provider subscription metric
			appName:                   "managed-app-1",
		},
		// Success case with consumer metric event
		{
			name:                      "WithLighthouse",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1, // API metric + Provider + Consumer subscription metric
			appName:                   "managed-app-2",
		},
		// Success case with no usage report
		{
			name:                      "WithLighthouseNoUsageReport",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{0},
			failUsageEventOnServer:    []bool{false},
			expectedLHEvents:          []int{0},
			expectedTransactionCount:  []int{0},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 0,
		},
		// Test case with failing request to LH, the subsequent successful request should contain the total count since initial failure
		{
			name:                      "WithLighthouseWithFailure",
			loopCount:                 3,
			retryBatchCount:           0,
			apiTransactionCount:       []int{5, 10, 12},
			failUsageEventOnServer:    []bool{false, true, false, false},
			expectedLHEvents:          []int{1, 1, 2},
			expectedTransactionCount:  []int{5, 5, 17},
			trackVolume:               true,
			expectedTransactionVolume: []int{50, 50, 170},
			expectedMetricEventsAcked: 1,
			appName:                   "unknown",
		},
		// Success case, retry metrics
		{
			name:                      "WithLighthouseAndMetricRetry",
			loopCount:                 1,
			retryBatchCount:           1,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               true,
			expectedTransactionVolume: []int{50},
			expectedMetricEventsAcked: 1,
			appName:                   "unknown",
		},
		// Retry limit hit
		{
			name:                      "WithLighthouseAndFailedMetric",
			loopCount:                 1,
			retryBatchCount:           4,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfg.SetAxwayManaged(test.trackVolume)
			setupMockClient(test.retryBatchCount)
			for l := 0; l < test.loopCount; l++ {
				fmt.Printf("\n\nTransaction Info: %+v\n\n", test.apiTransactionCount[l])
				for i := 0; i < test.apiTransactionCount[l]; i++ {
					metricDetail := Detail{
						APIDetails: APIDetails{"111", "111", 1, teamID, "", "", ""},
						StatusCode: "200",
						Duration:   10,
						Bytes:      10,
						AppDetails: AppDetails{
							ID:   "111",
							Name: test.appName,
						},
					}
					metricCollector.AddMetricDetail(metricDetail)
				}
				s.failUsageEvent = test.failUsageEventOnServer[l]

				metricCollector.Execute()
				metricCollector.publisher.Execute()
				assert.Equal(t, test.expectedLHEvents[l], s.lighthouseEventCount)
				assert.Equal(t, test.expectedTransactionCount[l], s.transactionCount)
				assert.Equal(t, test.expectedTransactionVolume[l], s.transactionVolume)
				assert.Equal(t, test.expectedMetricEventsAcked, myMockClient.(*MockClient).eventsAcked)
			}
			s.resetConfig()
		})
	}
}

func TestMetricCollectorCache(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()

	testCases := []struct {
		name        string
		trackVolume bool
	}{
		{
			name:        "UsageOnly",
			trackVolume: false,
		},
		{
			name:        "UsageAndVolume",
			trackVolume: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfg := createCentralCfg(s.server.URL, "demo")
			cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + "/lighthouse"
			cfg.SetEnvironmentID("267bd671-e5e2-4679-bcc3-bbe7b70f30fd")
			cfg.SetAxwayManaged(test.trackVolume)
			cmd.BuildDataPlaneType = "Azure"
			agent.Initialize(cfg)

			traceability.SetDataDirPath(".")
			myCollector := createMetricCollector()
			metricCollector := myCollector.(*collector)

			metricCollector.AddMetric(APIDetails{"111", "111", 1, teamID, "", "", ""}, "200", 5, 10, "")
			metricCollector.AddMetric(APIDetails{"111", "111", 1, teamID, "", "", ""}, "200", 10, 10, "")
			metricCollector.Execute()
			metricCollector.publisher.Execute()
			metricCollector.AddMetric(APIDetails{"111", "111", 1, teamID, "", "", ""}, "401", 15, 10, "")
			metricCollector.AddMetric(APIDetails{"222", "222", 1, teamID, "", "", ""}, "200", 20, 10, "")
			metricCollector.AddMetric(APIDetails{"222", "222", 1, teamID, "", "", ""}, "200", 10, 10, "")

			// No event generation/publish, store the cache
			metricCollector.storage.save()
			// Validate only one usage report sent with first 2 transactions
			assert.Equal(t, 1, s.lighthouseEventCount)
			assert.Equal(t, 2, s.transactionCount)
			if test.trackVolume {
				assert.Equal(t, 20, s.transactionVolume)
			}
			s.resetConfig()

			// Recreate the collector that loads the stored metrics, so 3 transactions
			myCollector = createMetricCollector()
			metricCollector = myCollector.(*collector)

			metricCollector.AddMetric(APIDetails{"111", "111", 1, teamID, "", "", ""}, "200", 5, 10, "")
			metricCollector.AddMetric(APIDetails{"111", "111", 1, teamID, "", "", ""}, "200", 10, 10, "")
			metricCollector.AddMetric(APIDetails{"111", "111", 1, teamID, "", "", ""}, "401", 15, 10, "")
			metricCollector.AddMetric(APIDetails{"222", "222", 1, teamID, "", "", ""}, "200", 20, 10, "")
			metricCollector.AddMetric(APIDetails{"222", "222", 1, teamID, "", "", ""}, "200", 10, 10, "")

			metricCollector.Execute()
			metricCollector.publisher.Execute()
			// Validate only one usage report sent with 3 previous transactions and 5 new transactions
			assert.Equal(t, 1, s.lighthouseEventCount)
			assert.Equal(t, 8, s.transactionCount)
			if test.trackVolume {
				assert.Equal(t, 80, s.transactionVolume)
			}

			s.resetConfig()
			// Recreate the collector that loads the stored metrics, 0 transactions
			myCollector = createMetricCollector()
			metricCollector = myCollector.(*collector)

			metricCollector.Execute()
			// Validate only no usage report sent as no previous or new transactions
			assert.Equal(t, 0, s.lighthouseEventCount)
			assert.Equal(t, 0, s.transactionCount)
			if test.trackVolume {
				assert.Equal(t, 0, s.transactionVolume)
			}
		})
	}
}

func TestOfflineMetricCollector(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + "/lighthouse"
	cfg.EnvironmentID = "267bd671-e5e2-4679-bcc3-bbe7b70f30fd"
	cmd.BuildDataPlaneType = "Azure"
	usgCfg := cfg.UsageReporting.(*config.UsageReportingConfiguration)
	usgCfg.Offline = true
	agent.Initialize(cfg)

	testCases := []struct {
		name                string
		loopCount           int
		apiTransactionCount []int
		generateReport      bool
	}{
		{
			name:                "NoReports",
			loopCount:           0,
			apiTransactionCount: []int{},
			generateReport:      true,
		},
		{
			name:                "OneReport",
			loopCount:           1,
			apiTransactionCount: []int{10},
			generateReport:      true,
		},
		{
			name:                "ThreeReports",
			loopCount:           3,
			apiTransactionCount: []int{5, 10, 2},
			generateReport:      true,
		},
		{
			name:                "ThreeReportsNoUsage",
			loopCount:           3,
			apiTransactionCount: []int{0, 0, 0},
			generateReport:      true,
		},
		{
			name:                "SixReports",
			loopCount:           6,
			apiTransactionCount: []int{5, 10, 2, 0, 3, 9},
			generateReport:      true,
		},
	}

	for testNum, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			startDate := time.Date(2021, 1, 31, 12, 30, 0, 0, time.Local)
			setupMockClient(0)
			testLoops := 0
			now = func() time.Time {
				next := startDate.Add(time.Hour * time.Duration(testLoops))
				fmt.Println(next.Format(ISO8601))
				return next
			}

			validateEvents := func(report LighthouseUsageEvent) {
				for j := 0; j < test.loopCount; j++ {
					reportKey := startDate.Add(time.Duration(j-1) * time.Hour).Format(ISO8601)
					assert.Equal(t, cmd.BuildDataPlaneType, report.Report[reportKey].Product)
					assert.Equal(t, test.apiTransactionCount[j], int(report.Report[reportKey].Usage[cmd.BuildDataPlaneType+".Transactions"]))
				}
				// validate granularity when reports not empty
				if test.loopCount != 0 {
					assert.Equal(t, int(time.Hour.Milliseconds()), report.Granularity)
					cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + "/lighthouse"
					assert.Equal(t, cfg.UsageReporting.GetURL()+schemaPath, report.SchemaID)
					assert.Equal(t, cfg.GetEnvironmentID(), report.EnvID)
				}
			}

			myCollector := createMetricCollector()
			metricCollector := myCollector.(*collector)

			reportGenerator := metricCollector.reports
			publisher := metricCollector.publisher
			for testLoops < test.loopCount {
				for i := 0; i < test.apiTransactionCount[testLoops]; i++ {
					metricCollector.AddMetric(APIDetails{"111", "111", 1, "team123", "", "", ""}, "200", 10, 10, "")
				}
				metricCollector.Execute()
				testLoops++
			}

			// Get the usage reports from the cache and validate
			events := myCollector.(*collector).reports.loadEvents()
			validateEvents(events)

			// generate the report file
			publisher.Execute()

			expectedFile := reportGenerator.generateReportPath(ISO8601Time(startDate), testNum-1)
			if test.loopCount == 0 {
				// no report expected, end the test here
				expectedFile = reportGenerator.generateReportPath(ISO8601Time(startDate), 0)
				assert.NoFileExists(t, expectedFile)
				return
			}

			// validate the file exists and open it
			assert.FileExists(t, expectedFile)
			data, err := os.ReadFile(expectedFile)
			assert.Nil(t, err)

			// unmarshall it
			var reportEvents LighthouseUsageEvent
			err = json.Unmarshal(data, &reportEvents)
			assert.Nil(t, err)
			assert.NotNil(t, reportEvents)

			// validate event in generated reports
			validateEvents(reportEvents)

			s.resetOffline(myCollector)
		})
	}
	cleanUpReportfiles()
}
