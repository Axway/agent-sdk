package sampling

import "strconv"

// SampleKey - the key used in the metadata when a transaction qualifies for sampling and should be sent to Observer
// defaultSamplingRate - the default sampling rate in percentage
const (
	SampleKey            = "sample"
	countMax             = 100
	defaultSamplingRate  = 0
	defaultSamplingLimit = 0
	maximumSamplingRate  = 10
	globalCounter        = "global"
)

// TransactionDetails - details about the transaction that are used for sampling
type TransactionDetails struct {
	Status string
	APIID  string
	SubID  string
}

type statusText string

const (
	Success   statusText = "Success"
	Failure   statusText = "Failure"
	Exception statusText = "Exception"
)

var statuses = map[string]statusText{
	Success.String():   Success,
	Failure.String():   Failure,
	Exception.String(): Exception,
}

func (s statusText) String() string {
	return string(s)
}

func GetStatusFromCodeString(statusCode string) statusText {
	if v, ok := statuses[statusCode]; ok {
		return v
	}

	httpStatusCode, _ := strconv.Atoi(statusCode)
	return getStatusFromCode(httpStatusCode)
}

func getStatusFromCode(statusCode int) statusText {
	switch {
	case statusCode >= 100 && statusCode < 400:
		return Success
	case statusCode >= 400 && statusCode < 500:
		return Failure
	default:
		return Exception
	}
}
