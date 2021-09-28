package transaction

import (
	"encoding/json"
	"net/http"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

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
