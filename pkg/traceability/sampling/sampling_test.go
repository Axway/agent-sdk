package sampling

import (
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
				Percentage: 100,
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
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := SetupSampling(test.config)
			if test.errExpected {
				assert.NotNil(t, err, "Expected the config to fail")
			} else {
				assert.Nil(t, err, "Expected the config to pass")
				assert.Equal(t, test.expectedConfig.Percentage, agentSamples.config.Percentage)
			}
		})
	}
}

func TestShouldSample(t *testing.T) {
	testCases := []struct {
		name             string
		testTransactions int
		expectedSampled  int
		config           Sampling
	}{
		{
			name:             "All Transactions",
			testTransactions: 2000,
			expectedSampled:  2000,
			config: Sampling{
				Percentage: 100,
			},
		},
		{
			name:             "50% of Transactions",
			testTransactions: 2000,
			expectedSampled:  1000,
			config: Sampling{
				Percentage: 50,
			},
		},
		{
			name:             "25% of Transactions",
			testTransactions: 2000,
			expectedSampled:  500,
			config: Sampling{
				Percentage: 25,
			},
		},
		{
			name:             "10% of Transactions",
			testTransactions: 2000,
			expectedSampled:  200,
			config: Sampling{
				Percentage: 10,
			},
		},
		{
			name:             "1% of Transactions",
			testTransactions: 2000,
			expectedSampled:  20,
			config: Sampling{
				Percentage: 1,
			},
		},
		{
			name:             "0% of Transactions",
			testTransactions: 2000,
			expectedSampled:  0,
			config: Sampling{
				Percentage: 0,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := SetupSampling(test.config)
			assert.Nil(t, err)

			sampled := 0
			for i := 0; i < test.testTransactions; i++ {
				testDetails := TransactionDetails{
					Status: "Success", // this does not matter at the moment
				}
				if ShouldSampleTransaction(testDetails) {
					sampled++
				}
			}

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
			err := SetupSampling(test.config)
			assert.Nil(t, err)

			eventsInTest := createEvents(test.testEvents, test.config.Percentage)
			filterEvents := FilterEvents(eventsInTest)

			assert.Len(t, filterEvents, test.eventsExpected)
		})
	}
}
