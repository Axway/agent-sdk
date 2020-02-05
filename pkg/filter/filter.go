package filter

import (
	log "github.com/sirupsen/logrus"
	"strconv"
)

// Filter - Interface for filter
type Filter interface {
	Evaluate(tags interface{}) bool
}

// AgentFilter - Represents the filter
type AgentFilter struct {
	filterConditions []Condition
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
	if af.filterConditions != nil && len(af.filterConditions) > 0 {
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
