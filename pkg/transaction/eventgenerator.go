package transaction

import (
	"encoding/json"
	"fmt"
	"time"

	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// EventGenerator - Create the events to be published to Condor
type EventGenerator interface {
	CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, privateData interface{}) (event beat.Event, err error)
	CheckHealth() error
}

// Generator - Create the events to be published to Condor
type Generator struct {
	tokenRequester *apicauth.PlatformTokenGetter
}

// NewEventGenerator - Create a new event generator
func NewEventGenerator(tokenURL, aud, privKey, pubKey, keyPwd, clientID string, authTimeout time.Duration) EventGenerator {
	return &Generator{
		tokenRequester: apicauth.NewPlatformTokenGetter(privKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout),
	}
}

// CreateEvent - Creates a new event to be sent to Condor
func (e *Generator) CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, privateData interface{}) (event beat.Event, err error) {
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

// CheckHealth -
func (e *Generator) CheckHealth() error {
	_, err := e.tokenRequester.GetToken()
	if err != nil {
		return fmt.Errorf("error trying to get platform token: %s. Check AMPLIFY Central configuration for AUTH_URL, AUTH_REALM, AUTH_CLIENTID, AUTH_PRIVATEKEY, and AUTH_PUBLICKEY", err.Error())
	}
	return nil
}

func (e *Generator) createEventData(message []byte) (eventData map[string]interface{}, err error) {
	fields, err := e.createEventFields()
	if err != nil {
		return nil, err
	}

	eventData = make(map[string]interface{})
	eventData["message"] = string(message)
	eventData["fields"] = fields
	return eventData, err
}

func (e *Generator) createEventFields() (fields map[string]string, err error) {
	var token string
	if token, err = e.tokenRequester.GetToken(); err != nil {
		return
	}
	fields = make(map[string]string)
	fields["axway-target-flow"] = "api-central-v8"
	fields["token"] = token
	return
}
