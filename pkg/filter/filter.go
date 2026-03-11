package filter

import (
	"strconv"

	log "github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	Dash            = "-"
	DashPlaceHolder = "__DASH__"
)

// Filter - Interface for filter
type Filter interface {
	Evaluate(tags interface{}) bool
}

// AgentFilter - Represents the filter
type AgentFilter struct {
	filterConditions []Condition
}

var defaultSupportedExpr = []CallType{GETVALUE, MATCHREGEX, CONTAINS, EXISTS, ANY}

var supportedExpr = defaultSupportedExpr

// SetSupportedCallExprTypes - Overrides the list of supported condition expression
func SetSupportedCallExprTypes(callTypes []CallType) {
	supportedExpr = defaultSupportedExpr
	overriddentSupportedExpr := []CallType{}
	for _, callType := range callTypes {
		switch callType {
		case GETVALUE:
			fallthrough
		case MATCHREGEX:
			fallthrough
		case CONTAINS:
			fallthrough
		case EXISTS:
			fallthrough
		case ANY:
			overriddentSupportedExpr = append(overriddentSupportedExpr, callType)
		}
	}
	if len(overriddentSupportedExpr) > 0 {
		supportedExpr = overriddentSupportedExpr
	}
}

// NewFilter - Creates a new instance of the filter
func NewFilter(filterConfig string) (filter Filter, err error) {
	conditionParser := NewConditionParser()
	filterConditions, err := conditionParser.Parse(filterConfig)
	if err != nil {
		return
	}
	filter = &AgentFilter{
		filterConditions: filterConditions,
	}
	return
}

// Evaluate - Performs the evaluation of the filter against the data
func (af *AgentFilter) Evaluate(tags interface{}) (result bool) {
	if len(af.filterConditions) > 0 {
		fd := NewFilterData(tags, nil)
		for _, filterCondition := range af.filterConditions {
			result = filterCondition.Evaluate(fd)
			log.Debug("Filter condition evaluation [Condition: " + filterCondition.String() + ", Result: " + strconv.FormatBool(result) + "]")
			if result {
				return
			}
		}
	} else {
		result = true
	}
	return
}
