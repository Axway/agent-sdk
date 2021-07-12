package metric

import (
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/traceability"
	beatPub "github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/rcrowley/go-metrics"
)

// EventBatch - creates a batch of MetricEvents to send to Condor
type EventBatch struct {
	beatPub.Batch

	events     []beatPub.Event
	histograms map[string]metrics.Histogram
	collector  *collector
}

// AddEvent - adds an event to the batch
func (b *EventBatch) AddEvent(event beatPub.Event, histogram metrics.Histogram) {
	b.events = append(b.events, event)
	eventID := event.Content.Meta[metricKey].(string)
	b.histograms[eventID] = histogram
}

// Publish - connects to the traceability clients and sends this batch of events
func (b *EventBatch) Publish() error {
	b.collector.batchLock.Lock()

	return b.publish()
}

func (b *EventBatch) publish() error {
	client, err := traceability.GetClient()
	if err != nil {
		b.collector.batchLock.Unlock()
		return err
	}
	err = client.Connect()
	if err != nil {
		b.collector.batchLock.Unlock()
		return err
	}
	client.Publish(b)
	return nil
}

// Events - return the events in the batch
func (b *EventBatch) Events() []beatPub.Event {
	return b.events
}

// ACK - all events have been acked, cleanup the counters
func (b *EventBatch) ACK() {
	for _, event := range b.events {
		var v4Event V4Event
		if data, found := event.Content.Fields[messageKey]; found {
			v4Bytes := data.(string)
			err := json.Unmarshal([]byte(v4Bytes), &v4Event)
			if err != nil {
				continue
			}
			eventID := event.Content.Meta[metricKey].(string)
			b.collector.cleanupMetricCounter(b.histograms[eventID], v4Event)
		}
	}
	b.collector.batchLock.Unlock()
}

// Drop - drop the entire batch
func (b *EventBatch) Drop() {
	b.collector.batchLock.Unlock()
}

// Retry - batch sent for retry, publish again
func (b *EventBatch) Retry() {
	b.retryEvents(b.events)
}

// Cancelled - batch has been cancelled
func (b *EventBatch) Cancelled() {
	b.collector.batchLock.Unlock()
}

func (b *EventBatch) retryEvents(events []beatPub.Event) {
	retryEvents := make([]beatPub.Event, 0)
	for _, event := range b.events {
		if _, found := event.Content.Meta[metricRetries]; !found {
			event.Content.Meta[metricRetries] = 0
		}
		count := event.Content.Meta[metricRetries].(int)
		newCount := count + 1
		if newCount <= 3 {
			event.Content.Meta[metricRetries] = newCount
			retryEvents = append(retryEvents, event)
		}

		// let the metric batch handle its own retries
		if _, found := event.Content.Meta[retries]; found {
			event.Content.Meta[retries] = 0
		}
	}
	b.events = retryEvents
	b.publish()
}

// RetryEvents - certain events sent to retry
func (b *EventBatch) RetryEvents(events []beatPub.Event) {
	b.retryEvents(events)
}

// CancelledEvents - events have been cancelled
func (b *EventBatch) CancelledEvents(events []beatPub.Event) {
	b.events = events
	b.publish()
}

// NewEventBatch - creates a new batch
func NewEventBatch(c *collector) *EventBatch {
	return &EventBatch{
		collector:  c,
		histograms: make(map[string]metrics.Histogram),
	}
}
