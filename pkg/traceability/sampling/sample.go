package sampling

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

type endpointsSampling struct {
	enabled       bool
	endpointsInfo map[string]bool
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
	resetterRunning    atomic.Bool
}

func (s *sample) EnableSampling(samplingLimit int32, samplingEndTime time.Time, endpointsInfo map[string]management.TraceabilityAgentAgentstateSamplingEndpoints) {
	if len(endpointsInfo) > 0 {
		go s.handleEndpointsSampling(endpointsInfo)
	}

	if time.Now().Before(samplingEndTime) {
		// only enable sampling if the end time is in the future
		s.samplingLock.Lock()
		s.enabled = true
		s.samplingLock.Unlock()

		s.endTime = samplingEndTime
		go s.disableSampling()
	}

	s.limit = samplingLimit
	s.resetSamplingCounter()

	// start limit reset job; limit is reset every minute
	go s.samplingCounterReset()
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

func (s *sample) handleEndpointsSampling(endpoints map[string]management.TraceabilityAgentAgentstateSamplingEndpoints) {
	s.endpointsSampling.endpointsLock.Lock()
	s.endpointsSampling.enabled = true
	s.endpointsSampling.endpointsLock.Unlock()

	wg := sync.WaitGroup{}

	for apiID, endpoint := range endpoints {
		s.endpointsSampling.endpointsLock.Lock()
		if _, ok := s.endpointsSampling.endpointsInfo[apiID]; ok {
			// skip if endpoint already exists
			s.endpointsSampling.endpointsLock.Unlock()
			continue
		}
		s.endpointsSampling.endpointsInfo[apiID] = endpoint.OnlyErrors
		s.endpointsSampling.endpointsLock.Unlock()

		wg.Add(1)
		go func(apiID string, endpointEndTime time.Time) {
			disableTimer := time.NewTimer(time.Until(endpointEndTime))
			defer wg.Done()

			<-disableTimer.C

			s.endpointsSampling.endpointsLock.Lock()
			delete(s.endpointsSampling.endpointsInfo, apiID)
			s.endpointsSampling.endpointsLock.Unlock()
		}(apiID, time.Time(endpoint.EndTime))
	}

	wg.Wait()
	s.resetEndpointSampling()
}

func (s *sample) resetEndpointSampling() {
	if len(s.endpointsSampling.endpointsInfo) > 0 {
		return
	}
	s.endpointsSampling.endpointsLock.Lock()
	s.endpointsSampling.enabled = false
	s.endpointsSampling.endpointsLock.Unlock()
	// stop limit reset job when sampling is disabled
	s.counterResetStopCh <- struct{}{}
}

func (s *sample) samplingCounterReset() {
	if s.resetterRunning.Load() {
		return // resetter is already running
	}
	s.resetterRunning.Store(true)
	resetPeriod := time.Duration(s.counterResetPeriod.Load())
	nextLimiterPeriod := time.Now().Round(resetPeriod)
	<-time.NewTimer(time.Until(nextLimiterPeriod)).C
	s.resetSamplingCounter()

	ticker := time.NewTicker(resetPeriod)

	defer ticker.Stop()
	for {
		select {
		case <-s.counterResetStopCh:
			s.resetterRunning.Store(false)
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
		apiID := strings.TrimPrefix(details.APIID, util.SummaryEventProxyIDPrefix)
		var ok bool
		onlyErrors, ok = s.endpointsSampling.endpointsInfo[apiID]
		if !ok {
			// if endpoint is not found or sampling is not enabled for this endpoint, return false
			return false
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
