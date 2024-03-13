package compliance

import (
	"encoding/json"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
)

const (
	runtimeComplianceEvent = "api.runtime.compliance.result"
)

func createV4Event(orgGUID string, v4data metric.V4Data) metric.V4Event {
	return metric.V4Event{
		ID:        v4data.GetEventID(),
		Timestamp: time.Now().UnixMilli(),
		Event:     runtimeComplianceEvent,
		App:       orgGUID,
		Version:   "4",
		Distribution: &metric.V4EventDistribution{
			Environment: agent.GetCentralConfig().GetEnvironmentID(),
			Version:     "1",
		},
		Data: v4data,
	}
}

func AddEventToBatch(metricEvent metric.V4Event, batch *EventBatch) error {
	metricData, _ := json.Marshal(metricEvent)

	cme := &metric.CondorMetricEvent{
		Message:   string(metricData),
		Fields:    make(map[string]interface{}),
		Timestamp: metricEvent.Data.GetStartTime(),
		ID:        metricEvent.ID,
	}
	event, err := cme.CreateEvent()
	if err != nil {
		return err
	}
	batch.addEvent(event)
	return nil
}
