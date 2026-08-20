package metric

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Axway/agent-sdk/pkg/agent"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	testEnvID         = "267bd671-e5e2-4679-bcc3-bbe7b70f30fd"
	testInstID        = "inst-1"
	testInstName      = "instance-1"
	testManagedApp1   = "managed-app-1"
	testManagedApp2   = "managed-app-2"
	testConsumerOrg   = "test-consumer-org"
	testAccessReq1    = "access-req-1"
	testAccessReq2    = "access-req-2"
	testSubscription1 = "subscription-1"
	testSubscription2 = "subscription-2"
	testCronSchedule  = "* * * * *"
	testLighthouse    = "/lighthouse"
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

var accessToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0IiwiaWF0IjoxNjE0NjA0NzE0LCJleHAiOjE2NDYxNDA3MTQsImF1ZCI6InRlc3RhdWQiLCJzdWIiOiIxMjM0NTYiLCJvcmdfZ3VpZCI6IjEyMzQtMTIzNC0xMjM0LTEyMzQifQ.5Uqt0oFhMgmI-sLQKPGkHwknqzlTxv-qs9I_LmZ18LQ" // NOSONAR - expired synthetic JWT used only in mock HTTP handler, not a real credential

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
		switch {
		case strings.Contains(req.RequestURI, "/auth"):
			s.handleAuth(resp)
		case strings.Contains(req.RequestURI, testLighthouse):
			s.handleLighthouse(resp, req)
		}
		resp.WriteHeader(202)
	}))
}

func (s *testHTTPServer) handleAuth(resp http.ResponseWriter) {
	token := "{\"access_token\":\"" + accessToken + "\",\"expires_in\": 12235677}"
	resp.Write([]byte(token))
}

func (s *testHTTPServer) handleLighthouse(resp http.ResponseWriter, req *http.Request) {
	if s.failUsageEvent {
		s.writeFailureResponse(resp)
		return
	}
	s.lighthouseEventCount++
	req.ParseMultipartForm(1 << 20)
	for _, fileHeaders := range req.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			s.processFileHeader(fileHeader)
		}
	}
}

func (s *testHTTPServer) writeFailureResponse(resp http.ResponseWriter) {
	if s.failUsageResponse != nil {
		b, _ := json.Marshal(*s.failUsageResponse)
		resp.WriteHeader(s.failUsageResponse.StatusCode)
		resp.Write(b)
		return
	}
	resp.WriteHeader(http.StatusBadRequest)
}

func (s *testHTTPServer) processFileHeader(fileHeader *multipart.FileHeader) {
	file, err := fileHeader.Open()
	if err != nil {
		return
	}
	body, _ := io.ReadAll(file)
	var usageEvent UsageEvent
	json.Unmarshal(body, &usageEvent)
	s.givenGranularity = usageEvent.Granularity
	s.eventTimestamp = usageEvent.Timestamp
	for _, report := range usageEvent.Report {
		s.tallyReport(report)
	}
}

