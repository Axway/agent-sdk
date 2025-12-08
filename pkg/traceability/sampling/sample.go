package sampling

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

type endpointsSampling struct {
	enabled       atomic.Bool
	endpointsInfo map[string]bool
	endpointsLock sync.Mutex
}

type concurrentTime struct {
	endTime time.Time
	mu      sync.RWMutex
}

func (ct *concurrentTime) SetEndTime(endTime time.Time) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.endTime = endTime
}

func (ct *concurrentTime) GetEndTime() time.Time {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.endTime
}

// sample - private struct that is used to keep track of the samples being taken
type sample struct {
	config              Sampling
	currentCounts       map[string]int
	samplingLock        sync.Mutex
	samplingCounter     int32
	counterResetPeriod  *atomic.Int64
	counterResetStopCh  chan struct{}
	enabled             atomic.Bool
	samplingTime        concurrentTime
	endpointsSampling   endpointsSampling
	limit               int32
	resetterRunning     atomic.Bool
	apiAppErrorSampling map[string]struct{}         // key: apiID - appID, value: doesn't matter, only key presence is used
	externalAppKeyData  definitions.ExternalAppData // field used to obtain external app value from agent details
}

func (s *sample) EnableSampling(samplingLimit int32, samplingEndTime time.Time, endpointsInfo map[string]management.TraceabilityAgentAgentstateSamplingEndpoints) {
	if len(endpointsInfo) > 0 {
		go s.handleEndpointsSampling(endpointsInfo)
	}

	if time.Now().Before(samplingEndTime) {
		// only enable sampling if the end time is in the future
		s.enabled.Store(true)
		s.samplingTime.SetEndTime(samplingEndTime)
		go s.disableSampling()
	}

	s.limit = samplingLimit
	s.resetSamplingCounter()

	// start limit reset job; limit is reset every minute
	go s.samplingCounterReset()
}

func (s *sample) disableSampling() {
	disableTimer := time.NewTimer(time.Until(s.samplingTime.GetEndTime()))
	<-disableTimer.C

	s.enabled.Store(false)
	// stop limit reset job when sampling is disabled

	s.stopCounterResetter()
}

func (s *sample) handleEndpointsSampling(endpoints map[string]management.TraceabilityAgentAgentstateSamplingEndpoints) {
	s.endpointsSampling.enabled.Store(true)

	wg := sync.WaitGroup{}

	for apiID, endpoint := range endpoints {
		if !s.addEndpointSampling(apiID, endpoint.OnlyErrors) {
			// skip if endpoint already exists
			continue
		}

		wg.Add(1)
		go func(apiID string, endpointEndTime time.Time) {
			disableTimer := time.NewTimer(time.Until(endpointEndTime))
			defer wg.Done()

			<-disableTimer.C

			s.removeEndpointSampling(apiID)
		}(apiID, time.Time(endpoint.EndTime))
	}

	wg.Wait()
	s.resetEndpointSampling()
}

func (s *sample) addEndpointSampling(apiID string, onlyErrors bool) bool {
	s.endpointsSampling.endpointsLock.Lock()
	defer s.endpointsSampling.endpointsLock.Unlock()
	if _, ok := s.endpointsSampling.endpointsInfo[apiID]; ok {
		// endpoint already exists, no need to add it again
		return false
	}
	s.endpointsSampling.endpointsInfo[apiID] = onlyErrors
	return true
}

func (s *sample) removeEndpointSampling(apiID string) {
	s.endpointsSampling.endpointsLock.Lock()
	defer s.endpointsSampling.endpointsLock.Unlock()
	delete(s.endpointsSampling.endpointsInfo, apiID)
}

func (s *sample) sampleEndpointAndOnlyErrors(apiID string) (bool, bool) {
	s.endpointsSampling.endpointsLock.Lock()
	defer s.endpointsSampling.endpointsLock.Unlock()
	if _, ok := s.endpointsSampling.endpointsInfo[apiID]; !ok {
		return false, false // endpoint not found
	}
	return true, s.endpointsSampling.endpointsInfo[apiID] // endpoint found, return onlyErrors
}

func (s *sample) resetEndpointSampling() {
	s.endpointsSampling.endpointsLock.Lock()
	defer s.endpointsSampling.endpointsLock.Unlock()
	if len(s.endpointsSampling.endpointsInfo) > 0 {
		return
	}
	s.endpointsSampling.enabled.Store(false)
	s.stopCounterResetter()
}

func (s *sample) stopCounterResetter() {
	if !s.resetterRunning.Load() {
		return // resetter already stopped
	}
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
	onlyErrors := s.config.OnlyErrors

	statusText := GetStatusFromCodeString(details.Status)

	// if both are disabled, skip. if endpoints is enabled and sampling is disabled, check if the endpoint is found
	if !s.enabled.Load() && !s.endpointsSampling.enabled.Load() {
		return false
	} else if s.endpointsSampling.enabled.Load() && !s.enabled.Load() {
		apiID := strings.TrimPrefix(details.APIID, util.SummaryEventProxyIDPrefix)
		var found bool
		found, onlyErrors = s.sampleEndpointAndOnlyErrors(apiID)
		if !found {
			// if endpoint is not found and sampling is not enabled for this endpoint, return false
			return false
		}
	}

	// sampling limit per minute exceeded
	s.samplingLock.Lock()
	defer s.samplingLock.Unlock()
	if s.limit <= s.samplingCounter {
		return false
	}

	hasFailedStatus := statusText == Failure
	// sample only failed transaction if OnlyErrors is set to `true` and the transaction summary's status is an error
	if !hasFailedStatus && onlyErrors {
		return false
	}

	// check if transaction is an error and sample it for api-app pair if no other error was found yet
	if hasFailedStatus && s.config.ErrorSamplingEnabled {
		key := FormatApiAppKey(details.APIID, details.SubID)
		if _, exists := s.apiAppErrorSampling[key]; exists {
			return false
		}
		s.apiAppErrorSampling[key] = struct{}{}

		// we don't count the unique combos so we don't add those to the counter
		return true
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
