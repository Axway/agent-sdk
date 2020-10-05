package transaction

import (
	"encoding/json"
	"time"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/agent"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/traceability"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"
	hc "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/healthcheck"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

// EventGenerator - Create the events to be published to Condor
type EventGenerator interface {
	CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, fields common.MapStr, privateData interface{}) (event beat.Event, err error)
}

// Generator - Create the events to be published to Condor
type Generator struct {
	shouldAddFields bool
}

// NewEventGenerator - Create a new event generator
func NewEventGenerator() EventGenerator {
	eventGen := &Generator{
		shouldAddFields: !traceability.IsHTTPTransport(),
	}
	hc.RegisterHealthcheck("Event Generator", "eventgen", eventGen.healthcheck)
	return eventGen
}

// CreateEvent - Creates a new event to be sent to Condor
func (e *Generator) CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) (event beat.Event, err error) {
	serializedLogEvent, err := json.Marshal(logEvent)
	if err != nil {
		return
	}

	eventData, err := e.createEventData(serializedLogEvent, eventFields)
	if err != nil {
		return
	}

	event = beat.Event{
		Timestamp: eventTime,
		Meta:      metaData,
		Private:   privateData,
		Fields:    eventData,
	}

	return
}

// healthcheck -
func (e *Generator) healthcheck(name string) (status *hc.Status) {
	// Create the default return
	status = &hc.Status{
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

	return
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
