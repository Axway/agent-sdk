package filter

// AnyExpr - Any implementation. Returns all filter data values to be checked against
type AnyExpr struct {
	FilterType string
}

func newAnyExpr(filterType string) CallExpr {
	return &AnyExpr{
		FilterType: filterType,
	}
}

// GetType - Returns the CallType
func (e *AnyExpr) GetType() CallType {
	return ANY
}

// Execute - Returns all filter data values to be checked against
func (e *AnyExpr) Execute(data Data) (interface{}, error) {
	return data.GetValues(e.FilterType), nil
}

func (e *AnyExpr) String() string {
	return e.FilterType + ".Any()"
}
