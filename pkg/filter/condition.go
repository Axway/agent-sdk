package filter

import (
	"go/token"
	"strconv"
)

// Condition - Interface for the filter condition
type Condition interface {
	Evaluate(data Data) bool
	String() string
}

// SimpleCondition - Identifies a simple condition
type SimpleCondition struct {
	LHSExpr  CallExpr
	Value    ComparableValue
	Operator string
}

// Evaluate - evaluates a simple/call expression condition
func (sf *SimpleCondition) Evaluate(data Data) (res bool) {
	lhsValue, err := sf.LHSExpr.Execute(data)
	if err != nil {
		return false
	}
	callType := sf.LHSExpr.GetType()
	switch callType {
	case ANY:
		if sf.Value != nil {
			res = sf.Value.any(lhsValue)
			if sf.Operator == token.NEQ.String() {
				res = !res
			}
		}
	default:
		if callType != GETVALUE {
			res = lhsValue.(bool)
			lhsValue = strconv.FormatBool(res)
		}

		if sf.Operator != "" && sf.Value != nil {
			if sf.Operator == token.EQL.String() {
				res = sf.Value.eq(lhsValue)
			} else if sf.Operator == token.NEQ.String() {
				res = sf.Value.neq(lhsValue)
			}
		}
	}

	return res
}

// String - string representation for simple condition
func (sf *SimpleCondition) String() string {
	str := sf.LHSExpr.String()
	if sf.Operator != "" {
		str += " " + sf.Operator
	}
	if sf.Value != nil && sf.Value.String() != "" {
		str += " " + sf.Value.String()
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
