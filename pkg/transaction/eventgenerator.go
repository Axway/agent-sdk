package transaction

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	cv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
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
	logger                         log.FieldLogger
}

// NewEventGenerator - Create a new event generator
func NewEventGenerator() EventGenerator {
	logger := log.NewFieldLogger().
		WithPackage("sdk.transaction.eventgenerator").
		WithComponent("eventgenerator")
	eventGen := &Generator{
		shouldAddFields:                !traceability.IsHTTPTransport(),
		shouldUseTrafficForAggregation: true,
		logger:                         logger,
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
		e.logger.Debug("Found api path in traceability api exceptions list.  Ignore transaction event")
		return events, nil
	}

	if summaryEvent.TransactionSummary != nil {
		txnSummary := e.updateTxnSummaryByAccessRequest(summaryEvent)
		if txnSummary != nil {
			summaryEvent.TransactionSummary = txnSummary
		}
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

// updateTxnSummaryByAccessRequest - get the consumer information to add to transaction event.  If we don't have any
// 		information we need to get the consumer information, then we just return nil
func (e *Generator) updateTxnSummaryByAccessRequest(summaryEvent LogEvent) *Summary {
	cacheManager := agent.GetCacheManager()
	appName := unknown
	apiID := ""
	stage := ""

	if summaryEvent.TransactionSummary.Application != nil {
		appName = summaryEvent.TransactionSummary.Application.Name
		e.logger.
			WithField("appName", appName).
			Trace("transaction summary application name")
	}

	// get proxy information
	if summaryEvent.TransactionSummary.Proxy == nil {
		e.logger.Debug("proxy information is not available, no consumer information attached")
		return nil
	}
	apiID = summaryEvent.TransactionSummary.Proxy.ID
	stage = summaryEvent.TransactionSummary.Proxy.Stage
	e.logger.
		WithField("apiID", apiID).
		WithField("stage", stage).
		Trace("transaction summary proxy information")

	// get the managed application
	managedApp := cacheManager.GetManagedApplicationByName(appName)
	if managedApp == nil {
		e.logger.
			WithField("appName", appName).
			Debug("could not get managed application by name, no consumer information attached")
		return nil
	}
	e.logger.
		WithField("appName", appName).
		WithField("managed app name", managedApp.Name).
		Trace("managed application info")

	// get the access request
	accessRequest := transutil.GetAccessRequest(cacheManager, managedApp, apiID, stage)
	if accessRequest == nil {
		e.logger.
			Debug("could not get access request, no consumer information attached")
		return nil
	}
	e.logger.
		WithField("managed app name", managedApp.Name).
		WithField("apiID", apiID).
		WithField("stage", stage).
		WithField("access request name", accessRequest.Name).
		Trace("managed application info")

	consumerDetails := &ConsumerDetails{}

	// get subscription info
	subRef := accessRequest.GetReferenceByGVK(cv1.SubscriptionGVK())
	if subRef.ID == "" || subRef.Name == "" {
		e.logger.Debug("could not get subscription, no consumer information attached")
		return nil
	}
	subscription := &Subscription{
		ID:   subRef.ID,
		Name: subRef.Name,
	}
	e.logger.
		WithField("subscription ID", subscription.ID).
		WithField("subscription name", subscription.Name).
		Trace("subscription information")

	// add subscription to consumer details
	consumerDetails.Subscription = subscription

	consumerOrgID := unknown

	if managedApp != nil {
		// get application info
		appRef := accessRequest.GetReferenceByGVK(cv1.ApplicationGVK())
		if appRef.ID == "" || appRef.Name == "" {
			e.logger.Debug("could not get application, no consumer information attached")
			return nil
		}
		application := &Application{
			ID:   appRef.ID,
			Name: appRef.Name,
		}
		e.logger.
			WithField("application ID", application.ID).
			WithField("application name", application.Name).
			Trace("application information")

		// add application to consumer details
		consumerDetails.Application = application

		// try to get consumer org ID from the managed app first
		consumerOrgID = transutil.GetConsumerOrgID(managedApp)
		if consumerOrgID == "" {
			e.logger.Debug("could not get consumer org ID from the managed app, try getting consumer org ID from subscription")
			return nil
		}
		e.logger.
			WithField("consumer org ID", consumerOrgID).
			Trace("consumer org ID ")

		// add organization ID to consumer details
		consumerDetails.OrgID = consumerOrgID

		// try to get Published product info
		publishProductRef := accessRequest.GetReferenceByGVK(cv1.PublishedProductGVK())
		if publishProductRef.ID == "" || publishProductRef.Name == "" {
			e.logger.Debug("could not get published product, no consumer information attached")
		}
		publishedProduct := &PublishedProduct{
			ID:   publishProductRef.ID,
			Name: publishProductRef.Name,
		}
		e.logger.
			WithField("application ID", publishedProduct.ID).
			WithField("application name", publishedProduct.Name).
			Trace("published product information")

		// add published product to consumer details
		consumerDetails.PublishedProduct = publishedProduct
	}

	// Update consumer details with Org, Application and Subscription
	summaryEvent.TransactionSummary.ConsumerDetails = consumerDetails
	e.updateProviderDetails(accessRequest, summaryEvent.TransactionSummary)
	return summaryEvent.TransactionSummary

}

func (e *Generator) updateProviderDetails(accessRequest *v1alpha1.AccessRequest, summaryEvent *Summary) *Summary {
	// get asset resource
	assetResourceRef := accessRequest.GetReferenceByGVK(cv1.AssetResourceGVK())
	if assetResourceRef.ID == "" || assetResourceRef.Name == "" {
		e.logger.Debug("could not get asset resource, not added to transaction summary")
	}
	assetResource := &AssetResource{
		ID:   assetResourceRef.ID,
		Name: assetResourceRef.Name,
	}
	e.logger.
		WithField("asset resource ID", assetResourceRef.ID).
		WithField("asset resource name", assetResource.Name).
		Trace("asset resource information")
	// add asset resource information
	summaryEvent.AssetResource = assetResource

	// get plan ID
	productPlanRef := accessRequest.GetReferenceByGVK(cv1.ProductPlanGVK())
	if productPlanRef.ID == "" {
		e.logger.Debug("could not get product plan ID, not added to transaction summary")
	}
	productPlan := &ProductPlan{
		ID: productPlanRef.ID,
	}
	e.logger.
		WithField("product plan ID", productPlanRef.ID).
		Trace("product plan ID information")
	// add product plan ID
	summaryEvent.ProductPlan = productPlan

	// get quota
	quotaRef := accessRequest.GetReferenceByGVK(cv1.QuotaGVK())
	if quotaRef.ID == "" {
		e.logger.Debug("could not get quota ID, not added to transaction summary")
	}
	quota := &Quota{
		ID: quotaRef.ID,
	}
	e.logger.
		WithField("quota ID", quota.ID).
		Trace("quota ID information")
	// add quota ID
	summaryEvent.Quota = quota

	apiserviceinstance := accessRequest.Spec.ApiServiceInstance
	if apiserviceinstance == "" {
		e.logger.Debug("could not get apiserviceinstance, not added to transaction summary")
	}
	e.logger.
		WithField("proxy ID", summaryEvent.Proxy.ID).
		WithField("proxy Name", summaryEvent.Proxy.Name).
		WithField("proxy Revision", summaryEvent.Proxy.Revision).
		WithField("apiserviceinstance", apiserviceinstance).
		Trace("apiserviceinstance information")
		// get apiservice instance information
	apiDetails := APIDetails{
		ID:                 summaryEvent.Proxy.ID,
		Name:               summaryEvent.Proxy.Name,
		Revision:           summaryEvent.Proxy.Revision,
		APIServiceInstance: apiserviceinstance,
	}
	// add apiDetails
	summaryEvent.API = apiDetails

	return summaryEvent
}

// createSamplingTransactionDetails -
func (e *Generator) createSamplingTransactionDetails(summaryEvent LogEvent) sampling.TransactionDetails {
	var status string
	var apiID string
	var subID string

	if summaryEvent.TransactionSummary != nil {
		status = summaryEvent.TransactionSummary.Status
		if summaryEvent.TransactionSummary.Proxy != nil {
			apiID = summaryEvent.TransactionSummary.Proxy.ID
		}

		consumerDetails := summaryEvent.TransactionSummary.ConsumerDetails
		if consumerDetails != nil && consumerDetails.Subscription != nil {
			subID = consumerDetails.Subscription.ID
		}
	}

	return sampling.TransactionDetails{
		Status: status,
		APIID:  apiID,
		SubID:  subID,
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
