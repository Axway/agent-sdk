package metric

import (
	"encoding/json"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/util/log"
	metrics "github.com/rcrowley/go-metrics"
)

// CondorMetricEvent - the condor event format to send metric data
type CondorMetricEvent struct {
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	Timestamp time.Time              `json:"-"`
	ID        string                 `json:"-"`
}

// AddCondorMetricEventToBatch - creates the condor metric event and adds to the batch
func AddCondorMetricEventToBatch(metricEvent V4Event, batch *EventBatch, histogram metrics.Histogram, counters map[string]metrics.Counter) error {
	metricData, _ := json.Marshal(metricEvent)

	cme := &CondorMetricEvent{
		Message:   string(metricData),
		Fields:    make(map[string]interface{}),
		Timestamp: metricEvent.Data.GetStartTime(),
		ID:        metricEvent.ID,
	}
	evt, err := cme.CreateEvent()
	if err != nil {
		return err
	}
	batch.AddEvent(evt, histogram, counters)
	return nil
}

// CreateEvent - creates the event to add to the batch
func (c *CondorMetricEvent) CreateEvent() (event.Event, error) {
	// Get the event token
	token, err := agent.GetCentralAuthToken()
	if err != nil {
		return event.Event{}, err
	}
	c.Fields["token"] = token
	c.Fields[traceability.FlowHeader] = metricFlow

	// convert the CondorMetricEvent to json then to map[string]interface{}
	cmeJSON, err := json.Marshal(c)
	if err != nil {
		return event.Event{}, err
	}

	var fieldsData map[string]interface{}
	err = json.Unmarshal(cmeJSON, &fieldsData)
	if err != nil {
		return event.Event{}, err
	}

	evt := event.Event{
		Timestamp: c.Timestamp,
		Meta: event.MapStr{
			metricKey:          c.ID,
			sampling.SampleKey: true, // All metric events should be sent
		},
		Fields: fieldsData,
	}
	log.Tracef("Created Metric Event: %+v", evt)
	return evt, nil
}
