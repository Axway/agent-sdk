package metric

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/stretchr/testify/assert"
)

var (
	apiDetails1 = models.APIDetails{
		ID:                 "111",
		Name:               "111",
		Revision:           1,
		TeamID:             teamID,
		APIServiceInstance: "",
		Stage:              "",
		Version:            "",
	}
	apiDetails2 = models.APIDetails{
		ID:                 "222",
		Name:               "222",
		Revision:           1,
		TeamID:             teamID,
		APIServiceInstance: "",
		Stage:              "",
		Version:            "",
	}
	traceStatus = healthcheck.OK
	appDetails1 = models.AppDetails{
		ID:            "111",
		Name:          "111",
		ConsumerOrgID: "org-id-111",
	}
)

func getFutureTime() time.Time {
	return time.Now().Add(10 * time.Minute)
}

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
	metricCfg := cfg.MetricReporting.(*config.MetricReportingConfiguration)
	metricCfg.Publish = true
	// metricCfg.Schedule = "1 * * * * * *"
	return cfg
}

var accessToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0IiwiaWF0IjoxNjE0NjA0NzE0LCJleHAiOjE2NDYxNDA3MTQsImF1ZCI6InRlc3RhdWQiLCJzdWIiOiIxMjM0NTYiLCJvcmdfZ3VpZCI6IjEyMzQtMTIzNC0xMjM0LTEyMzQifQ.5Uqt0oFhMgmI-sLQKPGkHwknqzlTxv-qs9I_LmZ18LQ"

var teamID = "team123"

type testHTTPServer struct {
	lighthouseEventCount int
	transactionCount     int
	transactionVolume    int
	failUsageEvent       bool
	failUsageResponse    *UsageResponse
	server               *httptest.Server
	reportCount          int
	givenGranularity     int
	eventTimestamp       ISO8601Time
}

func (s *testHTTPServer) startServer() {
	s.server = httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"" + accessToken + "\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/lighthouse") {
			if s.failUsageEvent {
				if s.failUsageResponse != nil {
					b, _ := json.Marshal(*s.failUsageResponse)
					resp.WriteHeader(s.failUsageResponse.StatusCode)
					resp.Write(b)
					return
				}
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
					var usageEvent UsageEvent
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
						s.reportCount++
					}
					s.givenGranularity = usageEvent.Granularity
					s.eventTimestamp = usageEvent.Timestamp
				}
			}
		}
		resp.WriteHeader(202)
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
	s.givenGranularity = 0
	s.reportCount = 0
}

func (s *testHTTPServer) resetOffline(myCollector Collector) {
	events := myCollector.(*collector).reports.loadEvents()
	events.Report = make(map[string]UsageReport)
	myCollector.(*collector).reports.updateEvents(events)
	s.resetConfig()
}

func cleanUpCachedMetricFile() {
	os.RemoveAll("./cache")
}

func generateMockReports(transactionPerReport []int) UsageEvent {
	jsonStructure := `{"envId":"267bd671-e5e2-4679-bcc3-bbe7b70f30fd","timestamp":"2024-02-14T10:30:00+02:00","granularity":3600000,"schemaId":"http://127.0.0.1:53493/lighthouse/api/v1/report.schema.json","report":{},"meta":{"AgentName":"","AgentVersion":""}}`
	var mockEvent UsageEvent
	json.Unmarshal([]byte(jsonStructure), &mockEvent)
	startDate := time.Time(mockEvent.Timestamp)
	nextTime := func(i int) string {
		next := startDate.Add(time.Hour * time.Duration(-i-1))
		return next.Format(ISO8601)
	}
	for i, transaction := range transactionPerReport {
		mockEvent.Report[nextTime(i)] = UsageReport{
			Product: "Azure",
			Usage:   map[string]int64{"Azure.Transactions": int64(transaction)},
		}
	}
	return mockEvent
}

