package metric

import "strconv"

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

func getStatusFromCodeString(statusCode string) statusText {
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
