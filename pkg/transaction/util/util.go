package util

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	unknown = "unknown"
	// SummaryEventProxyIDPrefix - Prefix for proxyID in summary event
	SummaryEventProxyIDPrefix = "remoteApiId_"

	// SummaryEventApplicationIDPrefix - Prefix for application.ID in summary event
	SummaryEventApplicationIDPrefix = "remoteAppId_"
)

// GetAccessRequest -
func GetAccessRequest(cacheManager cache.Manager, managedApp *v1.ResourceInstance, apiID, stage, version string) *management.AccessRequest {
	if managedApp == nil || cacheManager == nil {
		if managedApp == nil {
			log.Trace("managedApp is nil")
		}
		if cacheManager == nil {
			log.Trace("cacheManager is nil")
		}
		return nil
	}

	// Lookup Access Request
	apiID = strings.TrimPrefix(apiID, "remoteApiId_")
	accessReq := &management.AccessRequest{}
	ri := cacheManager.GetAccessRequestByAppAndAPIStageVersion(managedApp.Name, apiID, stage, version)
	if ri == nil {
		return nil
	}
	if err := accessReq.FromInstance(ri); err != nil {
		log.Errorf("failed to convert resource instance to access request: %v", err)
		return nil
	}
	return accessReq
}

// GetSubscriptionID -
func GetSubscriptionID(subscription *v1.ResourceInstance) string {
	if subscription == nil {
		return unknown
	}
	return subscription.Metadata.ID
}

// GetConsumerOrgID -
func GetConsumerOrgID(ri *v1.ResourceInstance) string {
	if ri == nil {
		return ""
	}

	// Lookup Subscription
	app := &management.ManagedApplication{}
	if err := app.FromInstance(ri); err != nil {
		log.Errorf("failed to convert resource instance to managed application: %v", err)
		return ""
	}

	if app.Marketplace.Resource.Owner != nil {
		log.Trace("retrieving consumer org ID from managed application marketplace owner")
		return app.Marketplace.Resource.Owner.Organization.ID
	}
	log.Trace("retrieving consumer org ID from managed application metadata")
	return ""
}

// GetConsumerApplication -
func GetConsumerApplication(ri *v1.ResourceInstance) (string, string) {
	if ri == nil {
		return "", ""
	}

	for _, ref := range ri.Metadata.References {
		// get the ID of the Catalog Application
		if ref.Kind == catalog.ApplicationGVK().Kind {
			return ref.ID, ref.Name
		}
	}

	return ri.Metadata.ID, ri.Name // default to the managed app id
}

// IsHTTPSuccessStatus - Returns true if the HTTP status is between 200 and 400
func IsHTTPSuccessStatus(status int) bool {
	return status >= http.StatusOK && status < http.StatusBadRequest
}

// IsSuccessStatus - Returns true if the HTTP status is between 200 and 400
func IsSuccessStatus(status int) bool {
	// DEPRECATED
	log.DeprecationWarningReplace("IsSuccessStatus", "IsHTTPSuccessStatus")
	return IsHTTPSuccessStatus(status)
}

// IsHTTPFailureStatus - Returns true if the HTTP status is between 400 and 500
func IsHTTPFailureStatus(status int) bool {
	return status >= http.StatusBadRequest && status < http.StatusInternalServerError
}

// IsFailureStatus - Returns true if the HTTP status is between 400 and 500
func IsFailureStatus(status int) bool {
	// DEPRECATED
	log.DeprecationWarningReplace("IsFailureStatus", "IsHTTPFailureStatus")
	return IsHTTPFailureStatus(status)
}

// IsHTTPExceptionStatus - Returns true if the HTTP status is between 500 and 511
func IsHTTPExceptionStatus(status int) bool {
	return status >= http.StatusInternalServerError && status <= http.StatusNetworkAuthenticationRequired
}

// IsExceptionStatus - Returns true if the HTTP status is between 500 and 511
func IsExceptionStatus(status int) bool {
	// DEPRECATED
	log.DeprecationWarningReplace("IsExceptionStatus", "IsHTTPExceptionStatus")
	return IsHTTPExceptionStatus(status)
}

