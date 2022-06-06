package transaction

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

// EventGenerator - Create the events to be published to Condor
type EventGenerator interface {
	CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, fields common.MapStr, privateData interface{}) (event beat.Event, err error) // DEPRECATED
	CreateEvents(summaryEvent LogEvent, detailEvents []LogEvent, eventTime time.Time, metaData common.MapStr, fields common.MapStr, privateData interface{}) (events []beat.Event, err error)
	SetUseTrafficForAggregation(useTrafficForAggregation bool)
}

// Generator - Create the events to be published to Condor
type Generator struct {
	shouldAddFields                bool
	shouldUseTrafficForAggregation bool
	collector                      metric.Collector
}

// NewEventGenerator - Create a new event generator
func NewEventGenerator() EventGenerator {
	eventGen := &Generator{
		shouldAddFields:                !traceability.IsHTTPTransport(),
		shouldUseTrafficForAggregation: true,
	}
	hc.RegisterHealthcheck("Event Generator", "eventgen", eventGen.healthcheck)

	return eventGen
}

// SetUseTrafficForAggregation - set the flag to use traffic events for aggregation.
func (e *Generator) SetUseTrafficForAggregation(useTrafficForAggregation bool) {
	e.shouldUseTrafficForAggregation = useTrafficForAggregation
}

// CreateEvent - Creates a new event to be sent to Amplify Observability
func (e *Generator) CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) (beat.Event, error) {
	// DEPRECATED
	log.DeprecationWarningReplace("CreateEvent", "CreateEvents")

	// if CreateEvent is being used, sampling will not work, so all events need to be sent
	if metaData == nil {
		metaData = common.MapStr{}
	}
	metaData.Put(sampling.SampleKey, true)

	if logEvent.TransactionSummary != nil {
		e.trackMetrics(logEvent, 0)
	}

	return e.createEvent(logEvent, eventTime, metaData, eventFields, privateData)
}

func (e *Generator) trackMetrics(summaryEvent LogEvent, bytes int64) {
	if e.shouldUseTrafficForAggregation {
		apiDetails := metric.APIDetails{
			ID:       summaryEvent.TransactionSummary.Proxy.ID,
			Name:     summaryEvent.TransactionSummary.Proxy.Name,
			Revision: summaryEvent.TransactionSummary.Proxy.Revision,
			Stage:    summaryEvent.TransactionSummary.Proxy.Stage,
		}

		if summaryEvent.TransactionSummary.Team != nil {
			apiDetails.TeamID = summaryEvent.TransactionSummary.Team.ID
		}

		statusCode := summaryEvent.TransactionSummary.StatusDetail
		duration := summaryEvent.TransactionSummary.Duration
		appDetails := metric.AppDetails{}
		if summaryEvent.TransactionSummary.Application != nil {
			appDetails.Name = summaryEvent.TransactionSummary.Application.Name
			appDetails.ID = strings.TrimLeft(summaryEvent.TransactionSummary.Application.ID, SummaryEventApplicationIDPrefix)
		}

		collector := metric.GetMetricCollector()
		if collector != nil {
			metricDetail := metric.Detail{
				APIDetails: apiDetails,
				StatusCode: statusCode,
				Duration:   int64(duration),
				Bytes:      bytes,
				AppDetails: appDetails,
			}
			collector.AddMetricDetail(metricDetail)
		}
	}
}

