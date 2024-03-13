package compliance

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/traceability"
	beatPub "github.com/elastic/beats/v7/libbeat/publisher"
)

const (
	complicanceRetries = "complianceRetry"
)

type EventBatch struct {
	beatPub.Batch

	events        []beatPub.Event
	haveBatchLock bool
}

// AddEvent - adds an event to the batch
func (b *EventBatch) addEvent(event beatPub.Event) {
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
		b.haveBatchLock = true
	}
}

// make sure batch does not unlock multiple times
func (b *EventBatch) batchUnlock() {
	if b.haveBatchLock {
		b.haveBatchLock = false
	}
}

// Events - return the events in the batch
func (b *EventBatch) Events() []beatPub.Event {
	return b.events
}

// ACK - all events have been acked, cleanup the counters
func (b *EventBatch) ACK() {
	b.batchUnlock()
}

// Drop - drop the entire batch
func (b *EventBatch) Drop() {
	b.batchUnlock()
}

// Retry - batch sent for retry, publish again
func (b *EventBatch) Retry() {
	b.retryEvents(b.events)
}

// Cancelled - batch has been cancelled
func (b *EventBatch) Cancelled() {
	b.batchUnlock()
}

func (b *EventBatch) retryEvents(events []beatPub.Event) {
	retryEvents := make([]beatPub.Event, 0)
	for _, event := range events {
		if _, found := event.Content.Meta[complicanceRetries]; !found {
			event.Content.Meta[complicanceRetries] = 0
		}
		count := event.Content.Meta[complicanceRetries].(int)
		newCount := count + 1
		if newCount <= 3 {
			event.Content.Meta[complicanceRetries] = newCount
			retryEvents = append(retryEvents, event)
		}

		// let the metric batch handle its own retries
		if _, found := event.Content.Meta[complicanceRetries]; found {
			event.Content.Meta[complicanceRetries] = 0
		}
	}
	b.events = retryEvents
	b.Publish()
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
func NewEventBatch() *EventBatch {
	return &EventBatch{
		haveBatchLock: false,
	}
}
