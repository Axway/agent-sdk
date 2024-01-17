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
				Percentage: 1,
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
			name:        "Good Config, Report All Errors",
			errExpected: false,
			config: Sampling{
				Percentage:      10,
				ReportAllErrors: true,
			},
			expectedConfig: Sampling{
				Percentage: 10,
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
		subIDs           map[string]string
	}{
		{
			name: "Maximum Transactions",
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
			name: "5% of Transactions when per api is disabled",
			apiTransactions: map[string]int{
				"id1": 50,
				"id2": 50,
				"id3": 50,
				"id4": 50,
			}, // Total = 200
			expectedSampled: 10,
			config: Sampling{
				Percentage: 5,
				PerAPI:     false,
			},
		},
		{
			name: "10% of Transactions when per api is disabled",
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
			name: "1% of Transactions when per api is disabled",
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
			name: "0% of Transactions when per api is disabled",
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
			name: "5% per API of Transactions when per api is enabled",
			apiTransactions: map[string]int{
				"id1": 50, // expect 50
				"id2": 50, // expect 50
				"id3": 50, // expect 50
				"id4": 50, // expect 50
			},
			expectedSampled: 20,
			config: Sampling{
				Percentage: 5,
				PerAPI:     true,
			},
		},
		{
			name: "5% of subscription transactions when per api and per sub are enabled",
			apiTransactions: map[string]int{
				"id1": 50, // expect 50
				"id2": 50, // expect 50
				"id3": 50, // expect 50
				"id4": 50, // expect 50
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
			apiTransactions: map[string]int{
				"id1": 50, // expect 50
				"id2": 50, // expect 50
				"id3": 50, // expect 50
				"id4": 50, // expect 50
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
			apiTransactions: map[string]int{
				"id1": 50, // expect 50
				"id2": 50, // expect 50
				"id3": 50, // expect 50
				"id4": 50, // expect 50
			},
			subIDs:          map[string]string{},
			expectedSampled: 20,
			config: Sampling{
				Percentage: 5,
				PerAPI:     true,
				PerSub:     true,
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

				var subID string
				if test.subIDs != nil {
					subID = test.subIDs[apiID]
				}

				go func(wg *sync.WaitGroup, id, subID string, calls int) {
					defer wg.Done()
					for i := 0; i < calls; i++ {
						testDetails := TransactionDetails{
							Status: "Success", // this does not matter at the moment
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
				}(&waitGroup, apiID, subID, numCalls)
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
