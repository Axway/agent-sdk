package traceability

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/google/uuid"
)

type pingBatch struct {
	orgID string
	acked bool
}

// newPingBatch creates a new empty batch.
func newPingBatch(orgID string) *pingBatch {
	return &pingBatch{
		orgID: orgID,
		acked: false,
	}
}

// Events returns an empty slice of events.
func (b *pingBatch) Events() []publisher.Event {
	return []publisher.Event{
		{
			Content: beat.Event{
				Fields: map[string]interface{}{
					"fields": map[string]interface{}{},
					"message": map[string]interface{}{
						"data": createV4PingEvent(b.orgID),
					},
				},
				Meta: map[string]interface{}{
					"metric": true,
				},
			},
		},
	}
}

// ACK is a no-op.
func (b *pingBatch) ACK() {
	b.acked = true
}

// Drop is a no-op.
func (b *pingBatch) Drop() {}

// Retry is a no-op.
func (b *pingBatch) Retry() {}

// RetryEvents is a no-op.
func (b *pingBatch) RetryEvents(events []publisher.Event) {}

// Cancelled is a no-op.
func (b *pingBatch) Cancelled() {}

// CancelledEvents is a no-op.
func (b *pingBatch) CancelledEvents(events []publisher.Event) {}

// String returns an empty string.
func (b *pingBatch) String() string {
	return ""
}

const (
	pingEvent = "analytics.ping"
)

// v4PingEvent - represents V7 event
type v4PingEvent struct {
	ID        string                 `json:"id"`
	Timestamp int64                  `json:"timestamp"`
	Event     string                 `json:"event"`
	Version   string                 `json:"version"`
	Data      map[string]interface{} `json:"data"`
}

func createV4PingEvent(orgID string) v4PingEvent {
	id := uuid.NewString()
	now := time.Now()
	return v4PingEvent{
		ID:        id,
		Timestamp: now.Unix(),
		Event:     pingEvent,
		Version:   "4",
		Data:      map[string]interface{}{},
	}
}
