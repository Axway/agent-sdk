package util

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	cv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
