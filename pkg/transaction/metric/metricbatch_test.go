package metric

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/event"
)

// compile-time proof that EventBatch satisfies the agent-sdk-owned event.Batch
// interface, replacing libbeat's publisher.Batch.
var _ event.Batch = (*EventBatch)(nil)

func newTestEventBatch() (*EventBatch, event.Event) {
	collector := &collector{batchLock: &sync.Mutex{}}
	batch := NewEventBatch(collector)
	evt := event.Event{Timestamp: time.Now(), Meta: event.MapStr{metricKey: "id-1"}, Fields: event.MapStr{"message": "m"}}
	return batch, evt
}

func TestEventBatchAddAndSetEvents(t *testing.T) {
	batch, evt := newTestEventBatch()

	batch.AddEvent(evt, nil, nil)
	assert.Equal(t, []event.Event{evt}, batch.Events())

	batch.SetEvents([]event.Event{})
	assert.Empty(t, batch.Events())

	batch.AddEventWithoutHistogram(evt)
	assert.Equal(t, []event.Event{evt}, batch.Events())
}

func TestEventBatchUnlocksAfterTerminalMethods(t *testing.T) {
	tests := map[string]struct {
		call func(b *EventBatch)
	}{
		"ACK":       {call: func(b *EventBatch) { b.ACK() }},
		"Retry":     {call: func(b *EventBatch) { b.Retry() }},
		"Drop":      {call: func(b *EventBatch) { b.Drop() }},
		"Cancelled": {call: func(b *EventBatch) { b.Cancelled() }},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			batch, evt := newTestEventBatch()
			batch.AddEvent(evt, nil, nil)

			batch.batchLock()
			tc.call(batch)
			assert.False(t, batch.haveBatchLock)
		})
	}
}

func TestCondorMetricEventCreateEventHasNoGuaranteedSendOrCache(t *testing.T) {
	evt := event.Event{}
	assert.Nil(t, evt.Private)
}
