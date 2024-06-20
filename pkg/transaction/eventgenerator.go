package transaction

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/traceability"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
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

type EventGeneratorOpt func(*Generator)

// Generator - Create the events to be published to Condor
type Generator struct {
	shouldAddFields                bool
	shouldUseTrafficForAggregation bool
	isOffline                      bool
	logger                         log.FieldLogger
}

// NewEventGenerator - Create a new event generator
func NewEventGenerator(opts ...EventGeneratorOpt) EventGenerator {
	logger := log.NewFieldLogger().
		WithPackage("sdk.transaction.eventgenerator").
		WithComponent("eventgenerator")
	eventGen := &Generator{
		shouldAddFields:                !traceability.IsHTTPTransport(),
		shouldUseTrafficForAggregation: true,
		logger:                         logger,
	}

	for _, o := range opts {
		o(eventGen)
	}

	hc.RegisterHealthcheck("Event Generator", "eventgen", eventGen.healthcheck)

	return eventGen
}

func WithOfflineModeSetting(isOffline bool) EventGeneratorOpt {
	return func(g *Generator) {
		g.isOffline = isOffline
	}
}

// SetUseTrafficForAggregation - set the flag to use traffic events for aggregation.
func (e *Generator) SetUseTrafficForAggregation(useTrafficForAggregation bool) {
	e.shouldUseTrafficForAggregation = useTrafficForAggregation
}

// CreateEvent - Creates a new event to be sent to Amplify Observability, expects sampling is handled by agent
func (e *Generator) CreateEvent(logEvent LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) (beat.Event, error) {
	// if CreateEvent is being used, sampling will not work, so all events need to be sent
	if metaData == nil {
		metaData = common.MapStr{}
	}
	metaData.Put(sampling.SampleKey, true)

	if logEvent.TransactionSummary != nil {

		e.processTxnSummary(logEvent)
		e.trackMetrics(logEvent, 0)
	}

	return e.createEvent(logEvent, eventTime, metaData, eventFields, privateData)
}