func cleanUpReportFiles() {
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

func runTestHealthcheck() {
	// register a healthcheck
	healthcheck.RegisterHealthcheck("Traceability", traceability.HealthCheckEndpoint,
		func(name string) *healthcheck.Status {
			return &healthcheck.Status{Result: traceStatus}
		},
	)
	healthcheck.RunChecks()
}

func TestMetricCollector(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + "/lighthouse"
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
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
		failUsageResponseOnServer []*UsageResponse
		expectedLHEvents          []int
		expectedTransactionCount  []int
		trackVolume               bool
		expectedTransactionVolume []int
		expectedMetricEventsAcked int
		appName                   string
		publishPrior              bool
		hcStatus                  healthcheck.StatusLevel
		skipWaitForPub            bool
	}{
		// Success case with no usage report
		{
			name:                      "WithUsageNoUsageReport",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{0},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{0},
			expectedTransactionCount:  []int{0},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 0,
			skipWaitForPub:            true,
		},
		// Success case with no app detail
		{
			name:                      "WithUsage",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1, // API metric + no Provider subscription metric
		},
		{
			name:                      "WithUsageWithPriorPublish",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1, // API metric + no Provider subscription metric
			publishPrior:              true,
		},
		// Success case
		{
			name:                      "WithUsage",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1, // API metric + Provider subscription metric
			appName:                   "managed-app-1",
		},
		// Success case with consumer metric event
		{
			name:                      "WithUsage",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1, // API metric + Provider + Consumer subscription metric
			appName:                   "managed-app-2",
		},
		// Test case with failing request to LH, the subsequent successful request should contain the total count since initial failure
		{
			name:                   "WithUsageWithFailure",
			loopCount:              3,
			retryBatchCount:        0,
			apiTransactionCount:    []int{5, 10, 12},
			failUsageEventOnServer: []bool{false, true, false, false},
			failUsageResponseOnServer: []*UsageResponse{
				nil, {
					Description: "Regular failure",
					StatusCode:  400,
					Success:     false,
				},
				nil,
				nil,
			},
			expectedLHEvents:          []int{1, 1, 2},
			expectedTransactionCount:  []int{5, 5, 17},
			trackVolume:               true,
			expectedTransactionVolume: []int{50, 50, 170},
			expectedMetricEventsAcked: 1,
			appName:                   "unknown",
		},
		// Test case with failing request to LH, no subsequent request triggered.
		{
			name:                   "WithUsageWithFailureWithSpecificDescription",
			loopCount:              3,
			retryBatchCount:        0,
			apiTransactionCount:    []int{1, 1, 1},
			failUsageEventOnServer: []bool{true, true, false},
			failUsageResponseOnServer: []*UsageResponse{
				{
					Description: "The file exceeds the maximum upload size of 454545",
					StatusCode:  400,
					Success:     false,
				},
				{
					Description: "Environment ID not found",
					StatusCode:  404,
					Success:     false,
				},
				nil,
			},
			expectedLHEvents:          []int{0, 0, 1},
			expectedTransactionCount:  []int{0, 0, 1},
			trackVolume:               true,
			expectedTransactionVolume: []int{0, 0, 10},
			expectedMetricEventsAcked: 1,
			appName:                   "unknown",
		},
		// Retry limit hit
		{
			name:                      "WithUsageAndFailedMetric",
			loopCount:                 1,
			retryBatchCount:           4,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 0,
		},
		// Traceability healthcheck failing, nothing reported
		{
			name:                      "WithUsageTraceabilityNotConnected",
			loopCount:                 1,
			retryBatchCount:           0,
			apiTransactionCount:       []int{0},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{0},
			expectedTransactionCount:  []int{0},
			trackVolume:               false,
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 0, // API metric + Provider subscription metric
			appName:                   "managed-app-1",
			hcStatus:                  healthcheck.FAIL,
			skipWaitForPub:            true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.hcStatus != "" {
				traceStatus = test.hcStatus
			}
			runTestHealthcheck()
			metricCollector.metricMap = make(map[string]map[string]map[string]map[string]*centralMetric)
			cfg.SetAxwayManaged(test.trackVolume)
			testClient := setupMockClient(test.retryBatchCount)
			mockClient := testClient.(*MockClient)

			for l := 0; l < test.loopCount; l++ {
				fmt.Printf("\n\nTransaction Info: %+v\n\n", test.apiTransactionCount[l])
				for i := 0; i < test.apiTransactionCount[l]; i++ {
					metricDetail := Detail{
						APIDetails: apiDetails1,
						StatusCode: "200",
						Duration:   10,
						Bytes:      10,
						AppDetails: models.AppDetails{
							ID:   "111",
							Name: test.appName,
						},
					}
					metricCollector.AddMetricDetail(metricDetail)
				}
				s.failUsageEvent = test.failUsageEventOnServer[l]
				s.failUsageResponse = test.failUsageResponseOnServer[l]
				if test.publishPrior {
					metricCollector.usagePublisher.Execute()
					metricCollector.Execute()
				} else {
					metricCollector.Execute()
					metricCollector.usagePublisher.Execute()
				}
			}
			assert.Equal(t, test.expectedMetricEventsAcked, mockClient.eventsAcked)
			s.resetConfig()
		})
	}
}

