package filter

// ExistsExpr - Exists implementation. Checks the existance of the selector as key in filter data
type ExistsExpr struct {
	FilterType string
	Name       string
}

func newExistsExpr(filterType, name string) CallExpr {
	return &ExistsExpr{
		FilterType: filterType,
		Name:       name,
	}
}

// GetType - Returns the CallType
func (e *ExistsExpr) GetType() CallType {
	return EXISTS
}

// Execute - Returns true if the selector key is found in filter data
func (e *ExistsExpr) Execute(data Data) (interface{}, error) {
	keysList := data.GetKeys(e.FilterType)
	for _, keyEntry := range keysList {
		if keyEntry == e.Name {
			return true, nil
		}
	}
	return false, nil
}

func (e *ExistsExpr) String() string {
	return e.FilterType + "." + e.Name + ".Exists()"
}
