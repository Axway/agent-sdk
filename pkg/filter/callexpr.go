package filter

import (
	"errors"
	"strings"
)

// CallType - Call type for filter condition
type CallType int

// Constants for Call type
const (
	GETVALUE CallType = iota
	MATCHREGEX
	CONTAINS
	EXISTS
	ANY
)

var callTypeMap = map[string]CallType{
	"any":        ANY,
	"exists":     EXISTS,
	"contains":   CONTAINS,
	"matchregex": MATCHREGEX,
}

// CallExpr - Interface for call expression in filter condition
type CallExpr interface {
	GetType() CallType
	Execute(data Data) (interface{}, error)
	String() string
}

// NewCallExpr - Factory method for creating CallExpr
func newCallExpr(callType CallType, filterType, name string, arguments []interface{}) (callExpr CallExpr, err error) {
	if (callType == ANY || callType == EXISTS) && len(arguments) != 0 {
		return nil, errors.New("syntax error, unrecognized argument(s)")
	}

	if callType == CONTAINS || callType == MATCHREGEX {
		if len(arguments) == 0 {
			return nil, errors.New("syntax error, missing argument")
		}
		if len(arguments) != 1 {
			return nil, errors.New("syntax error, unrecognized argument(s)")
		}
	}
	callTypeSupported := false
	for _, supportedCallType := range supportedExpr {
		if supportedCallType == callType {
			callTypeSupported = true
			break
		}
	}

	if !callTypeSupported {
		return nil, errors.New("syntax error, unsupported condition")
	}

	switch callType {
	case GETVALUE:
		callExpr = newValueExpr(filterType, name)
	case EXISTS:
		callExpr = newExistsExpr(filterType, name)
	case ANY:
		callExpr = newAnyExpr(filterType)
	case CONTAINS:
		callExpr = newContainsExpr(filterType, name, arguments[0].(string))
	case MATCHREGEX:
		callExpr, err = newMatchRegExExpr(filterType, name, arguments[0].(string))
	}

	return
}

// GetCallType - Converts a string to its corresponding call type.
func GetCallType(callTypeString string) (callType CallType, err error) {
	callType, ok := callTypeMap[strings.ToLower(callTypeString)]
	if !ok {
		err = errors.New("unsupported call")
	}
	return
}