func (e *Generator) trackMetrics(summaryEvent LogEvent, bytes int64) {
	if e.shouldUseTrafficForAggregation {
		apiDetails := models.APIDetails{
			ID:       summaryEvent.TransactionSummary.Proxy.ID,
			Name:     summaryEvent.TransactionSummary.Proxy.Name,
			Revision: summaryEvent.TransactionSummary.Proxy.Revision,
			Stage:    summaryEvent.TransactionSummary.Proxy.Stage,
			Version:  summaryEvent.TransactionSummary.Proxy.Version,
		}

		if summaryEvent.TransactionSummary.Team != nil {
			apiDetails.TeamID = summaryEvent.TransactionSummary.Team.ID
		}

		statusCode := summaryEvent.TransactionSummary.StatusDetail
		duration := summaryEvent.TransactionSummary.Duration
		appDetails := models.AppDetails{}
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

	// Check to see if marketplace provisioning/subs is enabled
	err := e.processTxnSummary(summaryEvent)
	if err != nil {
		return nil, err
	}

	//if no summary is sent then prepare the array of TransactionEvents for publishing
	if summaryEvent == (LogEvent{}) {
		return e.handleTransactionEvents(detailEvents, eventTime, metaData, eventFields, privateData)
	}

	shouldSample := false
	if !e.isOffline { // do not set sampling when offline
		// Add this to sample or not
		shouldSample, err = sampling.ShouldSampleTransaction(e.createSamplingTransactionDetails(summaryEvent))
	}
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

func (e *Generator) handleTransactionEvents(detailEvents []LogEvent, eventTime time.Time, metaData common.MapStr, eventFields common.MapStr, privateData interface{}) ([]beat.Event, error) {
	events := make([]beat.Event, 0)
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

func (e *Generator) processTxnSummary(summaryEvent LogEvent) error {
	// only process if there is a central client and marketplace subs are enabled
	if agent.GetCentralClient() == nil || !agent.GetCentralClient().IsMarketplaceSubsEnabled() {
		return nil
	}
	if summaryEvent.TransactionSummary != nil {
		txnSummary := e.updateTxnSummaryByAccessRequest(summaryEvent)
		if txnSummary != nil {
			jsonData, err := json.Marshal(&txnSummary)
			if err != nil {
				return err
			}
			e.logger.Trace(string(jsonData))
			summaryEvent.TransactionSummary = txnSummary
		}
	}
	return nil
}

// updateTxnSummaryByAccessRequest - get the consumer information to add to transaction event.  If we don't have any
//
//	information we need to get the consumer information, then we just return nil
func (e *Generator) updateTxnSummaryByAccessRequest(summaryEvent LogEvent) *Summary {
	cacheManager := agent.GetCacheManager()

	// get proxy information
	if summaryEvent.TransactionSummary.Proxy == nil {
		e.logger.Debug("proxy information is not available, no consumer information attached")
		return nil
	}

	// Go get the access request and managed app
	accessRequest, managedApp := e.getAccessRequest(cacheManager, summaryEvent)

	// Update the consumer details
	summaryEvent.TransactionSummary.ConsumerDetails = transutil.UpdateWithConsumerDetails(accessRequest, managedApp, e.logger)

	// Update provider details
	updatedSummaryEvent := updateWithProviderDetails(accessRequest, managedApp, summaryEvent.TransactionSummary, e.logger)

	return updatedSummaryEvent
}

// getAccessRequest -
func (e *Generator) getAccessRequest(cacheManager cache.Manager, summaryEvent LogEvent) (*management.AccessRequest, *v1.ResourceInstance) {
	appName := unknown
	apiID := summaryEvent.TransactionSummary.Proxy.ID
	stage := summaryEvent.TransactionSummary.Proxy.Stage
	version := summaryEvent.TransactionSummary.Proxy.Version
	e.logger.
		WithField("api-id", apiID).
		WithField("stage", stage).
		Trace("transaction summary proxy information")

	if summaryEvent.TransactionSummary.Application != nil {
		appName = summaryEvent.TransactionSummary.Application.Name
		e.logger.
			WithField("app-name", appName).
			Trace("transaction summary dataplane details application name")
	}

	// get the managed application
	managedApp := cacheManager.GetManagedApplicationByName(appName)
	if managedApp == nil {
		e.logger.
			WithField("app-name", appName).
			Trace("could not get managed application by name, no consumer information attached")
		return nil, nil
	}
	e.logger.
		WithField("app-name", appName).
		WithField("managed-app-name", managedApp.Name).
		Trace("managed application info")

	// get the access request
	accessRequest := transutil.GetAccessRequest(cacheManager, managedApp, apiID, stage, version)
	if accessRequest == nil {
		e.logger.
			Warn("could not get access request, no consumer information attached")
		return nil, nil
	}
	e.logger.
		WithField("managed-app-name", managedApp.Name).
		WithField("api-id", apiID).
		WithField("stage", stage).
		WithField("access-request-name", accessRequest.Name).
		Trace("managed application info")

	return accessRequest, managedApp
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

// updateWithProviderDetails -
func updateWithProviderDetails(accessRequest *management.AccessRequest, managedApp *v1.ResourceInstance, summaryEvent *Summary, log log.FieldLogger) *Summary {

	// Set default to provider details in case access request or managed apps comes back nil
	summaryEvent.AssetResource = &models.AssetResource{
		ID:   unknown,
		Name: unknown,
	}

	summaryEvent.Product = &models.Product{
		ID:          unknown,
		Name:        unknown,
		VersionID:   unknown,
		VersionName: unknown,
	}

	summaryEvent.ProductPlan = &models.ProductPlan{
		ID: unknown,
	}

	summaryEvent.Quota = &models.Quota{
		ID: unknown,
	}

	if accessRequest == nil || managedApp == nil {
		log.Trace("access request or managed app is nil. Setting default values to unknown")
		return summaryEvent
	}

	productRef := accessRequest.GetReferenceByGVK(catalog.ProductGVK())
	if productRef.ID == "" || productRef.Name == "" {
		log.Trace("could not get product information, setting product to unknown")
	} else {
		summaryEvent.Product.ID = productRef.ID
		summaryEvent.Product.Name = productRef.Name
	}

	productReleaseRef := accessRequest.GetReferenceByGVK(catalog.ProductReleaseGVK())
	if productReleaseRef.ID == "" || productReleaseRef.Name == "" {
		log.Trace("could not get product release information, setting product release to unknown")
	} else {
		summaryEvent.Product.VersionID = productReleaseRef.ID
		summaryEvent.Product.VersionName = productReleaseRef.Name
	}
	log.
		WithField("product-id", summaryEvent.Product.ID).
		WithField("product-name", summaryEvent.Product.Name).
		WithField("product-version-id", summaryEvent.Product.VersionID).
		WithField("product-version-name", summaryEvent.Product.VersionName).
		Trace("product information")

	assetResourceRef := accessRequest.GetReferenceByGVK(catalog.AssetResourceGVK())
	if assetResourceRef.ID == "" || assetResourceRef.Name == "" {
		log.Trace("could not get asset resource, setting asset resource to unknown")
	} else {
		summaryEvent.AssetResource.ID = assetResourceRef.ID
		summaryEvent.AssetResource.Name = assetResourceRef.Name
	}
	log.
		WithField("asset-resource-id", summaryEvent.AssetResource.ID).
		WithField("asset-resource-name", summaryEvent.AssetResource.Name).
		Trace("asset resource information")

	api := &models.APIDetails{
		ID:                 summaryEvent.Proxy.ID,
		Name:               summaryEvent.Proxy.Name,
		Revision:           summaryEvent.Proxy.Revision,
		TeamID:             summaryEvent.Team.ID,
		APIServiceInstance: accessRequest.Spec.ApiServiceInstance,
	}
	summaryEvent.API = api
	log.
		WithField("proxy-id", summaryEvent.Proxy.ID).
		WithField("proxy-name", summaryEvent.Proxy.Name).
		WithField("proxy-revision", summaryEvent.Proxy.Revision).
		WithField("proxy-team-id", summaryEvent.Team.ID).
		WithField("apiservice", accessRequest.Spec.ApiServiceInstance).
		Trace("api details information")

	productPlanRef := accessRequest.GetReferenceByGVK(catalog.ProductPlanGVK())
	if productPlanRef.ID == "" {
		log.Debug("could not get product plan ID, setting product plan to unknown")
	} else {
		summaryEvent.ProductPlan.ID = productPlanRef.ID
	}
	log.
		WithField("product-plan-id", summaryEvent.ProductPlan.ID).
		Trace("product plan ID information")

	quotaRef := accessRequest.GetReferenceByGVK(catalog.QuotaGVK())
	if quotaRef.ID == "" {
		log.Debug("could not get quota ID, setting quota to unknown")
	} else {
		summaryEvent.Quota.ID = quotaRef.ID
	}
	log.
		WithField("quota-id", summaryEvent.Quota.ID).
		Trace("quota ID information")

	return summaryEvent
}
