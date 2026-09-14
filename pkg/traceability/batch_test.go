package traceability

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/stretchr/testify/assert"
)

func TestNewEventBatchEvents(t *testing.T) {
	events := []event.Event{{}, {}}
	batch := NewEventBatch(events)

	assert.Equal(t, events, batch.Events())

	replacement := []event.Event{{}}
	batch.SetEvents(replacement)
	assert.Equal(t, replacement, batch.Events())
}

func TestNewEventBatchCompletionCallbacksDoNotPanic(t *testing.T) {
	batch := NewEventBatch([]event.Event{{}})

	assert.NotPanics(t, batch.ACK)
	assert.NotPanics(t, batch.Drop)
	assert.NotPanics(t, batch.Retry)
	assert.NotPanics(t, batch.Cancelled)
	assert.NotPanics(t, func() { batch.RetryEvents([]event.Event{{}}) })
	assert.NotPanics(t, func() { batch.CancelledEvents([]event.Event{{}}) })
}
