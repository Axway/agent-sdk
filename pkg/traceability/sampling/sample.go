package sampling

import (
	"fmt"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

const limiterCounter = "LimiterCounter"

// sample - private struct that is used to keep track of the samples being taken
type sample struct {
	config         Sampling
	currentCounts  map[string]int
	counterLock    sync.Mutex
	limiterRunning bool
	minuteLimit    int
	stopChan       chan struct{}
}

func (s *sample) stopLimiter() {
	if !s.limiterRunning {
		return
	}
	defer s.setLimiterRunning(false)
	s.stopChan <- struct{}{}
}

func (s *sample) setLimiterRunning(val bool) {
	s.limiterRunning = val
}

func (s *sample) runLimiter() {
	// wait for the next minute to start
	nextMinute := time.Now().Truncate(time.Minute).Add(time.Minute)
	<-time.NewTimer(time.Until(nextMinute)).C
	s.resetLimiter() // reset the limiter

	// trigger reset every minute now that we are at top of minute
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.resetLimiter()
		case <-s.stopChan:
			return
		}
	}
}

func (s *sample) startLimiter() {
	if s.limiterRunning {
		return
	}

	// get the limit
	s.minuteLimit = agent.GetCentralConfig().GetSamplingPerMinuteLimit()
	defer s.setLimiterRunning(true)

	// init counter limiter
	s.resetLimiter()

	go s.runLimiter()
}

func (s *sample) resetLimiter() {
	s.counterLock.Lock()
	defer s.counterLock.Unlock()
	s.currentCounts[limiterCounter] = 0
}

func (s *sample) checkLimiter(shouldSample bool) bool {
	if !shouldSample {
		return shouldSample
	}

	// no need to lock as this is called by a method that already locks the map
	if s.currentCounts[limiterCounter] == s.minuteLimit {
		return false
	}

	s.currentCounts[limiterCounter]++
	return true
}

// ShouldSampleTransaction - receives the transaction details and returns true to sample it false to not
func (s *sample) ShouldSampleTransaction(details TransactionDetails) bool {
	if !agent.GetCentralConfig().IsSamplingEnabled() {
		s.stopLimiter()
		return false
	}
	s.startLimiter()

	hasFailedStatus := details.Status == "Failure"
	// sample only failed transaction if OnlyErrors is set to `true` and the transaction summary's status is an error
	if !hasFailedStatus && s.config.OnlyErrors {
		return false
	}

	sampleGlobal := s.shouldSampleWithCounter(globalCounter)
	perAPIEnabled := s.config.PerAPI && details.APIID != ""

	if s.config.PerSub && details.SubID != "" {
		apiSample := false
		if perAPIEnabled {
			apiSample = s.shouldSampleWithCounter(details.APIID)
		}
		return s.shouldSampleWithCounter(fmt.Sprintf("%s-%s", details.APIID, details.SubID)) || apiSample
	}

	if perAPIEnabled {
		return s.shouldSampleWithCounter(details.APIID)
	}

	return sampleGlobal
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
	if s.currentCounts[counterName] < s.config.shouldSampleMax {
		shouldSample = true
	}
	s.currentCounts[counterName]++

	// reset the count once we hit 100 * 10^(nb_decimals) messages
	if s.currentCounts[counterName] == s.config.countMax {
		s.currentCounts[counterName] = 0
	}

	// return if we should sample this transaction
	return s.checkLimiter(shouldSample)
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
