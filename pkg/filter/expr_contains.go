package filter

import (
	"strings"
)

// ContainsExpr - Contains implementation. Check if the argument string is contained in value of specified filter data
type ContainsExpr struct {
	FilterType string
	Name       string
	Arg        string
}

func newContainsExpr(filterType, name, containsArg string) CallExpr {
	return &ContainsExpr{
		FilterType: filterType,
		Name:       name,
		Arg:        containsArg,
	}
}

// GetType - Returns the CallType
func (e *ContainsExpr) GetType() CallType {
	return CONTAINS
}

// Execute - Returns true if the argument string is contained in value of specified filter data
func (e *ContainsExpr) Execute(data Data) (interface{}, error) {
	valueToCompare, ok := data.GetValue(e.FilterType, e.Name)
	if ok && strings.Contains(valueToCompare, e.Arg) {
		return true, nil
	}

	return false, nil
}

func (e *ContainsExpr) String() string {
	return e.FilterType + "." + e.Name + ".Contains(\"" + e.Arg + "\")"
}