func (s *testHTTPServer) tallyReport(report UsageReport) {
	for usageType, usage := range report.Usage {
		if strings.Index(usageType, "Transactions") > 0 {
			s.transactionCount += int(usage)
		} else if strings.Index(usageType, "Volume") > 0 {
			s.transactionVolume += int(usage)
		}
	}
	s.reportCount++
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
					{
						Group: catalog.SubscriptionGVK().Group,
						Kind:  catalog.SubscriptionGVK().Kind,
						ID:    subscriptionName,
						Name:  "catalog/" + subscriptionName,
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

type metricCollectorTestCase struct {
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
}

func metricCollectorTestCases() []metricCollectorTestCase {
	return []metricCollectorTestCase{
		{
			name:                      "WithUsageNoUsageReport",
			loopCount:                 1,
			apiTransactionCount:       []int{0},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{0},
			expectedTransactionCount:  []int{0},
			expectedTransactionVolume: []int{0},
			skipWaitForPub:            true,
		},
		{
			name:                      "WithUsageNoApp",
			loopCount:                 1,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1,
		},
		{
			name:                      "WithUsageWithPriorPublish",
			loopCount:                 1,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1,
			publishPrior:              true,
		},
		{
			name:                      "WithUsageProviderApp",
			loopCount:                 1,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1,
			appName:                   testManagedApp1,
		},
		{
			name:                      "WithUsageConsumerApp",
			loopCount:                 1,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			expectedTransactionVolume: []int{0},
			expectedMetricEventsAcked: 1,
			appName:                   testManagedApp2,
		},
		{
			name:                   "WithUsageWithFailure",
			loopCount:              3,
			apiTransactionCount:    []int{5, 10, 12},
			failUsageEventOnServer: []bool{false, true, false, false},
			failUsageResponseOnServer: []*UsageResponse{
				nil,
				{Description: "Regular failure", StatusCode: 400, Success: false},
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
		{
			name:                   "WithUsageWithFailureWithSpecificDescription",
			loopCount:              3,
			apiTransactionCount:    []int{1, 1, 1},
			failUsageEventOnServer: []bool{true, true, false},
			failUsageResponseOnServer: []*UsageResponse{
				{Description: "The file exceeds the maximum upload size of 454545", StatusCode: 400, Success: false},
				{Description: "Environment ID not found", StatusCode: 404, Success: false},
				nil,
			},
			expectedLHEvents:          []int{0, 0, 1},
			expectedTransactionCount:  []int{0, 0, 1},
			trackVolume:               true,
			expectedTransactionVolume: []int{0, 0, 10},
			expectedMetricEventsAcked: 1,
			appName:                   "unknown",
		},
		{
			name:                      "WithUsageAndFailedMetric",
			loopCount:                 1,
			retryBatchCount:           4,
			apiTransactionCount:       []int{5},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{1},
			expectedTransactionCount:  []int{5},
			expectedTransactionVolume: []int{0},
		},
		{
			name:                      "WithUsageTraceabilityNotConnected",
			loopCount:                 1,
			apiTransactionCount:       []int{0},
			failUsageEventOnServer:    []bool{false},
			failUsageResponseOnServer: []*UsageResponse{nil},
			expectedLHEvents:          []int{0},
			expectedTransactionCount:  []int{0},
			expectedTransactionVolume: []int{0},
			appName:                   testManagedApp1,
			hcStatus:                  healthcheck.FAIL,
			skipWaitForPub:            true,
		},
	}
}

func setupMetricCollectorTest(t *testing.T, s *testHTTPServer) (*collector, *config.CentralConfiguration) {
	t.Helper()
	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + testLighthouse
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
	cfg.SetEnvironmentID(testEnvID)
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	cm := agent.GetCacheManager()
	cm.AddAPIServiceInstance(createAPIServiceInstance(testInstID, testInstName, "111"))
	cm.AddManagedApplication(createManagedApplication("app-1", testManagedApp1, ""))
	cm.AddManagedApplication(createManagedApplication("app-2", testManagedApp2, testConsumerOrg))
	cm.AddAccessRequest(createAccessRequest("ac-1", testAccessReq1, testManagedApp1, testInstID, testInstName, testSubscription1))
	cm.AddAccessRequest(createAccessRequest("ac-2", testAccessReq2, testManagedApp2, testInstID, testInstName, testSubscription2))

	return createMetricCollector().(*collector), cfg
}

func runMetricCollectorLoop(s *testHTTPServer, metricCollector *collector, test metricCollectorTestCase, l int) {
	for i := 0; i < test.apiTransactionCount[l]; i++ {
		metricCollector.AddMetricDetail(Detail{
			APIDetails: apiDetails1,
			StatusCode: "200",
			Duration:   10,
			Bytes:      10,
			AppDetails: models.AppDetails{ID: "111", Name: test.appName},
		})
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

func TestMetricCollector(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	metricCollector, cfg := setupMetricCollectorTest(t, s)

	for _, test := range metricCollectorTestCases() {
		t.Run(test.name, func(t *testing.T) {
			if test.hcStatus != "" {
				traceStatus = test.hcStatus
			}
			runTestHealthcheck()
			metricCollector.registry = newRegistry()
			cfg.SetAxwayManaged(test.trackVolume)
			testClient := setupMockClient(test.retryBatchCount)

			for l := 0; l < test.loopCount; l++ {
				runMetricCollectorLoop(s, metricCollector, test, l)
			}

			assert.Equal(t, test.expectedMetricEventsAcked, testClient.(*MockClient).eventsAcked)
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
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + testLighthouse
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
	cfg.SetEnvironmentID(testEnvID)
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)
	myCollector := createMetricCollector()
	metricCollector := myCollector.(*collector)
	traceStatus = healthcheck.OK
	runTestHealthcheck()

	apiDetails := []models.APIDetails{
		{ID: "000", Name: "000", Revision: 1, TeamID: teamID},
		{ID: "111", Name: "111", Revision: 1, TeamID: teamID},
	}
	appDetails := []models.AppDetails{
		{ID: "000", Name: "app0"},
		{ID: "111", Name: "app1"},
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
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + testLighthouse
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
	cfg.SetEnvironmentID(testEnvID)
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	// setup the cache for handling custom metrics
	cm := agent.GetCacheManager()
	cm.AddAPIServiceInstance(createAPIServiceInstance(testInstID, testInstName, "111"))

	cm.AddManagedApplication(createManagedApplication("app-1", testManagedApp1, ""))
	cm.AddManagedApplication(createManagedApplication("app-2", testManagedApp2, testConsumerOrg))

	cm.AddAccessRequest(createAccessRequest("ac-1", testAccessReq1, testManagedApp1, testInstID, testInstName, testSubscription1))
	cm.AddAccessRequest(createAccessRequest("ac-2", testAccessReq2, testManagedApp2, testInstID, testInstName, testSubscription2))

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
			metricCollector.usagePublisher.schedule = testCronSchedule
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
			assert.Equal(t, metricCollector.usageStartTime, now().Truncate(time.Minute))
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
			cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + testLighthouse
			cfg.SetEnvironmentID(testEnvID)
			cfg.SetAxwayManaged(test.trackVolume)
			cmd.BuildDataPlaneType = "Azure"
			agent.Initialize(cfg)

			traceability.SetDataDirPath(".")
			myCollector := createMetricCollector()
			metricCollector := myCollector.(*collector)
			metricCollector.usagePublisher.schedule = testCronSchedule
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
			metricCollector.usagePublisher.schedule = testCronSchedule
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
			metricCollector.usagePublisher.schedule = testCronSchedule
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

func TestMetricCollectorPublishesAllSubscriptionsAndCleansRegistry(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	metricCollector, cfg := setupMetricCollectorTest(t, s)
	traceStatus = healthcheck.OK
	runTestHealthcheck()

	const (
		testManagedApp3   = "managed-app-3"
		testAccessReq3    = "access-req-3"
		testSubscription3 = "subscription-3"
	)
	cm := agent.GetCacheManager()
	cm.AddManagedApplication(createManagedApplication("app-3", testManagedApp3, testConsumerOrg))
	cm.AddAccessRequest(createAccessRequest("ac-3", testAccessReq3, testManagedApp3, testInstID, testInstName, testSubscription3))

	metricCollector.registry = newRegistry()
	cfg.SetAxwayManaged(false)
	testClient := setupMockClient(0)

	subscriptions := []struct {
		appID        string
		appName      string
		subscription string
	}{
		{"app-1", testManagedApp1, testSubscription1},
		{"app-2", testManagedApp2, testSubscription2},
		{"app-3", testManagedApp3, testSubscription3},
	}

	for _, sub := range subscriptions {
		metricCollector.AddMetricDetail(Detail{
			APIDetails: apiDetails1,
			StatusCode: "200",
			Duration:   10,
			Bytes:      10,
			AppDetails: models.AppDetails{ID: sub.appID, Name: sub.appName},
		})
	}

	// each subscription's transactions should be tracked under its own registry group
	registeredMetricGroups := 0
	metricCollector.registry.Each(func(name string, _ interface{}) {
		if strings.HasPrefix(name, metricKeyPrefix+".") {
			registeredMetricGroups++
		}
	})
	assert.Equal(t, len(subscriptions), registeredMetricGroups)

	assert.NoError(t, metricCollector.Execute())
	metricCollector.usagePublisher.Execute()

	mock := testClient.(*MockClient)
	assert.Equal(t, len(subscriptions), mock.eventsAcked)
	assert.Len(t, mock.capturedEvents, len(subscriptions))

	publishedSubscriptions := map[string]bool{}
	for _, event := range mock.capturedEvents {
		metric := getMetricFromEvent(event)
		if assert.NotNil(t, metric) && assert.NotNil(t, metric.Subscription) {
			publishedSubscriptions[metric.Subscription.ID] = true
		}
	}
	for _, sub := range subscriptions {
		assert.True(t, publishedSubscriptions[sub.subscription], "expected subscription %s to be published", sub.subscription)
	}

	// once every group's event has been acked, the registry should be free of metric groups
	remainingMetricGroups := 0
	metricCollector.registry.Each(func(name string, _ interface{}) {
		if strings.HasPrefix(name, metricKeyPrefix+".") {
			remainingMetricGroups++
		}
	})
	assert.Equal(t, 0, remainingMetricGroups)

	s.resetConfig()
}

func metricStorageKeys(c *collector) []string {
	cs := c.storage.(*cacheStorage)
	var keys []string
	for _, k := range cs.storage.GetKeys() {
		if strings.HasPrefix(k, metricKeyPrefix+".") {
			keys = append(keys, k)
		}
	}
	return keys
}

func metricRegistryGroups(c *collector) []string {
	var names []string
	c.registry.Each(func(name string, _ interface{}) {
		if strings.HasPrefix(name, metricKeyPrefix+".") {
			names = append(names, name)
		}
	})
	return names
}

// TestMetricCollectorPublishesAllStatusesAndUnitsAndCleansStorage loads several HTTP status metrics and
// custom unit metrics, publishes them, and verifies every status/unit is present on the published events
// and that both the registry and the persisted cache storage are fully cleaned once the events are acked.
func TestMetricCollectorPublishesAllStatusesAndUnitsAndCleansStorage(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	metricCollector, _ := setupMetricCollectorTest(t, s)
	traceStatus = healthcheck.OK
	runTestHealthcheck()

	// start from a clean registry and an empty in-memory storage so the assertions below observe only
	// the metrics recorded by this test
	metricCollector.registry = newRegistry()
	freshStorage := newStorageCache(metricCollector).(*cacheStorage)
	freshStorage.isInitialized = true
	metricCollector.storage = freshStorage

	testClient := setupMockClient(0)

	// several HTTP statuses for one app/api - each distinct status text (Success/Failure/Exception) is
	// tracked as its own metric entry within the app/api's registry group
	for _, status := range []string{"200", "400", "500"} {
		metricCollector.AddMetricDetail(Detail{
			APIDetails: apiDetails1,
			StatusCode: status,
			Duration:   10,
			Bytes:      10,
			AppDetails: models.AppDetails{ID: "app-1", Name: testManagedApp1},
		})
	}

	// custom units for a second app/api - these land in a separate registry group and publish together
	// on a single custom-unit event
	for _, unit := range []string{"widgets", "gadgets"} {
		metricCollector.AddCustomMetricDetail(models.CustomMetricDetail{
			APIDetails:  models.APIDetails{ID: "111", Name: "111"},
			AppDetails:  models.AppDetails{ID: "app-2", Name: testManagedApp2},
			UnitDetails: models.Unit{Name: unit},
			Count:       3,
		})
	}

	// before publishing: 3 status + 2 unit entries are cached, spread across 2 registry groups
	assert.Len(t, metricStorageKeys(metricCollector), 5)
	assert.Len(t, metricRegistryGroups(metricCollector), 2)

	assert.NoError(t, metricCollector.Execute())
	metricCollector.usagePublisher.Execute()

	mock := testClient.(*MockClient)

	// the batch should carry one event per status plus one custom-unit event
	assert.Equal(t, 4, mock.eventsAcked)
	assert.Len(t, mock.capturedEvents, 4)

	publishedStatuses := map[string]bool{}
	publishedUnits := map[string]bool{}
	for _, event := range mock.capturedEvents {
		if metric := getMetricFromEvent(event); assert.NotNil(t, metric) &&
			metric.Units != nil && metric.Units.Transactions != nil && metric.Units.Transactions.Status != "" {
			publishedStatuses[metric.Units.Transactions.Status] = true
		}
		// custom units are dropped when reconstructing from the event (json:"-"), so read them from the
		// raw published data instead
		if data := getRawEventData(event); data != nil {
			if units, ok := data["units"].(map[string]interface{}); ok {
				for name, v := range units {
					if name != "transactions" && v != nil {
						publishedUnits[name] = true
					}
				}
			}
		}
	}

	assert.Equal(t, map[string]bool{"Success": true, "Failure": true, "Exception": true}, publishedStatuses)
	assert.Equal(t, map[string]bool{"widgets": true, "gadgets": true}, publishedUnits)

	// once every event is acked, the registry and the persisted cache storage should both be free of
	// metric entries
	assert.Empty(t, metricRegistryGroups(metricCollector), "registry should be cleaned of all metric groups")
	assert.Empty(t, metricStorageKeys(metricCollector), "cache storage should be cleaned of all metric entries")

	s.resetConfig()
}

// TestMetricStorageKeysAreIsolatedByGenerationStartTime verifies that metrics for the same
// app/api/status collected in different generations are stored under distinct, start-time-qualified
// keys, so cleaning up a generation whose event was published does not remove a later generation's
// metric that was collected while the first was still in flight.
func TestMetricStorageKeysAreIsolatedByGenerationStartTime(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	metricCollector, _ := setupMetricCollectorTest(t, s)
	traceStatus = healthcheck.OK
	runTestHealthcheck()

	metricCollector.registry = newRegistry()
	freshStorage := newStorageCache(metricCollector).(*cacheStorage)
	freshStorage.isInitialized = true
	metricCollector.storage = freshStorage

	addDetail := func() {
		metricCollector.AddMetricDetail(Detail{
			APIDetails: apiDetails1,
			StatusCode: "200",
			Duration:   10,
			Bytes:      10,
			AppDetails: models.AppDetails{ID: "app-1", Name: testManagedApp1},
		})
	}

	// generation 1
	metricCollector.metricStartTime = time.UnixMilli(60000)
	addDetail()

	// generation 2 for the same app/api/status, as if collected while generation 1 is publishing
	metricCollector.metricStartTime = time.UnixMilli(120000)
	addDetail()

	// each generation is stored under its own start-time-qualified key and its own registry group
	assert.Len(t, metricStorageKeys(metricCollector), 2)
	assert.Len(t, metricRegistryGroups(metricCollector), 2)

	// find generation 1's registry group and clean it up, as happens when its published event is acked
	var gen1Name string
	var gen1Group groupedMetrics
	metricCollector.registry.Each(func(name string, v interface{}) {
		if strings.HasSuffix(name, ".60000") {
			gen1Name = name
			gen1Group, _ = v.(groupedMetrics)
		}
	})
	if !assert.NotEmpty(t, gen1Name) {
		return
	}
	gen1Metric, ok := gen1Group.getMetric("Success")
	if !assert.True(t, ok) {
		return
	}
	metricCollector.cleanupMetricCounters(gen1Name, gen1Group.counters, gen1Group, gen1Metric)

	// generation 1 is gone from both registry and storage; generation 2 is untouched
	if remaining := metricStorageKeys(metricCollector); assert.Len(t, remaining, 1) {
		assert.True(t, strings.HasSuffix(remaining[0], ".120000"),
			"generation 2 metric should remain in storage, got %s", remaining[0])
	}
	if groups := metricRegistryGroups(metricCollector); assert.Len(t, groups, 1) {
		assert.True(t, strings.HasSuffix(groups[0], ".120000"),
			"generation 2 group should remain in registry, got %s", groups[0])
	}

	s.resetConfig()
}

// TestMetricCacheRoundTripRekeysAndCleansUpWithoutOrphaning saves transaction metrics from two different
// generations, reloads them into a fresh collector, and verifies each entry is re-keyed to the current
// run's storage key (the key is recomputed from the reloaded context, not restored from a persisted
// cachedMetric.Key) and is fully removed from both the registry and storage on publish - i.e. no stale,
// orphaned cache entry survives the round trip.
func TestMetricCacheRoundTripRekeysAndCleansUpWithoutOrphaning(t *testing.T) {
	cleanUpCachedMetricFile()
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	collector1, _ := setupMetricCollectorTest(t, s)
	traceStatus = healthcheck.OK
	runTestHealthcheck()

	// generation 1: a success transaction, keyed with start time 60000
	collector1.metricStartTime = time.UnixMilli(60000)
	collector1.AddMetricDetail(Detail{
		APIDetails: apiDetails1,
		StatusCode: "200",
		Duration:   10,
		Bytes:      10,
		AppDetails: models.AppDetails{ID: "app-1", Name: testManagedApp1},
	})

	// generation 2: a failure transaction, keyed with start time 120000 - this later value is what gets
	// persisted as the metric start time and loaded by the next collector
	collector1.metricStartTime = time.UnixMilli(120000)
	collector1.AddMetricDetail(Detail{
		APIDetails: apiDetails1,
		StatusCode: "400",
		Duration:   10,
		Bytes:      10,
		AppDetails: models.AppDetails{ID: "app-1", Name: testManagedApp1},
	})

	assert.Len(t, metricStorageKeys(collector1), 2)
	collector1.storage.save()

	// reload into a fresh collector
	collector2 := createMetricCollector().(*collector)

	// both metrics are restored and re-keyed to this run's start time (120000); the stale generation-1
	// key (60000) must not survive
	reloadedKeys := metricStorageKeys(collector2)
	assert.Len(t, reloadedKeys, 2)
	for _, k := range reloadedKeys {
		assert.Falsef(t, strings.HasSuffix(k, ".60000"), "stale generation-1 key should be re-keyed, got %s", k)
	}

	// publish and confirm the registry and storage are fully cleaned - nothing is orphaned
	testClient := setupMockClient(0)
	assert.NoError(t, collector2.Execute())

	assert.Equal(t, 2, testClient.(*MockClient).eventsAcked)
	assert.Empty(t, metricRegistryGroups(collector2), "registry should be cleaned after publish")
	assert.Empty(t, metricStorageKeys(collector2), "storage should be cleaned after publish (no orphans)")

	s.resetConfig()
}

// TestMetricEventsReportedWithOwnGenerationStartTime verifies that when the registry holds metric groups
// from more than one generation, each published event is stamped with its own generation's start time
// rather than all sharing the current publish cycle's start time.
func TestMetricEventsReportedWithOwnGenerationStartTime(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	metricCollector, _ := setupMetricCollectorTest(t, s)
	traceStatus = healthcheck.OK
	runTestHealthcheck()

	metricCollector.registry = newRegistry()
	freshStorage := newStorageCache(metricCollector).(*cacheStorage)
	freshStorage.isInitialized = true
	metricCollector.storage = freshStorage

	addSuccess := func() {
		metricCollector.AddMetricDetail(Detail{
			APIDetails: apiDetails1,
			StatusCode: "200",
			Duration:   10,
			Bytes:      10,
			AppDetails: models.AppDetails{ID: "app-1", Name: testManagedApp1},
		})
	}

	// two generations for the same app/api/status coexist in the registry (e.g. an earlier generation
	// that was not yet acked), each under its own start time
	metricCollector.metricStartTime = time.UnixMilli(60000)
	addSuccess()
	metricCollector.metricStartTime = time.UnixMilli(120000)
	addSuccess()

	// publish; the current start time (120000) is >= both generations, so both are reported this cycle
	testClient := setupMockClient(0)
	assert.NoError(t, metricCollector.Execute())

	mock := testClient.(*MockClient)
	assert.Equal(t, 2, mock.eventsAcked)

	// each event carries its own generation's start time, not a single shared publish-cycle start time
	starts := map[int64]bool{}
	for _, event := range mock.capturedEvents {
		raw, ok := event.Content.Fields[messageKey].(string)
		if !ok {
			continue
		}
		var v4 map[string]any
		if err := json.Unmarshal([]byte(raw), &v4); err != nil {
			continue
		}
		if ts, ok := v4["timestamp"].(float64); ok {
			starts[int64(ts)] = true
		}
	}
	assert.Equal(t, map[int64]bool{60000: true, 120000: true}, starts)

	s.resetConfig()
}

func setupAPIMetricCollectorTest(t *testing.T, s *testHTTPServer) *collector {
	t.Helper()
	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + testLighthouse
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
	cfg.SetEnvironmentID(testEnvID)
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	traceStatus = healthcheck.OK
	runTestHealthcheck()

	return createMetricCollector().(*collector)
}

func TestAddAPIMetricAggregatesTransactionCounter(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	metricCollector := setupAPIMetricCollectorTest(t, s)
	testClient := setupMockClient(0)

	newAPIMetric := func(count, min, max int64, avg float64) *APIMetric {
		return &APIMetric{
			Subscription: models.Subscription{ID: "sub-1"},
			App:          models.AppDetails{ID: "app-1"},
			API:          models.APIDetails{ID: "api-1", Name: "api-1"},
			StatusCode:   "200",
			Count:        count,
			Response:     ResponseMetrics{Min: min, Max: max, Avg: avg},
			Observation:  models.ObservationDetails{Start: 10, End: 20},
		}
	}

	// two calls for the same subscription/app/api/status should merge into a single cached metric
	metricCollector.AddAPIMetric(newAPIMetric(5, 10, 50, 20))
	metricCollector.AddAPIMetric(newAPIMetric(3, 5, 80, 40))

	assert.NoError(t, metricCollector.Execute())

	mock := testClient.(*MockClient)
	assert.Equal(t, 1, mock.eventsAcked)
	if !assert.Len(t, mock.capturedEvents, 1) {
		return
	}

	metric := getMetricFromEvent(mock.capturedEvents[0])
	if !assert.NotNil(t, metric) {
		return
	}
	assert.Equal(t, "sub-1", metric.Subscription.ID)
	assert.Equal(t, int64(8), metric.Units.Transactions.Count)
	assert.Equal(t, int64(5), metric.Units.Transactions.Response.Min)
	assert.Equal(t, int64(80), metric.Units.Transactions.Response.Max)
	assert.Equal(t, 27.5, metric.Units.Transactions.Response.Avg)

	s.resetConfig()
}

func TestAddAPIMetricAggregatesCustomUnitCounter(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	metricCollector := setupAPIMetricCollectorTest(t, s)
	testClient := setupMockClient(0)

	newCustomAPIMetric := func(count int64) *APIMetric {
		return &APIMetric{
			Subscription: models.Subscription{ID: "sub-2"},
			App:          models.AppDetails{ID: "app-2"},
			API:          models.APIDetails{ID: "api-2", Name: "api-2"},
			Unit:         &models.Unit{Name: "widgets"},
			Count:        count,
			Observation:  models.ObservationDetails{Start: 10, End: 20},
		}
	}

	// two calls for the same subscription/app/api/unit should merge into a single cached metric
	metricCollector.AddAPIMetric(newCustomAPIMetric(4))
	metricCollector.AddAPIMetric(newCustomAPIMetric(6))

	assert.NoError(t, metricCollector.Execute())

	mock := testClient.(*MockClient)
	assert.Equal(t, 1, mock.eventsAcked)
	if !assert.Len(t, mock.capturedEvents, 1) {
		return
	}

	data := getRawEventData(mock.capturedEvents[0])
	if !assert.NotNil(t, data) {
		return
	}
	units, ok := data["units"].(map[string]interface{})
	if !assert.True(t, ok, "expected units in published event data") {
		return
	}
	unitData, ok := units["widgets"].(map[string]interface{})
	if !assert.True(t, ok, "expected widgets unit in published event data") {
		return
	}
	assert.Equal(t, float64(10), unitData["count"])

	s.resetConfig()
}

type offlineMetricTestCase struct {
	name                string
	loopCount           int
	apiTransactionCount []int
}

func offlineMetricTestCases() []offlineMetricTestCase {
	return []offlineMetricTestCase{
		{name: "NoReports", loopCount: 0, apiTransactionCount: []int{}},
		{name: "OneReport", loopCount: 1, apiTransactionCount: []int{10}},
		{name: "ThreeReports", loopCount: 3, apiTransactionCount: []int{5, 10, 2}},
		{name: "ThreeReportsNoUsage", loopCount: 3, apiTransactionCount: []int{0, 0, 0}},
		{name: "SixReports", loopCount: 6, apiTransactionCount: []int{5, 10, 2, 0, 3, 9}},
	}
}

func setupOfflineCollectorCfg(t *testing.T, s *testHTTPServer) *config.CentralConfiguration {
	t.Helper()
	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + testLighthouse
	cfg.EnvironmentID = testEnvID
	cmd.BuildDataPlaneType = "Azure"
	cfg.UsageReporting.(*config.UsageReportingConfiguration).Offline = true
	agent.Initialize(cfg)
	return cfg
}

func validateOfflineEvents(t *testing.T, cfg *config.CentralConfiguration, s *testHTTPServer, report UsageEvent, test offlineMetricTestCase, startDate time.Time) {
	t.Helper()
	for j := 0; j < test.loopCount; j++ {
		reportKey := startDate.Add(time.Duration(j-1) * time.Hour).Format(ISO8601)
		assert.Equal(t, cmd.BuildDataPlaneType, report.Report[reportKey].Product)
		assert.Equal(t, test.apiTransactionCount[j], int(report.Report[reportKey].Usage[cmd.BuildDataPlaneType+".Transactions"]))
	}
	if test.loopCount == 0 {
		return
	}
	assert.Equal(t, int(time.Hour.Milliseconds()), report.Granularity)
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + testLighthouse
	assert.Equal(t, cfg.UsageReporting.GetURL()+schemaPath, report.SchemaID)
	assert.Equal(t, cfg.GetEnvironmentID(), report.EnvID)
}

func runOfflineCollectorLoops(myCollector Collector, test offlineMetricTestCase, startDate time.Time) {
	metricCollector := myCollector.(*collector)
	testLoops := 0
	now = func() time.Time {
		next := startDate.Add(time.Hour * time.Duration(testLoops))
		return next
	}
	for testLoops < test.loopCount {
		for i := 0; i < test.apiTransactionCount[testLoops]; i++ {
			myCollector.AddMetric(apiDetails1, "200", 10, 10, "")
		}
		metricCollector.Execute()
		testLoops++
	}
}

func validateOfflineReportFile(t *testing.T, cfg *config.CentralConfiguration, s *testHTTPServer, myCollector Collector, test offlineMetricTestCase, startDate time.Time, testNum int) {
	t.Helper()
	metricCollector := myCollector.(*collector)
	reportGenerator := metricCollector.reports
	publisher := metricCollector.usagePublisher

	events := metricCollector.reports.loadEvents()
	validateOfflineEvents(t, cfg, s, events, test, startDate)

	publisher.Execute()

	if test.loopCount == 0 {
		expectedFile := reportGenerator.generateReportPath(ISO8601Time(startDate), 0)
		assert.NoFileExists(t, expectedFile)
		return
	}

	expectedFile := reportGenerator.generateReportPath(ISO8601Time(startDate), testNum-1)
	assert.FileExists(t, expectedFile)

	data, err := os.ReadFile(expectedFile)
	assert.Nil(t, err)

	var reportEvents UsageEvent
	err = json.Unmarshal(data, &reportEvents)
	assert.Nil(t, err)
	assert.NotNil(t, reportEvents)

	validateOfflineEvents(t, cfg, s, reportEvents, test, startDate)
	s.resetOffline(myCollector)
}

func TestOfflineMetricCollector(t *testing.T) {
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")
	traceStatus = healthcheck.OK
	runTestHealthcheck()

	cfg := setupOfflineCollectorCfg(t, s)

	for testNum, test := range offlineMetricTestCases() {
		t.Run(test.name, func(t *testing.T) {
			setupMockClient(0)
			startDate := time.Date(2021, 1, 31, 12, 30, 0, 0, time.Local)
			myCollector := createMetricCollector()

			runOfflineCollectorLoops(myCollector, test, startDate)
			validateOfflineReportFile(t, cfg, s, myCollector, test, startDate, testNum)
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
	cfg.SetEnvironmentID(testEnvID)
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	cm := agent.GetCacheManager()
	cm.AddAPIServiceInstance(createAPIServiceInstance(testInstID, testInstName, "111"))

	cm.AddManagedApplication(createManagedApplication("app-1", testManagedApp1, ""))
	cm.AddManagedApplication(createManagedApplication("app-2", testManagedApp2, testConsumerOrg))

	cm.AddAccessRequest(createAccessRequest("ac-1", testAccessReq1, testManagedApp1, testInstID, testInstName, testSubscription1))
	cm.AddAccessRequest(createAccessRequest("ac-2", testAccessReq2, testManagedApp2, testInstID, testInstName, testSubscription2))

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
			metricCollector.registry = newRegistry()
			metricCollector.AddCustomMetricDetail(tc.metricEvent1)
			if tc.metricEvent2.Count > 0 {
				metricCollector.AddCustomMetricDetail(tc.metricEvent2)
			}
			metricsCount := 0
			metricCollector.registry.Each(func(_ string, v interface{}) {
				if gm, ok := v.(groupedMetrics); ok {
					metricsCount += len(gm.metrics)
				}
			})
			assert.Equal(t, tc.expectedMetrics, metricsCount)
		})
	}
}

func TestCollectorCreateOrUpdateHistogramIDResolution(t *testing.T) {
	// Mock setup would go here - this is a conceptual test
	tests := []struct {
		name          string
		apiID         string
		apiName       string
		expectedAPIID string
		description   string
	}{
		{
			name:          "API ID with content after prefix",
			apiID:         "remoteApiId_dwight",
			apiName:       "schrute",
			expectedAPIID: "remoteApiId_dwight",
			description:   "Should preserve original API ID when it has content after prefix",
		},
		{
			name:          "API ID is just prefix, use name",
			apiID:         "remoteApiId_",
			apiName:       "schrute",
			expectedAPIID: "remoteApiName_schrute",
			description:   "Should use API name with name prefix when ID is just the prefix",
		},
		{
			name:          "Empty API ID and name",
			apiID:         "",
			apiName:       "",
			expectedAPIID: "remoteApiId_unknown",
			description:   "Should use unknown with prefix when both are empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would require proper mock setup for the collector
			// The test would verify that detail.APIDetails.ID gets resolved correctly
			// before being used in the metric generation
		})
	}
}

func TestCreateAPIDetail(t *testing.T) {
	cases := map[string]struct {
		apiServiceRI *apiv1.ResourceInstance
		apiID        string
		apiName      string
		wantOwnType  string
		wantOwnGUID  string
		wantSvcID    string
	}{
		"real ID prefix resolves owner and apiServiceId from cache": {
			apiServiceRI: withMetaID(makeAPIServiceRI("api-1", &apiv1.Owner{Type: apiv1.TeamOwner, ID: testTeamGUID1}), "svc-meta-api-1"),
			apiID:        transutil.SummaryEventProxyIDPrefix + "api-1",
			apiName:      "api-one",
			wantOwnType:  "team",
			wantOwnGUID:  testTeamGUID1,
			wantSvcID:    "svc-meta-api-1",
		},
		"name-fallback prefix resolves owner and apiServiceId from cache": {
			apiServiceRI: withMetaID(makeAPIServiceRI("api-2", &apiv1.Owner{Type: apiv1.TeamOwner, ID: testTeamGUID1}), "svc-meta-api-2"),
			apiID:        transutil.SummaryEventAPINamePrefix + "api-2",
			apiName:      "api-two",
			wantOwnType:  "team",
			wantOwnGUID:  testTeamGUID1,
			wantSvcID:    "svc-meta-api-2",
		},
		"api not found in cache returns unknown owner and apiServiceId": {
			apiServiceRI: nil,
			apiID:        transutil.SummaryEventProxyIDPrefix + "missing-api",
			apiName:      "missing",
			wantOwnType:  "unknown",
			wantSvcID:    unknown,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			agent.InitializeForTest(nil, agent.TestWithCentralConfig(newUtilCacheConfig()))
			if tc.apiServiceRI != nil {
				agent.GetCacheManager().AddAPIService(tc.apiServiceRI)
			}

			c := &collector{logger: log.NewFieldLogger()}
			ref := c.createAPIDetail(models.APIDetails{ID: tc.apiID, Name: tc.apiName})
			if ref == nil {
				t.Fatal("expected non-nil API resource reference")
			}
			assert.Equal(t, tc.apiID, ref.ID)
			assert.Equal(t, tc.apiName, ref.Name)
			assert.Equal(t, tc.wantSvcID, ref.APIServiceID)
			if ref.Owner == nil {
				t.Fatal("expected non-nil owner")
			}
			assert.Equal(t, tc.wantOwnType, ref.Owner.Type)
			if tc.wantOwnGUID != "" {
				assert.Equal(t, tc.wantOwnGUID, ref.Owner.TeamGUID)
			}
		})
	}
}

func TestGetAccessRequestAndManagedApp(t *testing.T) {
	const (
		remoteAppMetaID = "app-remote"
		plainAppMetaID  = "app-plain"
		otherAppMetaID  = "app-other"
	)
	cases := map[string]struct {
		managedAppRI *apiv1.ResourceInstance
		appID        string
		appName      string
		wantAppMeta  string
	}{
		"remoteAppId_ prefix stripped before lookup": {
			managedAppRI: createManagedApplication(remoteAppMetaID, testManagedApp1, ""),
			appID:        transutil.SummaryEventApplicationIDPrefix + remoteAppMetaID,
			wantAppMeta:  remoteAppMetaID,
		},
		"unprefixed appID still resolves": {
			managedAppRI: createManagedApplication(plainAppMetaID, testManagedApp1, ""),
			appID:        plainAppMetaID,
			wantAppMeta:  plainAppMetaID,
		},
		"app not found in cache returns nil": {
			managedAppRI: createManagedApplication(otherAppMetaID, testManagedApp1, ""),
			appID:        transutil.SummaryEventApplicationIDPrefix + "missing-app",
			appName:      "no-such-app",
			wantAppMeta:  "",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			agent.InitializeForTest(nil, agent.TestWithCentralConfig(newUtilCacheConfig()))
			agent.GetCacheManager().AddManagedApplication(tc.managedAppRI)

			c := &collector{logger: log.NewFieldLogger()}
			_, managedApp := c.getAccessRequestAndManagedApp(agent.GetCacheManager(), transactionContext{
				AppDetails: models.AppDetails{ID: tc.appID, Name: tc.appName},
			})

			if tc.wantAppMeta == "" {
				assert.Nil(t, managedApp)
				return
			}
			require.NotNil(t, managedApp)
			assert.Equal(t, tc.wantAppMeta, managedApp.Metadata.ID)
		})
	}
}

func quotaReference(unit, name string) map[string]interface{} {
	return map[string]interface{}{
		"kind": catalog.QuotaGVK().Kind,
		"unit": unit,
		"name": name,
	}
}

func TestGetQuota(t *testing.T) {
	const (
		quotaRefPrefix    = "org/catalog/"
		quotaUnitName     = "widgets"
		quotaRefName      = "quota-ref-name"
		quotaGUID         = "quota-guid-1"
		otherQuotaRefName = "quota-ref-name-2"
		otherQuotaGUID    = "quota-guid-2"
	)
	metaRefFor := func(name, id string) apiv1.Reference {
		return apiv1.Reference{Group: catalog.QuotaGVK().Group, Kind: catalog.QuotaGVK().Kind, Name: name, ID: id}
	}

	cases := map[string]struct {
		accessRequest *management.AccessRequest
		unitName      string
		wantID        string
	}{
		"nil access request returns unknown": {
			accessRequest: nil,
			unitName:      defaultUnit,
			wantID:        unknown,
		},
		"no quota reference for unit returns unknown": {
			accessRequest: &management.AccessRequest{
				References: []interface{}{quotaReference(quotaUnitName, quotaRefPrefix+quotaRefName)},
			},
			unitName: defaultUnit,
			wantID:   unknown,
		},
		"quota name resolved but no matching metadata reference returns unknown": {
			accessRequest: &management.AccessRequest{
				References: []interface{}{quotaReference(defaultUnit, quotaRefPrefix+quotaRefName)},
			},
			unitName: defaultUnit,
			wantID:   unknown,
		},
		"quota resolves to id for default unit": {
			accessRequest: &management.AccessRequest{
				References: []interface{}{quotaReference(defaultUnit, quotaRefPrefix+quotaRefName)},
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{References: []apiv1.Reference{metaRefFor(quotaRefName, quotaGUID)}},
				},
			},
			unitName: defaultUnit,
			wantID:   quotaGUID,
		},
		"empty unit name falls back to default unit": {
			accessRequest: &management.AccessRequest{
				References: []interface{}{quotaReference(defaultUnit, quotaRefPrefix+quotaRefName)},
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{References: []apiv1.Reference{metaRefFor(quotaRefName, quotaGUID)}},
				},
			},
			unitName: "",
			wantID:   quotaGUID,
		},
		"quota resolves to id for a named custom unit, ignoring other units' quotas": {
			accessRequest: &management.AccessRequest{
				References: []interface{}{
					quotaReference(defaultUnit, quotaRefPrefix+quotaRefName),
					quotaReference(quotaUnitName, quotaRefPrefix+otherQuotaRefName),
				},
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{References: []apiv1.Reference{
						metaRefFor(quotaRefName, quotaGUID),
						metaRefFor(otherQuotaRefName, otherQuotaGUID),
					}},
				},
			},
			unitName: quotaUnitName,
			wantID:   otherQuotaGUID,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &collector{logger: log.NewFieldLogger()}
			ref := c.getQuota(tc.accessRequest, tc.unitName)
			require.NotNil(t, ref)
			assert.Equal(t, tc.wantID, ref.ID)
		})
	}
}

// metaRef builds a Metadata.References entry for gvk, the structured relationship reference
// read by GetReferenceByGVK (distinct from the raw references[] sub-resource getQuota reads).
func metaRef(gvk apiv1.GroupVersionKind, id string) apiv1.Reference {
	return apiv1.Reference{Group: gvk.Group, Kind: gvk.Kind, ID: id}
}

func TestGetAPIServiceRevision(t *testing.T) {
	const revisionGUID = "instance-guid-1"

	cases := map[string]struct {
		accessRequest *management.AccessRequest
		wantID        string
	}{
		"nil access request returns unknown": {
			accessRequest: nil,
			wantID:        unknown,
		},
		"no APIServiceInstance reference returns unknown": {
			accessRequest: &management.AccessRequest{},
			wantID:        unknown,
		},
		"APIServiceInstance reference resolves to its id": {
			accessRequest: &management.AccessRequest{
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{References: []apiv1.Reference{
						metaRef(management.APIServiceInstanceGVK(), revisionGUID),
					}},
				},
			},
			wantID: revisionGUID,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &collector{logger: log.NewFieldLogger()}
			ref := c.getAPIServiceRevision(tc.accessRequest)
			require.NotNil(t, ref)
			assert.Equal(t, tc.wantID, ref.ID)
		})
	}
}

func TestGetAssetResource(t *testing.T) {
	const assetGUID = "asset-guid-1"

	cases := map[string]struct {
		accessRequest *management.AccessRequest
		wantID        string
	}{
		"nil access request returns unknown": {
			accessRequest: nil,
			wantID:        unknown,
		},
		"no AssetResource reference returns unknown": {
			accessRequest: &management.AccessRequest{},
			wantID:        unknown,
		},
		"AssetResource reference resolves to its id": {
			accessRequest: &management.AccessRequest{
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{References: []apiv1.Reference{
						metaRef(catalog.AssetResourceGVK(), assetGUID),
					}},
				},
			},
			wantID: assetGUID,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &collector{logger: log.NewFieldLogger()}
			ref := c.getAssetResource(tc.accessRequest)
			require.NotNil(t, ref)
			assert.Equal(t, tc.wantID, ref.ID)
		})
	}
}

