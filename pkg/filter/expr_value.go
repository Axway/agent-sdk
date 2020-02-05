package filter

import "errors"

// ValueExpr - Get Value implementation. Evaulates the value from filter data based on selector name
type ValueExpr struct {
	FilterType string
	Name       string
}

func newValueExpr(filterType, name string) CallExpr {
	return &ValueExpr{
		FilterType: filterType,
		Name:       name,
	}
}

// GetType - Returns the CallType
func (e *ValueExpr) GetType() CallType {
	return GETVALUE
}

// Execute - Returns the value based on the selector name
func (e *ValueExpr) Execute(data Data) (interface{}, error) {
	val, ok := data.GetValue(e.FilterType, e.Name)
	if !ok {
		return nil, errors.New("Filter key " + e.FilterType + "." + e.Name + " not found in filter data")
	}
	return val, nil
}

func (e *ValueExpr) String() string {
	return e.FilterType + "." + e.Name
}
