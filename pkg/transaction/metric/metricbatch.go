package metric

import (
	"encoding/json"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
	beatPub "github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/rcrowley/go-metrics"
)

const (
	messageKey = "message"
	metricKey  = "metric"
)

// CondorMetricEvent - the condor event format to send metric data
type CondorMetricEvent struct {
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	Timestamp time.Time              `json:"-"`
	ID        string                 `json:"-"`
}

// AddCondorMetricEventToBatch - creates the condor metric event and adds to the batch
func AddCondorMetricEventToBatch(metricEvent V4Event, batch *EventBatch, histogram metrics.Histogram) error {
	metricData, _ := json.Marshal(metricEvent)

	cme := &CondorMetricEvent{
		Message:   string(metricData),
		Fields:    make(map[string]interface{}),
		Timestamp: metricEvent.Data.StartTime,
		ID:        metricEvent.ID,
	}
	event, err := cme.CreateEvent()
	log.Tracef("%+v", event)
	if err != nil {
		return err
	}
	batch.AddEvent(event, histogram)
	return nil
}

// CreateEvent - creates the beat event to add to the batch
func (c *CondorMetricEvent) CreateEvent() (beatPub.Event, error) {
	// Get the event token
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		return beatPub.Event{}, err
	}
	c.Fields["token"] = token

	// convert the CondorMetricEvent to json then to map[string]interface{}
	cmeJSON, err := json.Marshal(c)
	log.Tracef(string(cmeJSON))
	if err != nil {
		return beatPub.Event{}, err
	}

	var fieldsData map[string]interface{}
	err = json.Unmarshal(cmeJSON, &fieldsData)
	if err != nil {
		return beatPub.Event{}, err
	}

	return beatPub.Event{
		Content: beat.Event{
			Timestamp: c.Timestamp,
			Meta: map[string]interface{}{
				metricKey:          c.ID,
				sampling.SampleKey: true, // All metric events should be sent
			},
			Fields: fieldsData,
		},
		Flags: beatPub.GuaranteedSend,
	}, nil
}

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
	client, err := traceability.GetClient()
	if err != nil {
		return err
	}
	client.Connect()
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
			b.collector.cleanupMetricCounter(b.histograms[eventID], v4Event.Data.API.ID, v4Event.Data.StatusCode)
		}
	}
}

// Drop - drop the entire batch
func (b *EventBatch) Drop() {}

// Retry - batch sent for retry, publish again
func (b *EventBatch) Retry() {
	b.Publish()
}

// Cancelled - batch has been cancelled
func (b *EventBatch) Cancelled() {}

// RetryEvents - certain events sent to retry
func (b *EventBatch) RetryEvents(events []beatPub.Event) {
	b.events = events
	b.Publish()
}

// CancelledEvents - events have been cancelled
func (b *EventBatch) CancelledEvents(events []beatPub.Event) {}

// NewEventBatch - creates a new batch
func NewEventBatch(c *collector) *EventBatch {
	return &EventBatch{
		collector:  c,
		histograms: make(map[string]metrics.Histogram),
	}
}