func TestGetProductPlan(t *testing.T) {
	const planGUID = "plan-guid-1"

	cases := map[string]struct {
		accessRequest *management.AccessRequest
		wantID        string
	}{
		"nil access request returns unknown": {
			accessRequest: nil,
			wantID:        unknown,
		},
		"no ProductPlan reference returns unknown": {
			accessRequest: &management.AccessRequest{},
			wantID:        unknown,
		},
		"ProductPlan reference resolves to its id": {
			accessRequest: &management.AccessRequest{
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{References: []apiv1.Reference{
						metaRef(catalog.ProductPlanGVK(), planGUID),
					}},
				},
			},
			wantID: planGUID,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &collector{logger: log.NewFieldLogger()}
			ref := c.getProductPlan(tc.accessRequest)
			require.NotNil(t, ref)
			assert.Equal(t, tc.wantID, ref.ID)
		})
	}
}

func TestGetProduct(t *testing.T) {
	const (
		productGUID    = "product-guid-1"
		releaseGUID    = "release-guid-1"
		productOwnerID = "product-team-guid-1"
	)
	embeddedOwner := func(owner *apiv1.Owner) map[string]apiv1.EmbeddedReferences {
		return map[string]apiv1.EmbeddedReferences{
			"metadata.references": {References: []apiv1.EmbeddedReference{
				{Group: catalog.PublishedProductGVK().Group, Kind: catalog.PublishedProductGVK().Kind, Owner: owner},
			}},
		}
	}

	cases := map[string]struct {
		accessRequest *management.AccessRequest
		wantID        string
		wantVersionID string
		wantOwner     *models.Owner
	}{
		"nil access request returns unknown id and versionId, no owner": {
			accessRequest: nil,
			wantID:        unknown,
			wantVersionID: unknown,
			wantOwner:     nil,
		},
		"no product or release reference returns unknown id and versionId, no owner": {
			accessRequest: &management.AccessRequest{},
			wantID:        unknown,
			wantVersionID: unknown,
			wantOwner:     nil,
		},
		"product reference resolves id and owner, versionId stays unknown without a release reference": {
			accessRequest: &management.AccessRequest{
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{
						References: []apiv1.Reference{metaRef(catalog.ProductGVK(), productGUID)},
					},
					Embedded: embeddedOwner(&apiv1.Owner{Type: apiv1.TeamOwner, ID: productOwnerID}),
				},
			},
			wantID:        productGUID,
			wantVersionID: unknown,
			wantOwner:     &models.Owner{Type: "team", TeamGUID: productOwnerID},
		},
		"product and release references both resolve": {
			accessRequest: &management.AccessRequest{
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{
						References: []apiv1.Reference{
							metaRef(catalog.ProductGVK(), productGUID),
							metaRef(catalog.ProductReleaseGVK(), releaseGUID),
						},
					},
					Embedded: embeddedOwner(&apiv1.Owner{Type: apiv1.TeamOwner, ID: productOwnerID}),
				},
			},
			wantID:        productGUID,
			wantVersionID: releaseGUID,
			wantOwner:     &models.Owner{Type: "team", TeamGUID: productOwnerID},
		},
		"product resolved without an embedded owner reference returns owner type none": {
			accessRequest: &management.AccessRequest{
				ResourceMeta: apiv1.ResourceMeta{
					Metadata: apiv1.Metadata{
						References: []apiv1.Reference{metaRef(catalog.ProductGVK(), productGUID)},
					},
				},
			},
			wantID:        productGUID,
			wantVersionID: unknown,
			wantOwner:     &models.Owner{Type: "none"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := &collector{logger: log.NewFieldLogger()}
			ref := c.getProduct(tc.accessRequest)
			require.NotNil(t, ref)
			assert.Equal(t, tc.wantID, ref.ID)
			assert.Equal(t, tc.wantVersionID, ref.VersionID)
			assert.Equal(t, tc.wantOwner, ref.Owner)
		})
	}
}

