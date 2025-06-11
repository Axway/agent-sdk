package sampling

import (
	"fmt"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	v1Time "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

func TestSamplingConfig(t *testing.T) {
	testCases := []struct {
		name           string
		errExpected    bool
		apicDeployment string
		qaOverride     string
		config         Sampling
		expectedConfig Sampling
	}{
		{
			name:        "Default Config",
			errExpected: false,
			config:      DefaultConfig(),
			expectedConfig: Sampling{
				Percentage: 0,
			},
		},
		{
			name:        "Good Custom Config",
			errExpected: false,
			config: Sampling{
				Percentage: 5,
			},
			expectedConfig: Sampling{
				Percentage: 5,
			},
		},
		{
			name:        "Bad Config Too Low",
			errExpected: true,
			config: Sampling{
				Percentage: -5,
			},
		},
		{
			name:        "Bad Config Too High",
			errExpected: true,
			config: Sampling{
				Percentage: 150,
			},
		},
		{
			name:           "QA Override for production",
			errExpected:    true,
			qaOverride:     "100",
			apicDeployment: "prod-eu",
			config: Sampling{
				Percentage: 150,
			},
		},
		{
			name:           "QA Override for non-production",
			errExpected:    false,
			qaOverride:     "100",
			apicDeployment: "preprod",
			config: Sampling{
				Percentage: 150,
			},
			expectedConfig: Sampling{
				Percentage: 100,
			},
		},
		{
			name:           "Invalid QA Override for non-production",
			errExpected:    true,
			qaOverride:     "150",
			apicDeployment: "preprod",
			config: Sampling{
				Percentage: 150,
			},
			expectedConfig: Sampling{
				Percentage: 1,
			},
		},
		{
			name:        "Good Config, Report All Errors",
			errExpected: false,
			config: Sampling{
				Percentage: 10,
				OnlyErrors: true,
			},
			expectedConfig: Sampling{
				Percentage: 10,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.NewTestCentralConfig(config.TraceabilityAgent)
			if test.apicDeployment != "" {
				centralCfg := cfg.(*config.CentralConfiguration)
				centralCfg.APICDeployment = test.apicDeployment
			}
			os.Setenv(qaSamplingPercentageEnvVar, test.qaOverride)

			err := SetupSampling(test.config, false, test.apicDeployment)
			if test.errExpected {
				assert.NotNil(t, err, "Expected the config to fail")
			} else {
				assert.Nil(t, err, "Expected the config to pass")
				assert.Equal(t, test.expectedConfig.Percentage, agentSamples.config.Percentage)
				percentage, _ := GetGlobalSamplingPercentage()
				assert.Equal(t, test.expectedConfig.Percentage, percentage)
			}
		})
	}
}

func TestShouldSample(t *testing.T) {
	type transactionCount struct {
		successCount int
	}
	testCases := []struct {
		skip                     bool
		name                     string
		globalSampling           bool
		apiTransactions          map[string]transactionCount
		maxSampled               int
		config                   Sampling
		subIDs                   map[string]string
		limit                    int32
		duration                 time.Duration
		counterResetPeriod       time.Duration
		endpointsInfo            map[string]management.TraceabilityAgentAgentstateSamplingEndpoints
		additionalEndpointsInfo  map[string]management.TraceabilityAgentAgentstateSamplingEndpoints
		expectedEndpointsSampled map[string]struct{}
	}{
		{
			skip:           false,
			name:           "Limit sampling to 10 per period",
			globalSampling: true,
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
			},
			maxSampled:         60,
			limit:              10,
			duration:           time.Second / 2,
			counterResetPeriod: time.Second / 10,
		},
		{
			skip:           false,
			name:           "Limit sampling to 100 per period",
			globalSampling: true,
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
			},
			maxSampled:         600,
			limit:              100,
			duration:           time.Second / 2,
			counterResetPeriod: time.Second / 10,
		},
		{
			skip:           false,
			name:           "Limit sampling to 1000 per period",
			globalSampling: true,
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
			},
			maxSampled:         6000,
			limit:              1000,
			duration:           time.Second / 2,
			counterResetPeriod: time.Second / 10,
		},
		{
			skip:           false,
			name:           "Limit sampling to 0",
			globalSampling: true,
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
			},
			maxSampled:         0,
			limit:              0,
			duration:           time.Second / 2,
			counterResetPeriod: time.Second / 10,
		},
		{
			skip: false,
			name: "Endpoints sampling enabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
				"id2": {successCount: 1000},
				"id3": {successCount: 1000},
				"id4": {successCount: 1000},
			},
			maxSampled:         600,
			limit:              100,
			duration:           time.Second / 2,
			counterResetPeriod: time.Second / 10,
			endpointsInfo: map[string]management.TraceabilityAgentAgentstateSamplingEndpoints{
				"id1": {BasePath: "/api/v1"},
				"id2": {BasePath: "/api/v2"},
			},
			expectedEndpointsSampled: map[string]struct{}{"id1": {}, "id2": {}},
		},
		{
			skip: false,
			name: "Endpoints sampling enabled with additional endpoints",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
				"id2": {successCount: 1000},
				"id3": {successCount: 1000},
				"id4": {successCount: 1000},
				"id5": {successCount: 1000},
				"id6": {successCount: 1000},
			},
			maxSampled:         800,
			limit:              100,
			duration:           time.Second / 2,
			counterResetPeriod: time.Second / 10,
			endpointsInfo: map[string]management.TraceabilityAgentAgentstateSamplingEndpoints{
				"id1": {BasePath: "/api/v1"},
				"id2": {BasePath: "/api/v2"},
			},
			additionalEndpointsInfo: map[string]management.TraceabilityAgentAgentstateSamplingEndpoints{
				"id1": {BasePath: "/api/v1"},
				"id2": {BasePath: "/api/v2"},
				"id3": {BasePath: "/api/v3"},
			},
			expectedEndpointsSampled: map[string]struct{}{"id1": {}, "id2": {}, "id3": {}},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.skip {
				t.Skip("skipping test")
			}
			waitGroup := sync.WaitGroup{}
			sampleCounterLock := sync.Mutex{}

			testEnd := time.Now().Truncate(test.counterResetPeriod).Add(test.duration)
			done := time.NewTicker(time.Until(testEnd))
			defer done.Stop()

			err := SetupSampling(test.config, false, "")
			endTime := time.Now().Add(-1 * time.Second) // reset endTime to avoid issues with the test
			if test.globalSampling {
				endTime = testEnd
			}

			period := &atomic.Int64{}
			period.Store(int64(test.counterResetPeriod))
			agentSamples.counterResetPeriod = period
			// update the endtime in the endpoints
			for i := range test.endpointsInfo {
				test.endpointsInfo[i] = management.TraceabilityAgentAgentstateSamplingEndpoints{
					BasePath:   test.endpointsInfo[i].BasePath,
					EndTime:    v1Time.Time(testEnd),
					OnlyErrors: test.endpointsInfo[i].OnlyErrors,
				}
			}
			agentSamples.EnableSampling(test.limit, endTime, test.endpointsInfo)
			assert.Nil(t, err)

			if len(test.additionalEndpointsInfo) > 0 {
				for i := range test.additionalEndpointsInfo {
					test.additionalEndpointsInfo[i] = management.TraceabilityAgentAgentstateSamplingEndpoints{
						BasePath:   test.endpointsInfo[i].BasePath,
						EndTime:    v1Time.Time(testEnd),
						OnlyErrors: test.endpointsInfo[i].OnlyErrors,
					}
				}
				agentSamples.EnableSampling(test.limit, endTime, test.additionalEndpointsInfo)
			}

			sampled := 0

			endpointsSampled := map[string]struct{}{}
			for apiID, numCalls := range test.apiTransactions {
				waitGroup.Add(1)

				var subID string
				if test.subIDs != nil {
					subID = test.subIDs[apiID]
				}

				go func(wg *sync.WaitGroup, id, subID string, calls transactionCount) {
					defer wg.Done()
					endTimeTimer := time.NewTimer(time.Until(testEnd))

					sampleFunc := func(id, subID string, status string) {
						testDetails := TransactionDetails{
							Status: status,
							APIID:  id,
							SubID:  subID,
						}
						sample, err := ShouldSampleTransaction(testDetails)
						if sample {
							sampleCounterLock.Lock()
							endpointsSampled[id] = struct{}{}
							sampled++
							sampleCounterLock.Unlock()
						}
						assert.Nil(t, err)
					}

					for {
						select {
						case <-endTimeTimer.C:
							return
						default:
							for i := 0; i < calls.successCount; i++ {
								sampleFunc(id, subID, "Success")
							}
						}
					}
				}(&waitGroup, apiID, subID, numCalls)
			}

			waitGroup.Wait()
			<-done.C
			time.Sleep(time.Second / 2) // wait for the sampling to finish

			assert.Nil(t, err)
			assert.LessOrEqual(t, sampled, test.maxSampled, "sampled transactions should be less than max sampled")
			if len(test.endpointsInfo) > 0 {
				agentSamples.endpointsSampling.endpointsLock.Lock()
				assert.Equal(t, 0, len(agentSamples.endpointsSampling.endpointsInfo),
					"Endpoints sampling should be disabled after the test")
				agentSamples.endpointsSampling.endpointsLock.Unlock()
			}
			// validate that only the expected endpoints are sampled
			if len(test.expectedEndpointsSampled) > 0 {
				for id := range endpointsSampled {
					assert.Contains(t, test.expectedEndpointsSampled, id, "endpoint %s should not have been sampled", id)
				}
				for id := range test.expectedEndpointsSampled {
					assert.Contains(t, endpointsSampled, id, "endpoint %s should have been sampled", id)
				}
			}
		})
	}
}

