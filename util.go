package transaction

import (
	"encoding/json"
	"net/http"
)

// GetTransactionSummaryStatus - Returns the summary status based on HTTP status code.
func GetTransactionSummaryStatus(status int) string {
	transSummaryStatus := "Unknown"
	if status >= http.StatusOK && status < http.StatusBadRequest {
		transSummaryStatus = "Success"
	} else if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
		transSummaryStatus = "Failure"
	} else if status >= http.StatusInternalServerError && status < http.StatusNetworkAuthenticationRequired {
		transSummaryStatus = "Exception"
	}
	return transSummaryStatus
}

// GetTransactionEventStatus - Returns the transaction event status based on HTTP status code.
func GetTransactionEventStatus(status int) string {
	transStatus := "Fail"
	if status >= http.StatusOK && status < http.StatusBadRequest {
		transStatus = "Pass"
	}
	return transStatus
}

// MarshalHeadersAsJSONString - Serializes the header key/values in map as JSON string
func MarshalHeadersAsJSONString(headers map[string]string) string {
	bb, _ := json.Marshal(headers)
	return string(bb)
}
