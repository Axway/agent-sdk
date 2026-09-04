package traceability

import (
	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// eventBatch is a minimal event.Batch implementation that logs ACK/Drop/Retry/
// Cancelled activity, for callers that publish a fixed slice of events and don't
// need custom batch-completion handling (e.g. forwarding to an upstream pipeline).
type eventBatch struct {
	events []event.Event
	logger log.FieldLogger
}

// NewEventBatch builds an event.Batch around a fixed slice of events, for callers
// of Client.Publish that don't already have their own Batch implementation.
func NewEventBatch(events []event.Event) event.Batch {
	return &eventBatch{
		events: events,
		logger: log.NewFieldLogger().WithPackage("sdk.traceability").WithComponent("eventBatch"),
	}
}

func (b *eventBatch) Events() []event.Event          { return b.events }
func (b *eventBatch) SetEvents(events []event.Event) { b.events = events }

func (b *eventBatch) ACK() {
	b.logger.WithField("count", len(b.events)).Info("published events")
}

func (b *eventBatch) Drop() {
	b.logger.WithField("count", len(b.events)).Warn("dropped events")
}

func (b *eventBatch) Retry() {
	b.logger.WithField("count", len(b.events)).Warn("retrying events")
}

func (b *eventBatch) Cancelled() {
	b.logger.WithField("count", len(b.events)).Warn("cancelled events")
}

func (b *eventBatch) RetryEvents(events []event.Event) {
	b.logger.WithField("count", len(events)).Warn("retrying events")
}

func (b *eventBatch) CancelledEvents(events []event.Event) {
	b.logger.WithField("count", len(events)).Warn("cancelled events")
}