// noopStorage satisfies the storageCache interface with no-ops for all methods except
// removeMetric, which records calls for assertion in cleanup tests.
type noopStorage struct {
	removed []*centralMetric
}

func (s *noopStorage) initialize()                                            { /* no-op */ }
func (s *noopStorage) updateUsage(_ int)                                      { /* no-op */ }
func (s *noopStorage) updateVolume(_ int64)                                   { /* no-op */ }
func (s *noopStorage) updateAppUsage(_ int, _ string)                         { /* no-op */ }
func (s *noopStorage) updateMetric(_ cachedMetricInterface, _ *centralMetric) { /* no-op */ }
func (s *noopStorage) save()                                                  { /* no-op */ }
func (s *noopStorage) removeMetric(m *centralMetric)                          { s.removed = append(s.removed, m) }

func newCleanupCollector() (*collector, *noopStorage) {
	st := &noopStorage{}
	c := &collector{
		storage: st,
		logger:  log.NewFieldLogger(),
	}
	return c, st
}

func newStatusMetric(status string) *centralMetric {
	return &centralMetric{
		Subscription: &models.ResourceReference{ID: "sub1"},
		App:          &models.ApplicationResourceReference{ResourceReference: models.ResourceReference{ID: "app1"}},
		API:          &models.APIResourceReference{ResourceReference: models.ResourceReference{ID: "api1"}, Name: "api1"},
		Units:        &Units{Transactions: &Transactions{Status: status}},
	}
}