func TestConcurrentMetricCollectorEvents(t *testing.T) {
	// this test has no assertions it is to ensure concurrent map writes do not occur while collecting metrics
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + "/lighthouse"
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
	cfg.SetEnvironmentID("267bd671-e5e2-4679-bcc3-bbe7b70f30fd")
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)
	myCollector := createMetricCollector()
	metricCollector := myCollector.(*collector)
	traceStatus = healthcheck.OK
	runTestHealthcheck()

	apiDetails := []models.APIDetails{
		{ID: "000", Name: "000", Revision: 1, TeamID: teamID},
		{ID: "111", Name: "111", Revision: 1, TeamID: teamID},
		{ID: "222", Name: "222", Revision: 1, TeamID: teamID},
		{ID: "333", Name: "333", Revision: 1, TeamID: teamID},
		{ID: "444", Name: "444", Revision: 1, TeamID: teamID},
		{ID: "555", Name: "555", Revision: 1, TeamID: teamID},
		{ID: "666", Name: "666", Revision: 1, TeamID: teamID},
		{ID: "777", Name: "777", Revision: 1, TeamID: teamID},
		{ID: "888", Name: "888", Revision: 1, TeamID: teamID},
		{ID: "999", Name: "999", Revision: 1, TeamID: teamID},
	}
	appDetails := []models.AppDetails{
		{ID: "000", Name: "app0"},
		{ID: "111", Name: "app1"},
		{ID: "222", Name: "app2"},
		{ID: "333", Name: "app3"},
		{ID: "444", Name: "app4"},
		{ID: "555", Name: "app5"},
		{ID: "666", Name: "app6"},
		{ID: "777", Name: "app7"},
		{ID: "888", Name: "app8"},
		{ID: "999", Name: "app9"},
	}

	codes := []string{"200", "201", "300", "301", "400", "401", "500"}

	details := []Detail{}

	// load a bunch of different api details
	for _, api := range apiDetails {
		for _, app := range appDetails {
			for _, code := range codes {
				details = append(details, Detail{APIDetails: api, AppDetails: app, StatusCode: code})
			}
		}
	}

	// add all metrics via go routines
	wg := sync.WaitGroup{}
	transactionCount := 100
	wg.Add(len(details) * transactionCount)

	for j := range details {
		for i := 0; i < transactionCount; i++ {
			go func(dets Detail) {
				defer wg.Done()
				metricCollector.AddMetricDetail(dets)
			}(details[j])
		}
	}

	wg.Wait()
}

func TestMetricCollectorUsageAggregation(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + "/lighthouse"
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
	cfg.SetEnvironmentID("267bd671-e5e2-4679-bcc3-bbe7b70f30fd")
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	// setup the cache for handling custom metrics
	cm := agent.GetCacheManager()
	cm.AddAPIServiceInstance(createAPIServiceInstance("inst-1", "instance-1", "111"))

	cm.AddManagedApplication(createManagedApplication("app-1", "managed-app-1", ""))
	cm.AddManagedApplication(createManagedApplication("app-2", "managed-app-2", "test-consumer-org"))

	cm.AddAccessRequest(createAccessRequest("ac-1", "access-req-1", "managed-app-1", "inst-1", "instance-1", "subscription-1"))
	cm.AddAccessRequest(createAccessRequest("ac-2", "access-req-2", "managed-app-2", "inst-1", "instance-1", "subscription-2"))

	traceStatus = healthcheck.OK
	runTestHealthcheck()

	testCases := []struct {
		name                      string
		transactionsPerReport     []int
		expectedTransactionCount  int
		expectedTransactionVolume int
		expectedGranularity       int
		expectedReportCount       int
	}{
		{
			name:                     "FourReports",
			transactionsPerReport:    []int{3, 4, 5, 6},
			expectedTransactionCount: 18,
			expectedGranularity:      4 * int(time.Hour/time.Millisecond),
		},
		{
			name:                     "SevenReports",
			transactionsPerReport:    []int{1, 2, 3, 4, 5, 6, 7},
			expectedTransactionCount: 28,
			expectedGranularity:      7 * int(time.Hour/time.Millisecond),
		},
		{
			name:                     "OneReport",
			transactionsPerReport:    []int{1},
			expectedTransactionCount: 1,
			expectedGranularity:      1 * int(time.Hour/time.Millisecond),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfg.SetAxwayManaged(false)
			setupMockClient(0)
			myCollector := createMetricCollector()
			metricCollector := myCollector.(*collector)
			metricCollector.usagePublisher.schedule = "* * * * *"
			metricCollector.usagePublisher.report.currTimeFunc = getFutureTime

			mockReports := generateMockReports(test.transactionsPerReport)
			b, _ := json.Marshal(mockReports)
			metricCollector.reports.reportCache.Set("lighthouse_events", string(b))
			now = func() time.Time {
				return time.Time(mockReports.Timestamp)
			}
			metricCollector.usagePublisher.Execute()
			assert.Equal(t, test.expectedTransactionCount, s.transactionCount)
			assert.Equal(t, 1, s.reportCount)
			assert.Equal(t, test.expectedGranularity, s.givenGranularity)
			assert.Equal(t, ISO8601Time(now()), s.eventTimestamp)
			s.resetConfig()
		})
	}
	cleanUpReportFiles()
}

