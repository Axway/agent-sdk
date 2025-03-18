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
	samplingLock       sync.Mutex
	samplingCounter    int32
	counterResetPeriod time.Duration
	counterResetStopCh chan struct{}
	disableSamplingCH  chan struct{}
	enabled            bool
	endTime            time.Time
	limit              int32
}

func NewSample(counterResetPeriod time.Duration) *sample {
	if counterResetPeriod == 0 {
		counterResetPeriod = time.Minute
	}

	sampler := &sample{
		disableSamplingCH:  make(chan struct{}),
		counterResetStopCh: make(chan struct{}),
		counterResetPeriod: counterResetPeriod,
	}

	return sampler
}

func (s *sample) EnableSampling(samplingLimit int32, samplingEndTime time.Time) {
	s.enabled = true
	s.endTime = samplingEndTime
	s.limit = samplingLimit

	s.resetSamplingCounter()

	// start limit reset job; limit is reset every minute
	go s.samplingCounterReset()

	// disable sampling at endTime
	go s.disableSampling()
}

func (s *sample) DisableSampling() {
	s.samplingLock.Lock()
	defer s.samplingLock.Unlock()
	if s.enabled {
		s.disableSamplingCH <- struct{}{}
	}
}

func (s *sample) disableSampling() {
	disableTimer := time.NewTimer(time.Until(s.endTime))
	select {
	case <-disableTimer.C:
	case <-s.disableSamplingCH:
	}

	s.samplingLock.Lock()
	s.enabled = false
	s.samplingLock.Unlock()

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
	s.samplingLock.Lock()
	defer s.samplingLock.Unlock()
	s.samplingCounter = 0
}

// ShouldSampleTransaction - receives the transaction details and returns true to sample it false to not
func (s *sample) ShouldSampleTransaction(details TransactionDetails) bool {
	s.samplingLock.Lock()
	defer s.samplingLock.Unlock()

	// check if sampling is enabled
	if !s.enabled {
		return false
	}

	// sampling limit per minute exceeded

	if s.limit <= s.samplingCounter {
		return false
	}

	hasFailedStatus := details.Status == "Failure"
	// sample only failed transaction if OnlyErrors is set to `true` and the transaction summary's status is an error
	if !hasFailedStatus && s.config.OnlyErrors {
		return false
	}

	s.samplingCounter++

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
