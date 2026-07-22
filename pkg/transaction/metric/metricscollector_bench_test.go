package metric

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	beatPub "github.com/elastic/beats/v7/libbeat/publisher"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

const benchManagedAppID = "bench-app-1"

// setupBenchmarkCollector spins up a real collector wired to a test lighthouse
// server (for usage events) and a captured mock traceability client (for metric
// events), so each ingress benchmark exercises the full publish path and can
// assert the published data matches what was fed in.
func setupBenchmarkCollector(b *testing.B, axwayManaged bool) (*collector, *testHTTPServer, *MockClient) {
	b.Helper()

	s := &testHTTPServer{}
	s.startServer()
	traceability.SetDataDirPath(".")

	cfg := createCentralCfg(s.server.URL, "demo")
	cfg.UsageReporting.(*config.UsageReportingConfiguration).URL = s.server.URL + testLighthouse
	cfg.MetricReporting.(*config.MetricReportingConfiguration).Publish = true
	cfg.SetEnvironmentID(testEnvID)
	cfg.SetAxwayManaged(axwayManaged)
	cmd.BuildDataPlaneType = "Azure"
	agent.Initialize(cfg)

	cm := agent.GetCacheManager()
	cm.AddAPIServiceInstance(createAPIServiceInstance(testInstID, testInstName, apiDetails1.ID))
	cm.AddManagedApplication(createManagedApplication(benchManagedAppID, testManagedApp1, testConsumerOrg))
	cm.AddAccessRequest(createAccessRequest("ac-bench-1", testAccessReq1, testManagedApp1, testInstID, testInstName, testSubscription1))

	traceStatus = healthcheck.OK
	runTestHealthcheck()

	mockClient := setupMockClient(0).(*MockClient)
	metricCollector := createMetricCollector().(*collector)

	b.Cleanup(func() {
		s.closeServer()
		cleanUpCachedMetricFile()
	})

	return metricCollector, s, mockClient
}

func benchAppDetails() models.AppDetails {
	return models.AppDetails{ID: benchManagedAppID, Name: testManagedApp1}
}

// getRawEventData decodes the "data" portion of a published metric event into a
// generic map. Custom units are marshaled as flattened keys under "units"
// (see Units.MarshalJSON) but are deliberately excluded from unmarshaling back
// into centralMetric (CustomUnits has a `json:"-"` tag), so getMetricFromEvent
// cannot recover them. This raw decode is needed to verify custom unit data
// survives the publish round trip.
func getRawEventData(event beatPub.Event) map[string]any {
	data, found := event.Content.Fields[messageKey]
	if !found {
		return nil
	}
	v4Event := make(map[string]any)
	if err := json.Unmarshal([]byte(data.(string)), &v4Event); err != nil {
		return nil
	}
	dataMap, _ := v4Event["data"].(map[string]any)
	return dataMap
}

// BenchmarkAddMetric benchmarks the usage/volume-only ingress path and verifies
// that every added transaction and byte is reflected in the published usage report.
func BenchmarkAddMetric(b *testing.B) {
	mc, s, _ := setupBenchmarkCollector(b, true)

	const bytesPerCall = 20

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.AddMetric(apiDetails1, "200", 10, bytesPerCall, testManagedApp1)
	}
	b.StopTimer()

	mc.Execute()
	mc.usagePublisher.Execute()

	assert.Equal(b, b.N, s.transactionCount)
	assert.Equal(b, b.N*bytesPerCall, s.transactionVolume)
}

// BenchmarkAddMetricDetail benchmarks the per-transaction histogram ingress path
// and verifies the single aggregated metric event published matches the total
// number of transactions added.
func BenchmarkAddMetricDetail(b *testing.B) {
	mc, s, mockClient := setupBenchmarkCollector(b, false)

	detail := Detail{
		APIDetails: apiDetails1,
		AppDetails: benchAppDetails(),
		StatusCode: "200",
		Duration:   15,
		Bytes:      10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.AddMetricDetail(detail)
	}
	b.StopTimer()

	mc.Execute()
	mc.usagePublisher.Execute()

	assert.Equal(b, b.N, s.transactionCount)

	if !assert.Len(b, mockClient.capturedEvents, 1) {
		return
	}
	metric := getMetricFromEvent(mockClient.capturedEvents[0])
	if !assert.NotNil(b, metric) {
		return
	}
	assert.Equal(b, int64(b.N), metric.Units.Transactions.Count)
	assert.Equal(b, apiDetails1.ID, metric.API.ID)
	assert.Equal(b, GetStatusText(detail.StatusCode), metric.Units.Transactions.Status)
}