// GetTransactionSummaryStatus - Returns the summary status based on HTTP status code.
func GetTransactionSummaryStatus(status int) string {
	transSummaryStatus := "Unknown"
	if IsHTTPSuccessStatus(status) {
		transSummaryStatus = "Success"
	} else if IsHTTPFailureStatus(status) {
		transSummaryStatus = "Failure"
	} else if IsHTTPExceptionStatus(status) {
		transSummaryStatus = "Exception"
	}
	return transSummaryStatus
}

// GetTransactionEventStatus - Returns the transaction event status based on HTTP status code.
func GetTransactionEventStatus(status int) string {
	transStatus := "Fail"
	if IsHTTPSuccessStatus(status) {
		transStatus = "Pass"
	}
	return transStatus
}

// MarshalHeadersAsJSONString - Serializes the header key/values in map as JSON string
func MarshalHeadersAsJSONString(headers map[string]string) string {
	bb, _ := json.Marshal(headers)
	return string(bb)
}

// FormatProxyID - Returns the prefixed proxyID for summary event.
func FormatProxyID(proxyID string) string {
	return SummaryEventProxyIDPrefix + proxyID
}

// FormatApplicationID - Returns the prefixed applicationID for summary event.
func FormatApplicationID(applicationID string) string {
	return SummaryEventApplicationIDPrefix + applicationID
}

// UpdateWithConsumerDetails -
func UpdateWithConsumerDetails(accessRequest *management.AccessRequest, managedApp *v1.ResourceInstance, log log.FieldLogger) *models.ConsumerDetails {

	// Set defaults to unknown to consumer details in case access request or managed apps comes back nil
	consumerDetails := &models.ConsumerDetails{
		Subscription: &models.Subscription{
			ID:   unknown,
			Name: unknown,
		},
		Application: &models.AppDetails{
			ConsumerOrgID: unknown,
			ID:            unknown,
			Name:          unknown,
		},
		PublishedProduct: &models.Product{
			ID:   unknown,
			Name: unknown,
		},
	}

	if accessRequest == nil || managedApp == nil {
		log.Trace("access request or managed app is nil. Setting default values to unknown")
		return consumerDetails
	}

	subRef := accessRequest.GetReferenceByGVK(catalog.SubscriptionGVK())
	if subRef.ID == "" || subRef.Name == "" {
		log.Debug("could not get subscription, setting subscription to unknown")
	} else {
		consumerDetails.Subscription.ID = subRef.ID
		consumerDetails.Subscription.Name = subRef.Name
	}
	log.
		WithField("subscription-id", consumerDetails.Subscription.ID).
		WithField("subscription-name", consumerDetails.Subscription.Name).
		Trace("subscription information")

	appRef := accessRequest.GetReferenceByGVK(catalog.ApplicationGVK())
	if appRef.ID == "" || appRef.Name == "" {
		log.Debug("could not get application, setting application to unknown")
	} else {
		consumerDetails.Application.ID = appRef.ID
		consumerDetails.Application.Name = appRef.Name
	}

	log.
		WithField("application-id", consumerDetails.Application.ID).
		WithField("application-name", consumerDetails.Application.Name).
		Trace("application information")

	// try to get consumer org ID from the managed app first

	consumerOrgID := GetConsumerOrgID(managedApp)
	if consumerOrgID == "" {
		log.Debug("could not get consumer org ID from the managed app, try getting consumer org ID from subscription")
	} else {
		consumerDetails.Application.ConsumerOrgID = consumerOrgID
	}
	log.
		WithField("consumer-org-id", consumerDetails.Application.ConsumerOrgID).
		Trace("consumer org ID ")

	publishProductRef := accessRequest.GetReferenceByGVK(catalog.PublishedProductGVK())
	if publishProductRef.ID == "" || publishProductRef.Name == "" {
		log.Debug("could not get published product, setting published product to unknown")
	} else {
		consumerDetails.PublishedProduct.ID = publishProductRef.ID
		consumerDetails.PublishedProduct.Name = publishProductRef.Name
	}

	log.
		WithField("published-product-id", consumerDetails.PublishedProduct.ID).
		WithField("published-product-name", consumerDetails.PublishedProduct.Name).
		Trace("published product information")

	return consumerDetails
}
