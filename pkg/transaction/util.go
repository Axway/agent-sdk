package transaction

import (
	"encoding/json"
	"net/http"
)

// IsSuccessStatus - Returns true if the HTTP status is between 200 and 400
func IsSuccessStatus(status int) bool {
	return status >= http.StatusOK && status < http.StatusBadRequest
}

// IsFailureStatus - Returns true if the HTTP status is between 400 and 500
func IsFailureStatus(status int) bool {
	return status >= http.StatusBadRequest && status < http.StatusInternalServerError
}

// IsExceptionStatus - Returns true if the HTTP status is between 500 and 511
func IsExceptionStatus(status int) bool {
	return status >= http.StatusInternalServerError && status < http.StatusNetworkAuthenticationRequired
}

// GetTransactionSummaryStatus - Returns the summary status based on HTTP status code.
func GetTransactionSummaryStatus(status int) string {
	transSummaryStatus := "Unknown"
	if IsSuccessStatus(status) {
		transSummaryStatus = "Success"
	} else if IsFailureStatus(status) {
		transSummaryStatus = "Failure"
	} else if IsExceptionStatus(status) {
		transSummaryStatus = "Exception"
	}
	return transSummaryStatus
}

// GetTransactionEventStatus - Returns the transaction event status based on HTTP status code.
func GetTransactionEventStatus(status int) string {
	transStatus := "Fail"
	if IsSuccessStatus(status) {
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
