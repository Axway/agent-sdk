package metric

import (
	"context"
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/traceability"
	beatPub "github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/rcrowley/go-metrics"
)

// EventBatch - creates a batch of MetricEvents to send to Condor
type EventBatch struct {
	beatPub.Batch
	events        []beatPub.Event
	histograms    map[string]metrics.Histogram
	collector     *collector
	haveBatchLock bool
}

// AddEvent - adds an event to the batch
func (b *EventBatch) AddEvent(event beatPub.Event, histogram metrics.Histogram) {
	b.events = append(b.events, event)
	eventID := event.Content.Meta[metricKey].(string)
	b.histograms[eventID] = histogram
}

// AddEvent - adds an event to the batch
func (b *EventBatch) AddEventWithoutHistogram(event beatPub.Event) {
	b.events = append(b.events, event)
}

// Publish - connects to the traceability clients and sends this batch of events
func (b *EventBatch) Publish() error {
	b.batchLock()
	return b.publish()
}

func (b *EventBatch) publish() error {
	client, err := traceability.GetClient()
	if err != nil {
		b.batchUnlock()
		return err
	}
	err = client.Connect()
	if err != nil {
		b.batchUnlock()
		return err
	}
	go b.logEvents("publishing", b.events)
	err = client.Publish(context.Background(), b)
	if err != nil {
		b.batchUnlock()
		return err
	}
	return nil
}

// make sure batch does not lock multiple times
func (b *EventBatch) batchLock() {
	if !b.haveBatchLock {
		b.collector.batchLock.Lock()
		b.haveBatchLock = true
	}
}

// make sure batch does not unlock multiple times
func (b *EventBatch) batchUnlock() {
	if b.haveBatchLock {
		b.collector.batchLock.Unlock()
		b.haveBatchLock = false
	}
}

// Events - return the events in the batch
func (b *EventBatch) Events() []beatPub.Event {
	return b.events
}

// ACK - all events have been acked, cleanup the counters
func (b *EventBatch) ACK() {
	for _, event := range b.events {
		metric := getMetricFromEvent(event)
		if metric != nil {
			b.collector.logMetric("published", metric)
			b.collector.cleanupMetricCounter(b.histograms[metric.EventID], metric)
		}
	}
	b.collector.metricStartTime = b.collector.metricEndTime
	b.batchUnlock()
}

// Drop - drop the entire batch
func (b *EventBatch) Drop() {
	go b.logEvents("drop called, retrying", b.events)
	b.publish()
}

// Retry - batch sent for retry, publish again
func (b *EventBatch) Retry() {
	go b.logEvents("retrying batch", b.events)
	b.publish()
}

// Cancelled - batch has been cancelled
func (b *EventBatch) Cancelled() {
	go b.logEvents("cancelled called, retrying", b.events)
	b.publish()
}

// RetryEvents - certain events sent to retry
func (b *EventBatch) RetryEvents(events []beatPub.Event) {
	go b.logEvents("retrying", events)
	b.events = events
	b.publish()
}

// CancelledEvents - events have been cancelled
func (b *EventBatch) CancelledEvents(events []beatPub.Event) {
	go b.logEvents("cancelled called, retrying", events)
	b.events = events
	b.publish()
}

// Events - return the events in the batch
func (b *EventBatch) logEvents(status string, events []beatPub.Event) {
	for _, event := range events {
		metric := getMetricFromEvent(event)
		if metric != nil {
			b.collector.logMetric(status, metric)
		}
	}
}

// NewEventBatch - creates a new batch
func NewEventBatch(c *collector) *EventBatch {
	return &EventBatch{
		collector:     c,
		histograms:    make(map[string]metrics.Histogram),
		haveBatchLock: false,
	}
}

func getMetricFromEvent(event beatPub.Event) *APIMetric {
	if data, found := event.Content.Fields[messageKey]; found {
		v4Bytes := data.(string)
		v4Event := make(map[string]interface{})
		err := json.Unmarshal([]byte(v4Bytes), &v4Event)
		if err != nil {
			return nil
		}
		eventType, ok := v4Event["event"]
		if !ok {
			return nil
		}
		if eventType.(string) != metricEvent {
			return nil
		}
		buf, err := json.Marshal(v4Event["data"])
		if err != nil {
			return nil
		}
		metric := &APIMetric{}
		err = json.Unmarshal(buf, metric)
		if err != nil {
			return nil
		}
		return metric
	}
	return nil
}
