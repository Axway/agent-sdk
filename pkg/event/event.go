// Package event replaces libbeat's beat.Event, common.MapStr, publisher.Event, and
// publisher.Batch.
package event

import "time"

// MapStr is a flat string-keyed metadata/fields map, replacing libbeat's common.MapStr.
type MapStr map[string]interface{}

// Event is a single event to publish, replacing libbeat's beat.Event/publisher.Event.
type Event struct {
	Timestamp time.Time
	Meta      MapStr
	Fields    MapStr
	Private   interface{}
}

// Batch is a collection of events to be published as a unit, replacing libbeat's
// publisher.Batch interface.
type Batch interface {
	Events() []Event
	SetEvents(events []Event)
	ACK()
	Drop()
	Retry()
	RetryEvents(events []Event)
	Cancelled()
	CancelledEvents(events []Event)
}
