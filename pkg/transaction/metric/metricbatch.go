package metric

import (
	"context"
	"encoding/json"

	"github.com/Axway/agent-sdk/pkg/traceability"
	beatPub "github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/rcrowley/go-metrics"
)

const cancelMsg = "event cancelled, counts added at next publish"

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
		go b.logEvents("rejected", b.events)
		b.batchUnlock()
		return err
	}
	err = client.Connect()
	if err != nil {
		go b.logEvents("rejected", b.events)
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

// ACK - all events have been acknowledgeded, cleanup the counters
func (b *EventBatch) ACK() {
	b.ackEvents(b.events)
	b.collector.metricStartTime = b.collector.metricEndTime
	b.batchUnlock()
}

func (b *EventBatch) eventsNotAcked(events []beatPub.Event) {
	go b.logEvents(cancelMsg, events)
	b.batchUnlock()
}

// Drop - drop the entire batch
func (b *EventBatch) Drop() {
	b.eventsNotAcked(b.events)
}

// Retry - batch sent for retry, publish again
func (b *EventBatch) Retry() {
	b.eventsNotAcked(b.events)
}

// Cancelled - batch has been cancelled
func (b *EventBatch) Cancelled() {
	b.eventsNotAcked(b.events)
}

// RetryEvents - certain events sent to retry
func (b *EventBatch) RetryEvents(events []beatPub.Event) {
	b.ackEvents(getEventsToAck(events, b.events))
	b.eventsNotAcked(events)
}

// CancelledEvents - events have been cancelled
func (b *EventBatch) CancelledEvents(events []beatPub.Event) {
	b.ackEvents(getEventsToAck(events, b.events))
	b.eventsNotAcked(events)
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

func (b *EventBatch) ackEvents(events []beatPub.Event) {
	for _, event := range events {
		metric := getMetricFromEvent(event)
		if metric == nil {
			continue
		}
		b.collector.logMetric("published", metric)
		histogram, found := b.histograms[metric.EventID]
		if !found {
			b.collector.metricLogger.WithField("eventID", metric.EventID).Warn("could not clean cached metric")
			continue
		}
		b.collector.cleanupMetricCounter(histogram, metric)
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

func getEventsToAck(retryEvents []beatPub.Event, events []beatPub.Event) []beatPub.Event {
	ackEvents := make([]beatPub.Event, 0)
	for _, e := range events {
		eID := ""
		if m := getMetricFromEvent(e); m != nil {
			eID = m.EventID
		}
		if eID == "" {
			continue
		}
		found := false
		for _, rE := range retryEvents {
			rEID := ""
			if m := getMetricFromEvent(rE); m != nil {
				rEID = m.EventID
			}
			if rEID == eID {
				found = true
				break
			}
		}
		if !found {
			ackEvents = append(ackEvents, e)
		}
	}
	return ackEvents
}

func getMetricFromEvent(event beatPub.Event) *APIMetric {
	if data, found := event.Content.Fields[messageKey]; found {
		v4Bytes := data.(string)
		v4Event := make(map[string]interface{})
		err := json.Unmarshal([]byte(v4Bytes), &v4Event)
		if err != nil {
			return nil
		}
		eventID, ok := v4Event["id"]
		if !ok {
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
		metric.EventID = eventID.(string)
		return metric
	}
	return nil
}
