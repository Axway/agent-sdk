package sampling

import "github.com/elastic/beats/v7/libbeat/publisher"

// sample - private struct that is used to keep track of the samples being taken
type sample struct {
	config       Sampling
	currentCount int
}

// ShouldSampleTransaction - receives the transaction details and returns true to sample it false to not
func (s *sample) ShouldSampleTransaction(details TransactionDetails) bool {
	// Only sampling on percentage, not currently looking at the details
	shouldSample := false
	if s.currentCount < s.config.Percentage {
		shouldSample = true
	}
	s.currentCount++

	// reset the count once we hit 100 messages
	if s.currentCount == 100 {
		s.currentCount = 0
	}

	// return if we should sample this transaction
	return shouldSample
}

// FilterEvents - returns an array of events that are part of the sample
func (s *sample) FilterEvents(events []publisher.Event) []publisher.Event {
	if s.config.Percentage == 100 {
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
