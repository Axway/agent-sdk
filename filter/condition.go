package filter

import (
	"go/token"
)

// Condition - Interface for the filter condition
type Condition interface {
	Evaluate(data Data) bool
	String() string
}

// SimpleCondition - Identifies a simple condition
type SimpleCondition struct {
	LHSExpr  *CallExpr
	Value    string
	Operator string
}

// Evaluate - evaluates a simple/call expression condition
func (sf *SimpleCondition) Evaluate(data Data) bool {
	valueToCompare, err := sf.LHSExpr.Execute(data)
	if err != nil {
		return false
	}
	res := false
	switch sf.LHSExpr.Type {
	case GETVALUE:
		if sf.Operator == token.EQL.String() {
			res = sf.compareEQ(valueToCompare.(string), sf.Value)
		} else if sf.Operator == token.NEQ.String() {
			res = sf.compareNEQ(valueToCompare.(string), sf.Value)
		}
	case EXISTS:
		res = sf.exists(valueToCompare.([]string), sf.LHSExpr.Name)
	case ANY:
		res = sf.matchAny(valueToCompare.([]string), sf.Value)
		if sf.Operator == token.NEQ.String() {
			res = !res
		}
	}

	return res
}

func (sf *SimpleCondition) compareEQ(valueToCompare string, value string) bool {
	return valueToCompare == value
}

func (sf *SimpleCondition) compareNEQ(valueToCompare string, value string) bool {
	return valueToCompare != value
}

func (sf *SimpleCondition) exists(keysList []string, key string) bool {
	for _, keyEntry := range keysList {
		if keyEntry == key {
			return true
		}
	}
	return false
}

func (sf *SimpleCondition) matchAny(valueList []string, value string) bool {
	return sf.exists(valueList, value)
}

// String - string representation for simple condition
func (sf *SimpleCondition) String() string {
	str := sf.LHSExpr.String()
	if sf.Operator != "" {
		str += " " + sf.Operator
	}
	if sf.Value != "" {
		str += " " + sf.Value
	}
	return "(" + str + ")"
}

// CompoundCondition - Represents group of simple conditions
type CompoundCondition struct {
	RHSCondition Condition
	LHSCondition Condition
	Operator     string
}

// Evaluate - evaulates the compound condition
func (cf *CompoundCondition) Evaluate(data Data) bool {
	lhsRes := cf.LHSCondition.Evaluate(data)
	rhsRes := cf.RHSCondition.Evaluate(data)
	if cf.Operator == token.LOR.String() {
		return (lhsRes || rhsRes)
	}
	return (lhsRes && rhsRes)
}

// String - string representation for compound condition
func (cf *CompoundCondition) String() string {
	if cf.RHSCondition != nil && cf.LHSCondition != nil {
		return "(" + cf.LHSCondition.String() + " " + cf.Operator + " " + cf.RHSCondition.String() + ")"
	}
	return ""
}