// BenchmarkAddAPIMetricDetail benchmarks the batched response-code ingress path
// (several transactions reported per call) and verifies the published transaction
// count matches the total number of synthetic samples generated across all calls.
func BenchmarkAddAPIMetricDetail(b *testing.B) {
	mc, _, mockClient := setupBenchmarkCollector(b, false)

	const perCallCount = 20
	detail := MetricDetail{
		APIDetails: apiDetails1,
		AppDetails: benchAppDetails(),
		StatusCode: "200",
		Count:      perCallCount,
		Response:   ResponseMetrics{Min: 5, Max: 500, Avg: 120},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.AddAPIMetricDetail(detail)
	}
	b.StopTimer()

	mc.Execute()
	mc.usagePublisher.Execute()

	if !assert.Len(b, mockClient.capturedEvents, 1) {
		return
	}
	metric := getMetricFromEvent(mockClient.capturedEvents[0])
	if !assert.NotNil(b, metric) {
		return
	}
	assert.Equal(b, int64(b.N*perCallCount), metric.Units.Transactions.Count)
}

// BenchmarkAddCustomMetricDetail benchmarks the custom unit ingress path and
// verifies the published custom unit count matches the total count added.
func BenchmarkAddCustomMetricDetail(b *testing.B) {
	mc, _, mockClient := setupBenchmarkCollector(b, false)

	const perCallCount = 3
	const unitName = "unit-name"
	detail := models.CustomMetricDetail{
		APIDetails:  apiDetails1,
		AppDetails:  benchAppDetails(),
		Count:       perCallCount,
		UnitDetails: models.Unit{Name: unitName},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.AddCustomMetricDetail(detail)
	}
	b.StopTimer()

	mc.Execute()
	mc.usagePublisher.Execute()

	if !assert.Len(b, mockClient.capturedEvents, 1) {
		return
	}
	data := getRawEventData(mockClient.capturedEvents[0])
	if !assert.NotNil(b, data) {
		return
	}
	units, ok := data["units"].(map[string]any)
	if !assert.True(b, ok, "expected units in published event data") {
		return
	}
	unitData, ok := units[unitName].(map[string]any)
	if !assert.True(b, ok, "expected custom unit %q in published event data", unitName) {
		return
	}
	assert.Equal(b, float64(b.N*perCallCount), unitData["count"])
}