func TestMetricCollectorCache(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()

	traceStatus = healthcheck.OK
	runTestHealthcheck()

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
			metricCollector.usagePublisher.schedule = "* * * * *"
			metricCollector.usagePublisher.report.currTimeFunc = getFutureTime

			metricCollector.AddMetric(apiDetails1, "200", 5, 10, "")
			metricCollector.AddMetric(apiDetails1, "200", 10, 10, "")
			metricCollector.Execute()
			metricCollector.usagePublisher.Execute()
			metricCollector.AddMetric(apiDetails1, "401", 15, 10, "")
			metricCollector.AddMetric(apiDetails2, "200", 20, 10, "")
			metricCollector.AddMetric(apiDetails2, "200", 10, 10, "")

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
			metricCollector.usagePublisher.schedule = "* * * * *"
			metricCollector.usagePublisher.report.currTimeFunc = getFutureTime

			metricCollector.AddMetric(apiDetails1, "200", 5, 10, "")
			metricCollector.AddMetric(apiDetails1, "200", 10, 10, "")
			metricCollector.AddMetric(apiDetails1, "401", 15, 10, "")
			metricCollector.AddMetric(apiDetails2, "200", 20, 10, "")
			metricCollector.AddMetric(apiDetails2, "200", 10, 10, "")

			metricCollector.Execute()
			metricCollector.usagePublisher.Execute()
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
			metricCollector.usagePublisher.schedule = "* * * * *"
			metricCollector.usagePublisher.report.currTimeFunc = getFutureTime

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

	traceStatus = healthcheck.OK
	runTestHealthcheck()

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

			validateEvents := func(report UsageEvent) {
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
			publisher := metricCollector.usagePublisher
			for testLoops < test.loopCount {
				for i := 0; i < test.apiTransactionCount[testLoops]; i++ {
					myCollector.AddMetric(apiDetails1, "200", 10, 10, "")
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
			var reportEvents UsageEvent
			err = json.Unmarshal(data, &reportEvents)
			assert.Nil(t, err)
			assert.NotNil(t, reportEvents)

			// validate event in generated reports
			validateEvents(reportEvents)

			s.resetOffline(myCollector)
		})
	}
	cleanUpReportFiles()
}

func TestCustomMetrics(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()

	traceStatus = healthcheck.OK
	traceability.SetDataDirPath(".")
	runTestHealthcheck()

	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + "/usage"
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

	base := models.CustomMetricDetail{
		APIDetails: apiDetails1,
		AppDetails: appDetails1,
		Count:      5,
		UnitDetails: models.Unit{
			Name: "unit-name",
		},
	}
	_ = base

	testCases := map[string]struct {
		skip            bool
		metricEvent1    models.CustomMetricDetail
		metricEvent2    models.CustomMetricDetail
		expectedMetrics int
	}{
		"no custom metric when api details not in event": {
			skip:         false,
			metricEvent1: models.CustomMetricDetail{},
		},
		"no custom metric when app details not in event": {
			skip: false,
			metricEvent1: models.CustomMetricDetail{
				APIDetails: apiDetails1,
			},
		},
		"no custom metric when unit details not in event": {
			skip: false,
			metricEvent1: models.CustomMetricDetail{
				APIDetails: apiDetails1,
				AppDetails: appDetails1,
			},
		},
		"expect custom metric when all needed data given": {
			skip:            false,
			metricEvent1:    base,
			expectedMetrics: 1,
		},
		"expect 1 metric when multiple updates for same unit and detials": {
			skip:            false,
			metricEvent1:    base,
			metricEvent2:    base,
			expectedMetrics: 1,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				return
			}
			metricCollector.metricMap = map[string]map[string]map[string]map[string]*centralMetric{}
			metricCollector.AddCustomMetricDetail(tc.metricEvent1)
			if tc.metricEvent2.Count > 0 {
				metricCollector.AddCustomMetricDetail(tc.metricEvent2)
			}
			assert.Equal(t, tc.expectedMetrics, len(metricCollector.metricMap))
		})
	}
}
