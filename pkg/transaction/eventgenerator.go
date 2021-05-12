package transaction

import (
	"encoding/json"
	"flag"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

// EventGenerator - Create the events to be published to Condor
type EventGenerator interface {
	CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, fields common.MapStr, privateData interface{}) (event beat.Event, err error)
	CreateTransactionEvents(summaryEvent LogEvent, detailEvents []LogEvent, eventTime time.Time, metaData common.MapStr, fields common.MapStr, privateData interface{}) (events []beat.Event, err error)
}

// Generator - Create the events to be published to Condor
type Generator struct {
	shouldAddFields bool
	collector       metric.Collector
}

// NewEventGenerator - Create a new event generator
func NewEventGenerator() EventGenerator {
	eventGen := &Generator{
		shouldAddFields: !traceability.IsHTTPTransport(),
	}
	hc.RegisterHealthcheck("Event Generator", "eventgen", eventGen.healthcheck)
	if flag.Lookup("test.v") == nil {
		metricEventChannel := make(chan interface{})
		eventGen.collector = metric.NewMetricCollector(metricEventChannel)
		metric.NewMetricPublisher(metricEventChannel)
	}
	return eventGen
}

// CreateEvent - Creates a new event to be sent to Condor
func (e *Generator) CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) (beat.Event, error) {
	log.Warnf("%s is deprecated, please start using %s", "CreateEvent", "CreateTransactionEvents")
	return e.createEvent(logEvent, eventTime, metaData, eventFields, privateData)
}

// CreateEvent - Creates a new event to be sent to Condor
func (e *Generator) createEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) (beat.Event, error) {
	event := beat.Event{}
	serializedLogEvent, err := json.Marshal(logEvent)
	if err != nil {
		return event, err
	}
	if logEvent.TransactionSummary != nil {
		apiID := logEvent.TransactionSummary.Proxy.ID
		apiName := logEvent.TransactionSummary.Proxy.Name
		statusCode := logEvent.TransactionSummary.StatusDetail
		duration := logEvent.TransactionSummary.Duration
		appName := ""
		if logEvent.TransactionSummary.Application != nil {
			appName = logEvent.TransactionSummary.Application.Name
		}
		teamName := ""
		if logEvent.TransactionSummary.Team != nil {
			teamName = logEvent.TransactionSummary.Team.ID
		}
		if e.collector != nil {
			e.collector.AddMetric(apiID, apiName, statusCode, int64(duration), appName, teamName)
		}
	}

	eventData, err := e.createEventData(serializedLogEvent, eventFields)
	if err != nil {
		return event, err
	}

	return beat.Event{
		Timestamp: eventTime,
		Meta:      metaData,
		Private:   privateData,
		Fields:    eventData,
	}, nil
}

// CreateTransactionEvents - Creates new events to be sent to Condor
func (e *Generator) CreateTransactionEvents(summaryEvent LogEvent, detailEvents []LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) ([]beat.Event, error) {
	events := make([]beat.Event, 0)

	// Add this to sample or not
	if sampling.ShouldSampleTransaction(e.createSamplingTransactionDetails(summaryEvent)) {
		if metaData == nil {
			metaData = common.MapStr{}
		}
		metaData.Put(sampling.SampleKey, true)
	}

	newEvent, err := e.createEvent(summaryEvent, eventTime, metaData, eventFields, privateData)
	if err != nil {
		return events, err
	}
	events = append(events, newEvent)
	for _, event := range detailEvents {
		newEvent, err := e.createEvent(event, eventTime, metaData, eventFields, privateData)
		if err == nil {
			events = append(events, newEvent)
		}
	}
	return events, nil
}

// createSamplingTransactionDetails -
func (e *Generator) createSamplingTransactionDetails(summaryEvent LogEvent) sampling.TransactionDetails {
	return sampling.TransactionDetails{
		Status: summaryEvent.TransactionSummary.Status,
	}
}

// healthcheck -
func (e *Generator) healthcheck(name string) *hc.Status {
	// Create the default return
	status := &hc.Status{
		Result:  hc.OK,
		Details: "",
	}

	_, err := agent.GetCentralAuthToken()
	if err != nil {
		status = &hc.Status{
			Result:  hc.FAIL,
			Details: errors.Wrap(apic.ErrAuthenticationCall, err.Error()).Error(),
		}
	}

	return status
}

func (e *Generator) createEventData(message []byte, eventFields common.MapStr) (eventData map[string]interface{}, err error) {
	eventData = make(map[string]interface{})
	// Copy event fields if specified
	if eventFields != nil && len(eventFields) > 0 {
		for key, value := range eventFields {
			// Ignore message field as it gets added with this method
			if key != "message" {
				eventData[key] = value
			}
		}
	}

	eventData["message"] = string(message)
	if e.shouldAddFields {
		fields, err := e.createEventFields()
		if err != nil {
			return nil, err
		}
		eventData["fields"] = fields
	}
	return eventData, err
}

func (e *Generator) createEventFields() (fields map[string]string, err error) {
	fields = make(map[string]string)
	var token string
	if token, err = agent.GetCentralAuthToken(); err != nil {
		return
	}
	fields["token"] = token
	fields["axway-target-flow"] = "api-central-v8"
	return
}