func TestCleanupMetricCounters(t *testing.T) {
	const registryKey = "metric.sub1.app1.api1.123"

	metric1 := newStatusMetric("Success")
	unitMetric := newStatusMetric("unit-name")

	tests := map[string]struct {
		metrics          map[string]*centralMetric
		apiCounters      map[string]*apiCounter
		counters         map[string]*counter
		wantRemoved      []*centralMetric
		wantDeregistered bool
	}{
		"removes the acked metric and deregisters a now-empty group": {
			metrics:          map[string]*centralMetric{"Success": metric1},
			apiCounters:      map[string]*apiCounter{"Success": newAPICounter()},
			counters:         map[string]*counter{},
			wantRemoved:      []*centralMetric{metric1},
			wantDeregistered: true,
		},
		"removes the acked metric and any custom unit metrics bundled with it": {
			metrics:          map[string]*centralMetric{"Success": metric1, "unit-name": unitMetric},
			apiCounters:      map[string]*apiCounter{"Success": newAPICounter()},
			counters:         map[string]*counter{"unit-name": newCounter()},
			wantRemoved:      []*centralMetric{metric1, unitMetric},
			wantDeregistered: true,
		},
		"leaves the group registered while a sibling status is still unacked": {
			metrics:          map[string]*centralMetric{"Success": metric1, "Failure": newStatusMetric("Failure")},
			apiCounters:      map[string]*apiCounter{"Success": newAPICounter(), "Failure": newAPICounter()},
			counters:         map[string]*counter{},
			wantRemoved:      []*centralMetric{metric1},
			wantDeregistered: false,
		},
		"counter key missing from the group's metrics does not panic": {
			metrics:          map[string]*centralMetric{"Success": metric1},
			apiCounters:      map[string]*apiCounter{"Success": newAPICounter()},
			counters:         map[string]*counter{"ghost": newCounter()},
			wantRemoved:      []*centralMetric{metric1},
			wantDeregistered: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c, st := newCleanupCollector()
			c.registry = newRegistry()

			group := newGroupedMetric()
			for k, m := range tc.metrics {
				group.metrics[k] = m
			}
			for k, ac := range tc.apiCounters {
				group.apiCounters[k] = ac
			}
			assert.NoError(t, c.registry.Register(registryKey, group))

			c.cleanupMetricCounters(registryKey, tc.counters, group, metric1)

			assert.ElementsMatch(t, tc.wantRemoved, st.removed)
			assert.Equal(t, !tc.wantDeregistered, c.registry.Get(registryKey) != nil)
		})
	}
}