func createEvents(numberOfEvents int, samplePercent float64) []publisher.Event {
	events := []publisher.Event{}

	count := 0
	sampled := 0
	countMax := 100 * int(math.Pow(10, float64(numberOfDecimals(samplePercent))))
	limit := int(float64(countMax) * samplePercent / 100)
	for i := 0; i < numberOfEvents; i++ {
		var event publisher.Event
		if count < limit {
			sampled++
			event = createEvent(true)
		} else {
			event = createEvent(false)
		}
		events = append(events, event)
		count++
		if count == countMax {
			count = 0
		}
	}

	return events
}

func createEvent(sampled bool) publisher.Event {
	fieldsData := common.MapStr{
		"message": "message value",
	}
	meta := common.MapStr{}
	if sampled {
		meta.Put(SampleKey, true)
	}
	return publisher.Event{
		Content: beat.Event{
			Timestamp: time.Now(),
			Meta:      meta,
			Private:   nil,
			Fields:    fieldsData,
		},
	}
}

func TestFilterEvents(t *testing.T) {
	testCases := []struct {
		name           string
		testEvents     int
		eventsExpected int
		config         Sampling
	}{
		{
			name:           "10% of Events",
			testEvents:     2000,
			eventsExpected: 200,
			config: Sampling{
				Percentage: 10,
			},
		},
		{
			name:           "1% of Events",
			testEvents:     2000,
			eventsExpected: 20,
			config: Sampling{
				Percentage: 1,
			},
		},
		{
			name:           "0.1% of Events",
			testEvents:     2000,
			eventsExpected: 2,
			config: Sampling{
				Percentage: 0.1,
			},
		},
		{
			name:           "0% of Events",
			testEvents:     2000,
			eventsExpected: 0,
			config: Sampling{
				Percentage: 0,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := SetupSampling(test.config, false, "")
			assert.Nil(t, err)

			eventsInTest := createEvents(test.testEvents, test.config.Percentage)
			filterEvents, err := FilterEvents(eventsInTest)

			assert.Nil(t, err)
			assert.Len(t, filterEvents, test.eventsExpected)
		})
	}
}

func Test_SamplingPercentageDecimals(t *testing.T) {
	testCases := []struct {
		value                float64
		expectedNbOfDecimals int
	}{
		{
			value:                10.9654,
			expectedNbOfDecimals: 4,
		},
		{
			value:                2.34567890,
			expectedNbOfDecimals: 7,
		},
		{
			value:                0,
			expectedNbOfDecimals: 0,
		},
		{
			value:                100,
			expectedNbOfDecimals: 0,
		},
	}
	for _, test := range testCases {
		t.Run(fmt.Sprintf("%f", test.value), func(t *testing.T) {
			assert.Equal(t, numberOfDecimals(test.value), test.expectedNbOfDecimals)
		})
	}
}
