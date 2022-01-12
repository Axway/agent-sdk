package sampling

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/stretchr/testify/assert"
)

func TestSamplingConfig(t *testing.T) {
	testCases := []struct {
		name           string
		errExpected    bool
		config         Sampling
		expectedConfig Sampling
	}{
		{
			name:        "Default Config",
			errExpected: false,
			config:      DefaultConfig(),
			expectedConfig: Sampling{
				Percentage: 10,
			},
		},
		{
			name:        "Good Custom Config",
			errExpected: false,
			config: Sampling{
				Percentage: 50,
			},
			expectedConfig: Sampling{
				Percentage: 50,
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
			name:        "Good Config, Report All Errors",
			errExpected: false,
			config: Sampling{
				Percentage:      50,
				ReportAllErrors: true,
			},
			expectedConfig: Sampling{
				Percentage: 50,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
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
	testCases := []struct {
		name             string
		apiTransactions  map[string]int
		testTransactions int
		expectedSampled  int
		config           Sampling
	}{
		{
			name: "All Transactions",
			apiTransactions: map[string]int{
				"id1": 1000,
				"id2": 1000,
			},
			expectedSampled: 2000,
			config: Sampling{
				Percentage: 100,
				PerAPI:     false,
			},
		},
		{
			name: "50% of Transactions",
			apiTransactions: map[string]int{
				"id1": 50,
				"id2": 50,
				"id3": 50,
				"id4": 50,
			}, // Total = 200
			expectedSampled: 100,
			config: Sampling{
				Percentage: 50,
				PerAPI:     false,
			},
		},
		{
			name: "25% of Transactions",
			apiTransactions: map[string]int{
				"id1": 105,
				"id2": 100,
				"id3": 50,
				"id4": 15,
				"id5": 5,
			}, // Total = 275
			expectedSampled: 75,
			config: Sampling{
				Percentage: 25,
				PerAPI:     false,
			},
		},
		{
			name: "10% of Transactions",
			apiTransactions: map[string]int{
				"id1": 1000,
				"id2": 1000,
			},
			expectedSampled: 200,
			config: Sampling{
				Percentage: 10,
				PerAPI:     false,
			},
		},
		{
			name: "1% of Transactions",
			apiTransactions: map[string]int{
				"id1": 1000,
				"id2": 1000,
			},
			expectedSampled: 20,
			config: Sampling{
				Percentage: 1,
				PerAPI:     false,
			},
		},
		{
			name: "0% of Transactions",
			apiTransactions: map[string]int{
				"id1": 1000,
				"id2": 1000,
			},
			expectedSampled: 0,
			config: Sampling{
				Percentage: 0,
				PerAPI:     false,
			},
		},
		{
			name: "50% per API of Transactions",
			apiTransactions: map[string]int{
				"id1": 50, // expect 50
				"id2": 50, // expect 50
				"id3": 50, // expect 50
				"id4": 50, // expect 50
			},
			expectedSampled: 200,
			config: Sampling{
				Percentage: 50,
				PerAPI:     true,
			},
		},
		{
			name: "25% per API of Transactions",
			apiTransactions: map[string]int{
				"id1": 105, // expect 30
				"id2": 100, // expect 25
				"id3": 50,  // expect 25
				"id4": 15,  // expect 15
				"id5": 5,   // expect 5
			},
			expectedSampled: 100,
			config: Sampling{
				Percentage: 25,
				PerAPI:     true,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			waitGroup := sync.WaitGroup{}
			sampleCounterLock := sync.Mutex{}
			err := SetupSampling(test.config, false)
			assert.Nil(t, err)

			sampled := 0

			for apiID, numCalls := range test.apiTransactions {
				waitGroup.Add(1)

				go func(wg *sync.WaitGroup, id string, calls int) {
					defer wg.Done()
					for i := 0; i < calls; i++ {
						testDetails := TransactionDetails{
							Status: "Success", // this does not matter at the moment
							APIID:  id,
						}
						sample, err := ShouldSampleTransaction(testDetails)
						if sample {
							sampleCounterLock.Lock()
							sampled++
							sampleCounterLock.Unlock()
						}
						assert.Nil(t, err)
					}
				}(&waitGroup, apiID, numCalls)
			}

			waitGroup.Wait()
			assert.Nil(t, err)
			assert.Equal(t, test.expectedSampled, sampled)
		})
	}
}

func createEvents(numberOfEvents, samplePercent int) []publisher.Event {
	events := []publisher.Event{}

	count := 0
	sampled := 0
	for i := 0; i < numberOfEvents; i++ {
		var event publisher.Event
		if count < samplePercent {
			sampled++
			event = createEvent(true)
		} else {
			event = createEvent(false)
		}
		events = append(events, event)
		count++
		if count == 100 {
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
			name:           "All Events",
			testEvents:     2000,
			eventsExpected: 2000,
			config: Sampling{
				Percentage: 100,
			},
		},
		{
			name:           "50% of Events",
			testEvents:     2000,
			eventsExpected: 1000,
			config: Sampling{
				Percentage: 50,
			},
		},
		{
			name:           "25% of Events",
			testEvents:     2000,
			eventsExpected: 500,
			config: Sampling{
				Percentage: 25,
			},
		},
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
			err := SetupSampling(test.config, false)
			assert.Nil(t, err)

			eventsInTest := createEvents(test.testEvents, test.config.Percentage)
			filterEvents, err := FilterEvents(eventsInTest)

			assert.Nil(t, err)
			assert.Len(t, filterEvents, test.eventsExpected)
		})
	}
}
