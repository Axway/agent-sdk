package sampling

import (
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

// sample - private struct that is used to keep track of the samples being taken
type sample struct {
	config             Sampling
	currentCounts      map[string]int
	counterLock        sync.Mutex
	samplingCounter    int32
	counterResetPeriod time.Duration
	counterResetStopCh chan struct{}
}

func (s *sample) EnableSampling(samplingLimit int32, samplingEndTime time.Time) {
	s.config.enabled = true
	s.config.endTime = samplingEndTime
	s.config.limit = samplingLimit

	s.resetSamplingCounter()

	// start limit reset job; limit is reset every minute
	go s.samplingCounterReset()

	// disable sampling at endTime
	go s.disableSampling()
}

func (s *sample) disableSampling() {
	disableTimer := time.NewTimer(time.Until(s.config.endTime))
	<-disableTimer.C

	s.config.enabled = false

	// stop limit reset job when sampling is disabled
	s.counterResetStopCh <- struct{}{}
}

func (s *sample) samplingCounterReset() {
	nextLimiterPeriod := time.Now().Round(s.counterResetPeriod)
	<-time.NewTimer(time.Until(nextLimiterPeriod)).C
	s.resetSamplingCounter()

	ticker := time.NewTicker(s.counterResetPeriod)

	defer ticker.Stop()
	for {
		select {
		case <-s.counterResetStopCh:
			return
		case <-ticker.C:
			s.resetSamplingCounter()
		}
	}
}

func (s *sample) resetSamplingCounter() {
	s.counterLock.Lock()
	defer s.counterLock.Unlock()
	s.samplingCounter = 0
}

// ShouldSampleTransaction - receives the transaction details and returns true to sample it false to not
func (s *sample) ShouldSampleTransaction(details TransactionDetails) bool {
	// check if sampling is enabled
	if !s.config.enabled {
		return false
	}

	// sampling limit per minute exceeded
	if s.config.limit <= s.samplingCounter {
		return false
	}

	hasFailedStatus := details.Status == "Failure"
	// sample only failed transaction if OnlyErrors is set to `true` and the transaction summary's status is an error
	if !hasFailedStatus && s.config.OnlyErrors {
		return false
	}

	s.counterLock.Lock()
	s.samplingCounter++
	s.counterLock.Unlock()

	return true
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
