package util

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	cv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
func GetAccessRequest(cacheManager cache.Manager, managedApp *v1.ResourceInstance, apiID, stage string) *v1alpha1.AccessRequest {
	if managedApp == nil {
		return nil
	}

	// Lookup Access Request
	apiID = strings.TrimPrefix(apiID, "remoteApiId_")
	accessReq := &v1alpha1.AccessRequest{}
	ri := cacheManager.GetAccessRequestByAppAndAPI(managedApp.Name, apiID, stage)
	accessReq.FromInstance(ri)
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
	app := &v1alpha1.ManagedApplication{}
	app.FromInstance(ri)

	return app.Marketplace.Resource.Owner.Organization.Id
}

// GetConsumerApplication -
func GetConsumerApplication(ri *v1.ResourceInstance) (string, string) {
	if ri == nil {
		return "", ""
	}

	for _, ref := range ri.Metadata.References {
		// get the ID of the Catalog Application
		if ref.Kind == cv1.ApplicationGVK().Kind {
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
func UpdateWithConsumerDetails(accessRequest *v1alpha1.AccessRequest, managedApp *v1.ResourceInstance, log log.FieldLogger) *models.ConsumerDetails {
	consumerDetails := &models.ConsumerDetails{}

	// get subscription info
	subscription := &models.Subscription{
		ID:   unknown,
		Name: unknown,
	}
	subRef := accessRequest.GetReferenceByGVK(cv1.SubscriptionGVK())
	if subRef.ID == "" || subRef.Name == "" {
		log.Debug("could not get subscription, setting subscription to unknown")
	} else {
		subscription.ID = subRef.ID
		subscription.Name = subRef.Name
	}
	log.
		WithField("subscription ID", subscription.ID).
		WithField("subscription name", subscription.Name).
		Trace("subscription information")

		// add subscription to consumer details
	consumerDetails.Subscription = subscription

	// get application info
	application := &models.AppDetails{
		ConsumerOrgID: unknown,
		ID:            unknown,
		Name:          unknown,
	}
	appRef := accessRequest.GetReferenceByGVK(cv1.ApplicationGVK())
	if appRef.ID == "" || appRef.Name == "" {
		log.Debug("could not get application, setting application to unknown")
	} else {
		application.ID = appRef.ID
		application.Name = appRef.Name
	}

	log.
		WithField("application ID", application.ID).
		WithField("application name", application.Name).
		Trace("application information")

	// try to get consumer org ID from the managed app first

	consumerOrgID := GetConsumerOrgID(managedApp)
	if consumerOrgID == "" {
		log.Debug("could not get consumer org ID from the managed app, try getting consumer org ID from subscription")
		consumerOrgID = unknown
	} else {
		application.ConsumerOrgID = consumerOrgID
	}
	log.
		WithField("consumer org ID", consumerOrgID).
		Trace("consumer org ID ")

	// add application to consumer details
	consumerDetails.Application = application

	// try to get Published product info
	publishedProduct := &models.PublishedProduct{
		ID:   unknown,
		Name: unknown,
	}
	publishProductRef := accessRequest.GetReferenceByGVK(cv1.PublishedProductGVK())
	if publishProductRef.ID == "" || publishProductRef.Name == "" {
		log.Debug("could not get published product, setting published product to unknown")
	} else {
		publishedProduct.ID = publishProductRef.ID
		publishedProduct.Name = publishProductRef.Name
	}

	log.
		WithField("application ID", publishedProduct.ID).
		WithField("application name", publishedProduct.Name).
		Trace("published product information")

	// add published product to consumer details
	consumerDetails.PublishedProduct = publishedProduct

	return consumerDetails
}

// UpdateWithProviderDetails -
func UpdateWithProviderDetails(accessRequest *v1alpha1.AccessRequest, log log.FieldLogger) models.ProviderDetails {
	providerDetails := models.ProviderDetails{}

	// get asset resource
	assetResource := &models.AssetResource{
		ID:   unknown,
		Name: unknown,
	}

	assetResourceRef := accessRequest.GetReferenceByGVK(cv1.AssetResourceGVK())
	if assetResourceRef.ID == "" || assetResourceRef.Name == "" {
		log.Debug("could not get asset resource, setting asset resource to unknown")
	} else {
		assetResource.ID = assetResourceRef.ID
		assetResource.Name = assetResourceRef.Name
	}
	log.
		WithField("asset resource ID", assetResource.ID).
		WithField("asset resource name", assetResource.Name).
		Trace("asset resource information")
	// add asset resource information
	providerDetails.AssetResource = assetResource

	// get product
	product := &models.Product{
		ID:      unknown,
		Name:    unknown,
		Version: unknown,
	}
	productRef := accessRequest.GetReferenceByGVK(cv1.ProductGVK())
	if productRef.ID == "" || productRef.Name == "" {
		log.Debug("could not get product ID or Name, setting product to unknown")
	} else {
		product.ID = productRef.ID
		product.Name = productRef.Name
		// product.Version = productRef.Version TODO
	}
	log.
		WithField("product ID", product.ID).
		WithField("product Name", product.Name).
		WithField("product Version", product.Version)
	// add product information
	providerDetails.Product = product

	// get plan ID
	productPlan := &models.ProductPlan{
		ID: unknown,
	}
	productPlanRef := accessRequest.GetReferenceByGVK(cv1.ProductPlanGVK())
	if productPlanRef.ID == "" {
		log.Debug("could not get product plan ID, setting product plan to unknown")
	} else {
		productPlan.ID = productPlanRef.ID
	}
	log.
		WithField("product plan ID", productPlan.ID).
		Trace("product plan ID information")
	// add product plan ID
	providerDetails.ProductPlan = productPlan

	// get quota
	quota := &models.Quota{
		ID: unknown,
	}
	quotaRef := accessRequest.GetReferenceByGVK(cv1.QuotaGVK())
	if quotaRef.ID == "" {
		log.Debug("could not get quota ID, setting quota to unknown")
	} else {
		quota.ID = quotaRef.ID
	}
	log.
		WithField("quota ID", quota.ID).
		Trace("quota ID information")
	// add quota ID
	providerDetails.Quota = quota

	return providerDetails
}
