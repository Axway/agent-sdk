package sampling

import (
	"fmt"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
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
			agent.Initialize(cfg)

			err := SetupSampling(test.config, false)
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
		errorCount   int
	}
	testCases := []struct {
		name            string
		apiTransactions map[string]transactionCount
		expectedSampled int
		config          Sampling
		subIDs          map[string]string
	}{
		{
			name: "Maximum Transactions",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
				"id2": {successCount: 1000},
			},
			expectedSampled: 200,
			config: Sampling{
				Percentage: 10,
				PerAPI:     false,
			},
		},
		{
			name: "Default config transactions",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
				"id2": {successCount: 1000},
			},
			expectedSampled: 0,
			config:          DefaultConfig(),
		},
		{
			name: "5% of Transactions when per api is disabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 50},
				"id2": {successCount: 50},
				"id3": {successCount: 50},
				"id4": {successCount: 50},
			}, // Total = 200
			expectedSampled: 10,
			config: Sampling{
				Percentage: 5,
				PerAPI:     false,
			},
		},
		{
			name: "10% of Transactions when per api is disabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
				"id2": {successCount: 1000},
			},
			expectedSampled: 200,
			config: Sampling{
				Percentage: 10,
				PerAPI:     false,
			},
		},
		{
			name: "0.55% of Transactions when per api is disabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 10000},
			},
			expectedSampled: 55,
			config: Sampling{
				Percentage: 0.55,
				PerAPI:     false,
			},
		},
		{
			name: "9.99% of Transactions when per api is disabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 10000},
				"id2": {successCount: 10000},
				"id3": {successCount: 10000},
			},
			expectedSampled: 30000 * 999 / 10000,
			config: Sampling{
				Percentage: 9.99,
				PerAPI:     false,
			},
		},
		{
			name: "0.0006% of Transactions when per api is disabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 2000000},
			},
			expectedSampled: 12,
			config: Sampling{
				Percentage: 0.0006,
				PerAPI:     false,
			},
		},
		{
			name: "1% of Transactions when per api is disabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
				"id2": {successCount: 1000},
			},
			expectedSampled: 20,
			config: Sampling{
				Percentage: 1,
				PerAPI:     false,
			},
		},
		{
			name: "0% of Transactions when per api is disabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 1000},
				"id2": {successCount: 1000},
			},
			expectedSampled: 0,
			config: Sampling{
				Percentage: 0,
				PerAPI:     false,
			},
		},
		{
			name: "5% per API of Transactions when per api is enabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 50}, // expect 50
				"id2": {successCount: 50}, // expect 50
				"id3": {successCount: 50}, // expect 50
				"id4": {successCount: 50}, // expect 50
			},
			expectedSampled: 20,
			config: Sampling{
				Percentage: 5,
				PerAPI:     true,
			},
		},
		{
			name: "5% of subscription transactions when per api and per sub are enabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 50}, // expect 50
				"id2": {successCount: 50}, // expect 50
				"id3": {successCount: 50}, // expect 50
				"id4": {successCount: 50}, // expect 50
			},
			subIDs: map[string]string{
				"id1": "sub1",
				"id2": "sub2",
				"id3": "sub3",
				"id4": "sub4",
			},
			expectedSampled: 20,
			config: Sampling{
				Percentage: 5,
				PerAPI:     true,
				PerSub:     true,
			},
		},
		{
			name: "5% of subscription transactions when per api is disabled and per sub is enabled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 50}, // expect 50
				"id2": {successCount: 50}, // expect 50
				"id3": {successCount: 50}, // expect 50
				"id4": {successCount: 50}, // expect 50
			},
			subIDs: map[string]string{
				"id1": "sub1",
				"id2": "sub2",
				"id3": "sub3",
				"id4": "sub4",
			},
			expectedSampled: 20,
			config: Sampling{
				Percentage: 5,
				PerAPI:     false,
				PerSub:     true,
			},
		},
		{
			name: "5% of per API transactions when per api and per sub are enabled, but no subID is found",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 50}, // expect 50
				"id2": {successCount: 50}, // expect 50
				"id3": {successCount: 50}, // expect 50
				"id4": {successCount: 50}, // expect 50
			},
			subIDs:          map[string]string{},
			expectedSampled: 20,
			config: Sampling{
				Percentage: 5,
				PerAPI:     true,
				PerSub:     true,
			},
		},
		{
			name: "only errors to be sampled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 500, errorCount: 500},
				"id2": {successCount: 500, errorCount: 500},
			},
			expectedSampled: 100,
			config: Sampling{
				Percentage: 10,
				OnlyErrors: true,
			},
		},
		{
			name: "errors and success to be sampled",
			apiTransactions: map[string]transactionCount{
				"id1": {successCount: 500, errorCount: 500},
				"id2": {successCount: 500, errorCount: 500},
			},
			expectedSampled: 200,
			config: Sampling{
				Percentage: 10,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			waitGroup := sync.WaitGroup{}
			sampleCounterLock := sync.Mutex{}
			centralCfg := config.NewTestCentralConfig(config.TraceabilityAgent)
			agent.Initialize(centralCfg)

			err := SetupSampling(test.config, false)
			assert.Nil(t, err)

			sampled := 0

			for apiID, numCalls := range test.apiTransactions {
				waitGroup.Add(1)

				var subID string
				if test.subIDs != nil {
					subID = test.subIDs[apiID]
				}

				go func(wg *sync.WaitGroup, id, subID string, calls transactionCount) {
					defer wg.Done()
					sampleFunc := func(id, subID string, status string) {
						testDetails := TransactionDetails{
							Status: status,
							APIID:  id,
							SubID:  subID,
						}
						sample, err := ShouldSampleTransaction(testDetails)
						if sample {
							sampleCounterLock.Lock()
							sampled++
							sampleCounterLock.Unlock()
						}
						assert.Nil(t, err)
					}
					for i := 0; i < calls.successCount; i++ {
						sampleFunc(id, subID, "Success")
					}
					for i := 0; i < calls.errorCount; i++ {
						sampleFunc(id, subID, "Failure")
					}
				}(&waitGroup, apiID, subID, numCalls)
			}

			waitGroup.Wait()
			assert.Nil(t, err)
			assert.Equal(t, test.expectedSampled, sampled)
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
			centralCfg := config.NewTestCentralConfig(config.TraceabilityAgent)
			agent.Initialize(centralCfg)

			err := SetupSampling(test.config, false)
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