// CreateEvent - Creates a new event to be sent to Amplify Observability
func (e *Generator) createEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) (beat.Event, error) {
	event := beat.Event{}
	serializedLogEvent, err := json.Marshal(logEvent)
	if err != nil {
		return event, err
	}

	eventData := eventFields
	// No need to get the other field data if not being sampled
	if sampled, found := metaData[sampling.SampleKey]; found && sampled.(bool) {
		eventData, err = e.createEventData(serializedLogEvent, eventFields)
	}
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

// CreateEvents - Creates new events to be sent to Amplify Observability
func (e *Generator) CreateEvents(summaryEvent LogEvent, detailEvents []LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) ([]beat.Event, error) {
	events := make([]beat.Event, 0)

	// See if the uri is in the api exceptions list
	if e.isInAPIExceptionsList(detailEvents) {
		log.Debug("Found api path in traceability api exceptions list.  Ignore transaction event")
		return events, nil
	}
	if util.IsNotTest() {
		summaryEvent.TransactionSummary.ConsumerDetails = e.getConsumerInfo(summaryEvent)
	}

	//if no summary is sent then prepare the array of TransactionEvents for publishing
	if summaryEvent == (LogEvent{}) {

		for _, event := range detailEvents {
			if metaData == nil {
				metaData = common.MapStr{}
			}
			metaData.Put(sampling.SampleKey, true)
			newEvent, err := e.createEvent(event, eventTime, metaData, eventFields, privateData)
			if err == nil {
				events = append(events, newEvent)
			}
		}

		return events, nil

	}

	// Add this to sample or not
	shouldSample, err := sampling.ShouldSampleTransaction(e.createSamplingTransactionDetails(summaryEvent))
	if err != nil {
		return events, err
	}
	if shouldSample {
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

	bytes := 0
	if len(detailEvents) > 0 {
		if httpEvent, ok := detailEvents[0].TransactionEvent.Protocol.(*Protocol); ok {
			bytes = httpEvent.BytesSent
		}
	}
	e.trackMetrics(summaryEvent, int64(bytes))

	return events, nil
}

// getConsumerInfo - get the consumer information to add to transaction event
func (e *Generator) getConsumerInfo(summaryEvent LogEvent) *ConsumerDetails {
	cacheManager := agent.GetCacheManager()
	appName := ""

	if summaryEvent.TransactionSummary.Application != nil {
		appName = summaryEvent.TransactionSummary.Application.Name
	}

	// get proxy information
	apiID := summaryEvent.TransactionSummary.Proxy.ID
	stage := summaryEvent.TransactionSummary.Proxy.Stage

	managedApp := cacheManager.GetManagedApplicationByName(appName)
	accessRequest := transutil.GetAccessRequest(cacheManager, managedApp, apiID, stage)

	// get subscription info
	subscription := transutil.GetSubscription(cacheManager, accessRequest)
	subscriptionID := transutil.GetSubscriptionID(subscription)
	subscriptionName := subscription.Name

	application := &Application{
		ID:   unknown,
		Name: unknown,
	}

	consumerOrgID := unknown

	if managedApp != nil {
		appID, appName := transutil.GetConsumerApplication(managedApp)
		application.ID = appID
		application.Name = appName
		consumerOrgID := transutil.GetConsumerOrgID(managedApp)
		if consumerOrgID == "" {
			consumerOrgID = transutil.GetConsumerOrgIDFromSubscription(subscription)
		}
	}

	// Update consumer details with Org, Application and Subscription
	consumerDetails := &ConsumerDetails{
		OrgID:       consumerOrgID,
		Application: application,
		Subscription: &Subscription{
			ID:   subscriptionID,
			Name: subscriptionName,
		},
	}

	return consumerDetails
}

// createSamplingTransactionDetails -
func (e *Generator) createSamplingTransactionDetails(summaryEvent LogEvent) sampling.TransactionDetails {
	var status string
	var apiID string

	if summaryEvent.TransactionSummary != nil {
		status = summaryEvent.TransactionSummary.Status
		if summaryEvent.TransactionSummary.Proxy != nil {
			apiID = summaryEvent.TransactionSummary.Proxy.ID
		}
	}

	return sampling.TransactionDetails{
		Status: status,
		APIID:  apiID,
	}
}

// Validate APIs in the traceability exceptions list
func (e *Generator) isInAPIExceptionsList(logEvents []LogEvent) bool {

	// Sanity check.
	if len(logEvents) == 0 {
		return false
	}

	// Check first leg for URI.  Use the raw value before redaction happens
	uriRaw := ""

	if httpEvent, ok := logEvents[0].TransactionEvent.Protocol.(*Protocol); ok {
		uriRaw = httpEvent.uriRaw
	}

	// Get the api exceptions list
	return traceability.ShouldIgnoreEvent(uriRaw)

}

// healthcheck -
func (e *Generator) healthcheck(name string) *hc.Status {
	// Create the default return
	status := &hc.Status{
		Result:  hc.OK,
		Details: "",
	}

	if percentage, _ := sampling.GetGlobalSamplingPercentage(); percentage == 0 {
		// Do not execute the healthcheck when sampling is 0
		return status
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
	fields[traceability.FlowHeader] = traceability.TransactionFlow
	return
}