// BenchmarkAddAPIMetric benchmarks the ingress path used by callers that build a
// fully-formed APIMetric themselves (e.g. the agents-controller). Each call here
// targets a distinct subscription/app/api, so every call registers its own cached
// metric; Execute publishes one event per group and this verifies the published
// data for each group matches its corresponding input by count and API ID.
func BenchmarkAddAPIMetric(b *testing.B) {
	mc, _, mockClient := setupBenchmarkCollector(b, false)

	inputs := make([]*APIMetric, b.N)
	for i := 0; i < b.N; i++ {
		inputs[i] = &APIMetric{
			EventID:      fmt.Sprintf("bench-event-%d", i),
			Subscription: models.Subscription{ID: fmt.Sprintf("bench-sub-%d", i)},
			App:          models.AppDetails{ID: fmt.Sprintf("bench-app-%d", i)},
			API:          models.APIDetails{ID: fmt.Sprintf("bench-api-%d", i), Name: fmt.Sprintf("bench-api-%d", i)},
			StatusCode:   "200",
			Count:        int64(i + 1),
			Response:     ResponseMetrics{Min: 5, Max: 100, Avg: 50},
			Observation:  models.ObservationDetails{Start: 10, End: 20},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.AddAPIMetric(inputs[i])
	}
	b.StopTimer()

	if err := mc.Execute(); err != nil {
		b.Fatalf("failed to publish batch: %v", err)
	}

	if !assert.Len(b, mockClient.capturedEvents, b.N) {
		return
	}
	inputsByAPIID := make(map[string]*APIMetric, len(inputs))
	for _, in := range inputs {
		inputsByAPIID[in.API.ID] = in
	}
	for _, event := range mockClient.capturedEvents {
		metric := getMetricFromEvent(event)
		if !assert.NotNil(b, metric) {
			continue
		}
		in, ok := inputsByAPIID[metric.API.ID]
		if !assert.True(b, ok, "unexpected API ID %q in published event", metric.API.ID) {
			continue
		}
		assert.Equal(b, in.Count, metric.Units.Transactions.Count)
	}
}

// BenchmarkAllIngressesCombined drives every transaction ingress method the collector
// exposes (AddMetric, AddMetricDetail, AddAPIMetricDetail, AddCustomMetricDetail, and
// AddAPIMetric for both a transaction status and a custom unit) against the *same*
// subscription/app/api, so they all land in the same cached group. It verifies the
// transaction counts and custom unit counts reported through every ingress method
// aggregate into a single published metric event rather than clobbering or losing
// each other's contributions, and that the usage counter separately reflects only
// the ingress methods that report usage (AddMetric, AddMetricDetail, AddAPIMetricDetail).
func BenchmarkAllIngressesCombined(b *testing.B) {
	mc, s, mockClient := setupBenchmarkCollector(b, true)

	const (
		usageBytesPerCall           = 20
		metricDetailBytes           = 10
		apiMetricDetailPerCall      = 20
		apiMetricTransactionPerCall = 7
		customMetricDetailPerCall   = 3
		apiMetricCustomUnitPerCall  = 4
		unitName                    = "unit-name"
	)

	appDetails := benchAppDetails()

	detail := Detail{
		APIDetails: apiDetails1,
		AppDetails: appDetails,
		StatusCode: "200",
		Duration:   15,
		Bytes:      metricDetailBytes,
	}
	metricDetail := MetricDetail{
		APIDetails: apiDetails1,
		AppDetails: appDetails,
		StatusCode: "200",
		Count:      apiMetricDetailPerCall,
		Response:   ResponseMetrics{Min: 5, Max: 500, Avg: 120},
	}
	customDetail := models.CustomMetricDetail{
		APIDetails:  apiDetails1,
		AppDetails:  appDetails,
		Count:       customMetricDetailPerCall,
		UnitDetails: models.Unit{Name: unitName},
	}
	transactionAPIMetric := &APIMetric{
		Subscription: models.Subscription{ID: testSubscription1},
		App:          models.AppDetails{ID: appDetails.ID, Name: appDetails.Name},
		API:          models.APIDetails{ID: apiDetails1.ID, Name: apiDetails1.Name},
		StatusCode:   "200",
		Count:        apiMetricTransactionPerCall,
		Response:     ResponseMetrics{Min: 1, Max: 50, Avg: 10},
		Observation:  models.ObservationDetails{Start: 10, End: 20},
	}
	customUnitAPIMetric := &APIMetric{
		Subscription: models.Subscription{ID: testSubscription1},
		App:          models.AppDetails{ID: appDetails.ID, Name: appDetails.Name},
		API:          models.APIDetails{ID: apiDetails1.ID, Name: apiDetails1.Name},
		Unit:         &models.Unit{Name: unitName},
		Count:        apiMetricCustomUnitPerCall,
		Observation:  models.ObservationDetails{Start: 10, End: 20},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.AddMetric(apiDetails1, "200", 10, usageBytesPerCall, testManagedApp1)
		mc.AddMetricDetail(detail)
		mc.AddAPIMetricDetail(metricDetail)
		mc.AddCustomMetricDetail(customDetail)
		mc.AddAPIMetric(transactionAPIMetric)
		mc.AddAPIMetric(customUnitAPIMetric)
	}
	b.StopTimer()

	mc.Execute()
	mc.usagePublisher.Execute()

	// AddMetric, AddMetricDetail (which calls AddMetric internally), and AddAPIMetricDetail
	// all report to the usage/transaction counter; AddCustomMetricDetail and AddAPIMetric do not.
	expectedUsageTransactions := b.N * (1 + 1 + apiMetricDetailPerCall)
	assert.Equal(b, expectedUsageTransactions, s.transactionCount)
	expectedVolume := b.N * (usageBytesPerCall + metricDetailBytes)
	assert.Equal(b, expectedVolume, s.transactionVolume)

	// every ingress reported against the same subscription/app/api, so they should all
	// have aggregated into a single published metric event
	if !assert.Len(b, mockClient.capturedEvents, 1) {
		return
	}

	metric := getMetricFromEvent(mockClient.capturedEvents[0])
	if !assert.NotNil(b, metric) || !assert.NotNil(b, metric.Units) || !assert.NotNil(b, metric.Units.Transactions) {
		return
	}
	expectedTransactionCount := int64(b.N) * int64(1+apiMetricDetailPerCall+apiMetricTransactionPerCall)
	assert.Equal(b, expectedTransactionCount, metric.Units.Transactions.Count)

	data := getRawEventData(mockClient.capturedEvents[0])
	if !assert.NotNil(b, data) {
		return
	}
	units, ok := data["units"].(map[string]any)
	if !assert.True(b, ok, "expected units in published event data") {
		return
	}
	unitData, ok := units[unitName].(map[string]any)
	if !assert.True(b, ok, "expected custom unit %q in published event data", unitName) {
		return
	}
	expectedCustomUnitCount := float64(b.N) * float64(customMetricDetailPerCall+apiMetricCustomUnitPerCall)
	assert.Equal(b, expectedCustomUnitCount, unitData["count"])
}