// buildTestJWT creates a minimal JWT with the given claims. The signature is
// not verified by GetOrgGUID (uses ParseUnverified), so a placeholder is fine.
func buildTestJWT(claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT","alg":"HS256"}`))
	payload, _ := json.Marshal(claims)
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".fakesig"
}

func TestGetOrgGUID(t *testing.T) {
	const wantOrgGUID = "1234-1234-1234-1234" // matches org_guid in accessToken

	cases := map[string]struct {
		setupToken string // token returned by mock auth server; empty = InitializeForTest(nil)
		wantGUID   string
	}{
		"valid token with org_guid returns GUID": {
			setupToken: accessToken,
			wantGUID:   wantOrgGUID,
		},
		"no auth token returns empty string": {
			wantGUID: "",
		},
		"malformed token returns empty string": {
			setupToken: "not-a-jwt",
			wantGUID:   "",
		},
		"valid JWT without org_guid claim returns empty string": {
			setupToken: buildTestJWT(map[string]interface{}{"sub": "test-user", "iss": "test"}),
			wantGUID:   "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			token := tc.setupToken
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if token == "" {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				w.Write([]byte(`{"access_token":"` + token + `","expires_in":3600}`))
			}))
			defer srv.Close()
			cfg := createCentralCfg(srv.URL, "test-env")
			agent.Initialize(cfg)
			assert.Equal(t, tc.wantGUID, GetOrgGUID())
		})
	}
}

