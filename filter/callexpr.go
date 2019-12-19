package filter

import "errors"

// CallType - Call type for filter condition
type CallType int

// Constants for Call type
const (
	GETVALUE CallType = iota
	EXISTS
	ANY
)

// CallExpr - Identifies call expression in filter condition
type CallExpr struct {
	FilterType string
	Name       string
	Type       CallType
}

// Execute - Returns the data to be processed in filter evaluation for a call expression
func (ce *CallExpr) Execute(data Data) (interface{}, error) {
	switch ce.Type {
	case GETVALUE:
		return data.GetValue(ce.FilterType, ce.Name), nil
	case EXISTS:
		keys := data.GetKeys(ce.FilterType)
		return keys, nil
	case ANY:
		return data.GetValues(ce.FilterType), nil
	}

	return nil, errors.New("Unsupported Call type")
}

// String - Returns the string representation
func (ce *CallExpr) String() string {
	switch ce.Type {
	case GETVALUE:
		return ce.FilterType + "." + ce.Name
	case EXISTS:
		return ce.FilterType + "." + ce.Name + ".Exists()"
	case ANY:
		return ce.FilterType + ".Any()"
	}
	return "Invalid call expression"
}
