package filter

// ComparableValue - Interface for RHS value operand
type ComparableValue interface {
	eq(interface{}) bool
	neq(interface{}) bool
	any(interface{}) bool
	String() string
}

// StringRHSValue - Represents the RHS value in simple condition
type StringRHSValue struct {
	value string
}

func newStringRHSValue(value string) ComparableValue {
	return &StringRHSValue{
		value: value,
	}
}

func (scv *StringRHSValue) eq(valueToCompare interface{}) bool {
	return valueToCompare == scv.value
}

func (scv *StringRHSValue) neq(valueToCompare interface{}) bool {
	return valueToCompare != scv.value
}

func (scv *StringRHSValue) any(valuesToCompare interface{}) bool {
	values := valuesToCompare.([]string)
	for _, valueEntry := range values {
		if scv.eq(valueEntry) {
			return true
		}
	}
	return false
}

func (scv *StringRHSValue) String() string {
	return scv.value
}
