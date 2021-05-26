package sampling

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

// sample - private struct that is used to keep track of the samples being taken
type sample struct {
	config        Sampling
	currentCounts map[string]int
	counterLock   sync.Mutex
}

// ShouldSampleTransaction - receives the transaction details and returns true to sample it false to not
func (s *sample) ShouldSampleTransaction(details TransactionDetails) bool {
	if s.config.PerAPI && details.APIID != "" {
		return s.shouldSampleWithCounter(details.APIID)
	}
	return s.shouldSampleWithCounter(globalCounter)
}

func (s *sample) shouldSampleWithCounter(counterName string) bool {
	s.counterLock.Lock()
	defer s.counterLock.Unlock()
	// check if counter needs initiated
	if _, found := s.currentCounts[counterName]; !found {
		s.currentCounts[counterName] = 0
	}

	// Only sampling on percentage, not currently looking at the details
	shouldSample := false
	if s.currentCounts[counterName] < s.config.Percentage {
		shouldSample = true
	}
	s.currentCounts[counterName]++

	// reset the count once we hit 100 messages
	if s.currentCounts[counterName] == countMax {
		s.currentCounts[counterName] = 0
	}

	// return if we should sample this transaction
	return shouldSample
}

// FilterEvents - returns an array of events that are part of the sample
func (s *sample) FilterEvents(events []publisher.Event) []publisher.Event {
	if s.config.Percentage == countMax {
		return events // all events are sampled by default
	}

	sampledEvents := make([]publisher.Event, 0)
	for _, event := range events {
		if _, sampled := event.Content.Meta[SampleKey]; sampled {
			sampledEvents = append(sampledEvents, event)
		}
	}
	return sampledEvents
}