const raceTestIterations = 20000

// TestGroupStartTimeRace guards against an issue where AddMetricDetail/AddAPIMetricDetail
// released c.lock/c.batchLock before finishing their update, leaving a window where a concurrent
// publish cycle's reset of c.metricStartTime (see generateEvents) could be read as the zero value
// and inserted into a metric's groupStartTime.
func TestGroupStartTimeRace(t *testing.T) {
	apiDetails := models.APIDetails{ID: "race-api", Name: "race-api"}
	appDetails := models.AppDetails{ID: "race-app", Name: "race-app"}

	tests := map[string]struct {
		addMetric func(c *collector)
	}{
		"AddMetricDetail": {
			addMetric: func(c *collector) {
				c.AddMetricDetail(Detail{
					APIDetails: apiDetails,
					AppDetails: appDetails,
					StatusCode: "200",
					Duration:   10,
					Bytes:      10,
				})
			},
		},
		"AddAPIMetricDetail": {
			addMetric: func(c *collector) {
				c.AddAPIMetricDetail(MetricDetail{
					APIDetails: apiDetails,
					AppDetails: appDetails,
					StatusCode: "200",
					Count:      1,
				})
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			runGroupStartTimeRaceCase(t, tc.addMetric)
		})
	}
}

// runGroupStartTimeRaceCase builds a fresh collector, races addMetric against a simulated
// generateEvents() reset of c.metricStartTime, then fails if any metric ends up with a
// zero-value groupStartTime.
func runGroupStartTimeRaceCase(t *testing.T, addMetric func(c *collector)) {
	t.Helper()
	defer cleanUpCachedMetricFile()
	s := &testHTTPServer{}
	defer s.closeServer()
	s.startServer()
	traceability.SetDataDirPath(".")

	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
	cfg.SetEnvironmentID(testEnvID)
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	c := createMetricCollector().(*collector)

	stop := make(chan struct{})
	var resetWG sync.WaitGroup
	resetWG.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				// exactly what generateEvents() does to c.metricStartTime, under the same lock
				c.lock.Lock()
				c.metricStartTime = time.Time{}
				c.lock.Unlock()
			}
		}
	})

	var addWG sync.WaitGroup
	for range raceTestIterations {
		addWG.Go(func() {
			addMetric(c)
		})
	}
	addWG.Wait()
	close(stop)
	resetWG.Wait()

	assertNoZeroGroupStartTime(t, c)
}

// assertNoZeroGroupStartTime fails the test if any metric in the collector's registry has a
// groupStartTime matching the Go zero-value time.Time.
func assertNoZeroGroupStartTime(t *testing.T, c *collector) {
	t.Helper()
	zeroMillis := time.Time{}.UnixMilli()
	found := 0
	c.registry.Each(func(name string, m any) {
		group, ok := m.(groupedMetrics)
		if !ok {
			return
		}
		for key, metric := range group.metrics {
			if metric.groupStartTime == zeroMillis {
				found++
				t.Logf("corrupted metric found: registry=%s key=%s groupStartTime=%d", name, key, metric.groupStartTime)
			}
		}
	})

	if found > 0 {
		t.Fatalf("found %d metric(s) with a zero-value groupStartTime out of %d add attempts", found, raceTestIterations)
	}
}
