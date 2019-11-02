package transaction

import (
	"encoding/json"
	"time"

	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// EventGenerator - Create the events to be published to Condor
type EventGenerator struct {
	tokenRequester *apicauth.PlatformTokenGetter
}

// NewEventGenerator - Create a new event generator
func NewEventGenerator(tokenURL, aud, privKey, pubKey, keyPwd, clientID string, authTimeout time.Duration) *EventGenerator {
	return &EventGenerator{
		tokenRequester: apicauth.NewPlatformTokenGetter(privKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout),
	}
}

// CreateEvent - Creates a new event to be sent to Condor
func (e *EventGenerator) CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, privateData interface{}) (event beat.Event, err error) {
	serializedLogEvent, err := json.Marshal(logEvent)
	if err != nil {
		return
	}

	eventData, err := e.createEventData(serializedLogEvent)
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

func (e *EventGenerator) createEventData(message []byte) (eventData map[string]interface{}, err error) {
	fields, err := e.createEventFields()
	if err != nil {
		return nil, err
	}

	eventData = make(map[string]interface{})
	eventData["message"] = string(message)
	eventData["fields"] = fields
	return eventData, err
}

func (e *EventGenerator) createEventFields() (fields map[string]interface{}, err error) {
	var token string
	if token, err = e.tokenRequester.GetToken(); err != nil {
		return
	}
	fields = make(map[string]interface{})
	fields["axway-target-flow"] = "api-central-v8"
	fields["token"] = token
	return
}
