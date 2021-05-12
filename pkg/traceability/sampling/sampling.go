package sampling

import "github.com/elastic/beats/v7/libbeat/publisher"

// Global Agent samples
var agentSamples sample

// Sampling - configures the sampling of events the agent sends to Amplify
type Sampling struct {
	Percentage int `config:"percentage"    validate:"min=0, max=100"`
}

//DefaultConfig - returns a default sampling config where all transactions are sent
func DefaultConfig() Sampling {
	return Sampling{
		Percentage: 100,
	}
}

// sample - private struct that is used to keep track of the samples being taken
type sample struct {
	config       Sampling
	currentCount int
}

// SetupSampling - set up redactionRegex based on the redactionConfig
func SetupSampling(cfg Sampling) error {
	agentSamples = sample{
		config:       cfg,
		currentCount: 0, // counter of events, only up to sample are returned
	}
	return nil
}

// ShouldSampleTransaction - receives the transaction details and returns true to sample it false to not
func ShouldSampleTransaction(details TransactionDetails) bool {
	return agentSamples.ShouldSampleTransaction(details)
}

// ShouldSampleTransaction - receives the transaction details and returns true to sample it false to not
func (s *sample) ShouldSampleTransaction(details TransactionDetails) bool {
	// Only sampling on percentage, not currently looking at the details
	samplePercent := s.config.Percentage
	shouldSample := false
	if s.currentCount < samplePercent {
		shouldSample = true
	}
	s.currentCount++

	// resent te count once we hit 100 messages
	if s.currentCount == 100 {
		s.currentCount = 0
	}

	// return if we should sample this transaction
	return shouldSample
}

// FilterEvents - returns an array of events that are part of the sample
func FilterEvents(events []publisher.Event) []publisher.Event {
	sampledEvents := make([]publisher.Event, 0)
	for _, event := range events {
		if _, sampled := event.Content.Meta[SampleKey]; sampled {
			sampledEvents = append(sampledEvents, event)
		}
	}
	return sampledEvents
}

// // SampleEvents - takes a batch of events and returns a batch of events to send
// func SampleEvents(eventsPointer *[]publisher.Event) {
// 	agentSamples.SampleEvents(eventsPointer)
// }

// // SampleEvents - takes a batch of events and returns a batch of events to send
// func (s *sample) SampleEvents(eventsPointer *[]publisher.Event) {
// 	// Check for send all
// 	samplePercent := s.config.Percentage
// 	if samplePercent == 100 {
// 		return
// 	}

// 	// Group the events by the id in meta data
// 	groupedEvents := make(map[string]*eventGroup)
// 	// groupedEvents := groupEvents(*eventsPointer)

// 	sampledEvents := make([]publisher.Event, 0)
// 	for id := range groupedEvents {
// 		if _, sampled := groupedEvents[id].summary.Content.Meta["sampled"]; sampled {
// 			// this must be retry of a sampled event. add them to the array but do not count them
// 			log.Tracef("found events already sampled")
// 			sampledEvents = append(sampledEvents, groupedEvents[id].summary)
// 			sampledEvents = append(sampledEvents, groupedEvents[id].legs...)
// 			continue
// 		}
// 		if s.currentCount < samplePercent {
// 			sampledEvents = append(sampledEvents, groupedEvents[id].summary)
// 			sampledEvents = append(sampledEvents, groupedEvents[id].legs...)
// 			groupedEvents[id].summary.Content.Meta["sampled"] = true
// 		}
// 		s.currentCount++ // increase the current count, even when not sending event
// 		if s.currentCount >= 100 {
// 			// reset the counters
// 			s.currentCount = 0
// 		}
// 	}

// 	// Point the batch to the new array of events
// 	*eventsPointer = sampledEvents
// }

// group events by the meta data id
// func groupEvents(events []publisher.Event) map[string]*eventGroup {
// 	// Group the events by the id in meta data
// 	groupedEvents := make(map[string]*eventGroup)

// 	for _, event := range events {
// 		// Check that the event has an ID before continuing
// 		id, ok := event.Content.Meta["id"]
// 		if !ok {
// 			continue
// 		}

// 		// Check that the ID is a string type
// 		eventID, ok := id.(string)
// 		if !ok {
// 			continue
// 		}

// 		// Create the eventGroup, if it does not exist
// 		if _, found := groupedEvents[eventID]; !found {
// 			groupedEvents[eventID] = &eventGroup{
// 				legs: make([]publisher.Event, 0),
// 			}
// 		}

// 		// Get message field
// 		var message string
// 		if data, ok := event.Content.Fields["message"]; ok {
// 			message = data.(string)
// 		}

// 		// unpack message to a logEvent
// 		var logEvent models.LogEvent
// 		err := json.Unmarshal([]byte(message), &logEvent)
// 		if err != nil {
// 			log.Errorf("error reading event %s", err.Error())
// 			continue
// 		}

// 		if logEvent.TransactionSummary != nil {
// 			groupedEvents[eventID].summary = event
// 			log.Tracef("Summary: %+v", logEvent.TransactionSummary)
// 		} else if logEvent.TransactionEvent != nil {
// 			groupedEvents[eventID].legs = append(groupedEvents[eventID].legs, event)
// 			log.Tracef("Details: %+v", logEvent.TransactionEvent)
// 		} else {
// 			log.Tracef("Unknown Log Event Type: %+v", logEvent)
// 		}
// 	}

// 	return groupedEvents
// }
