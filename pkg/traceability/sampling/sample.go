package sampling

import (
	"sync"
	"sync/atomic"
	"time"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

type endpointsSampling struct {
	enabled       bool
	endpointsInfo map[string]management.TraceabilityAgentAgentstateSamplingEndpoints
	endpointsLock sync.Mutex
}

// sample - private struct that is used to keep track of the samples being taken
type sample struct {
	config             Sampling
	currentCounts      map[string]int
	samplingLock       sync.Mutex
	samplingCounter    int32
	counterResetPeriod *atomic.Int64
	counterResetStopCh chan struct{}
	enabled            bool
	endTime            time.Time
	endpointsSampling  endpointsSampling
	limit              int32
}

func (s *sample) EnableSampling(samplingLimit int32, samplingEndTime time.Time, endpointsInfo map[string]management.TraceabilityAgentAgentstateSamplingEndpoints) {
	if len(endpointsInfo) > 0 {
		s.endpointsSampling.enabled = true
		s.endpointsSampling.endpointsInfo = endpointsInfo
	}

	s.samplingLock.Lock()
	s.enabled = true
	s.samplingLock.Unlock()

	s.endTime = samplingEndTime
	s.limit = samplingLimit

	s.resetSamplingCounter()

	// start limit reset job; limit is reset every minute
	go s.samplingCounterReset()

	if s.endpointsSampling.enabled {
		go s.disableEndpointsSampling()
	} else {
		go s.disableSampling()
	}
}

func (s *sample) disableSampling() {
	disableTimer := time.NewTimer(time.Until(s.endTime))

	<-disableTimer.C

	s.samplingLock.Lock()
	s.enabled = false
	s.samplingLock.Unlock()

	// stop limit reset job when sampling is disabled
	s.counterResetStopCh <- struct{}{}
}

func (s *sample) disableEndpointsSampling() {
	wg := sync.WaitGroup{}

	for apiID, endpoint := range s.endpointsSampling.endpointsInfo {
		wg.Add(1)
		go func(apiID string, endpoint management.TraceabilityAgentAgentstateSamplingEndpoints) {
			disableTimer := time.NewTimer(time.Until(time.Time(endpoint.EndTime)))
			defer wg.Done()

			<-disableTimer.C

			s.endpointsSampling.endpointsLock.Lock()
			delete(s.endpointsSampling.endpointsInfo, apiID)
			s.endpointsSampling.endpointsLock.Unlock()
		}(apiID, endpoint)
	}

	wg.Wait()

	s.endpointsSampling.endpointsLock.Lock()
	s.endpointsSampling.enabled = false
	s.endpointsSampling.endpointsLock.Unlock()

	// stop limit reset job when sampling is disabled
	s.counterResetStopCh <- struct{}{}
}

func (s *sample) samplingCounterReset() {
	resetPeriod := time.Duration(s.counterResetPeriod.Load())
	nextLimiterPeriod := time.Now().Round(time.Duration(resetPeriod))
	<-time.NewTimer(time.Until(nextLimiterPeriod)).C
	s.resetSamplingCounter()

	ticker := time.NewTicker(resetPeriod)

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

	s.endpointsSampling.endpointsLock.Lock()
	defer s.endpointsSampling.endpointsLock.Unlock()

	onlyErrors := s.config.OnlyErrors

	// check if sampling is enabled
	if !s.enabled && !s.endpointsSampling.enabled {
		return false
	} else if s.endpointsSampling.enabled {
		if endpoint, ok := s.endpointsSampling.endpointsInfo[details.APIID]; !ok {
			return false
		} else {
			onlyErrors = endpoint.OnlyErrors
		}
	}

	// sampling limit per minute exceeded

	if s.limit <= s.samplingCounter {
		return false
	}

	hasFailedStatus := details.Status == "Failure"
	// sample only failed transaction if OnlyErrors is set to `true` and the transaction summary's status is an error
	if !hasFailedStatus && onlyErrors {
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

func (s *sample) GetSamplePercentage() float64 {
	return s.config.Percentage
}
