package filter

import (
	"errors"
	"regexp"
)

// MatchRegExExpr - MatchRegEx implementation. Performs a regular expression match for argument against value of specified filter data
type MatchRegExExpr struct {
	FilterType string
	Name       string
	Arg        *regexp.Regexp
}

func newMatchRegExExpr(filterType, name, regexStr string) (CallExpr, error) {
	regExArg, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, errors.New("Invalid regular expression(" + err.Error() + ") in MatchRegEx call")
	}
	return &MatchRegExExpr{
		FilterType: filterType,
		Name:       name,
		Arg:        regExArg,
	}, nil
}

// GetType - Returns the CallType
func (e *MatchRegExExpr) GetType() CallType {
	return MATCHREGEX
}

// Execute - Returns true if the regular expression in argument matches the value for specified filter data
func (e *MatchRegExExpr) Execute(data Data) (interface{}, error) {
	if e.Name == "" {
		for _, key := range data.GetKeys(e.FilterType) {
			if e.Arg.MatchString(key) {
				return true, nil
			}
		}
	}

	valueToMatch, ok := data.GetValue(e.FilterType, e.Name)
	if !ok {
		return false, nil
	}
	return e.Arg.MatchString(valueToMatch), nil
}

func (e *MatchRegExExpr) String() string {
	return e.FilterType + "." + e.Name + ".MatchRegEx(\"" + e.Arg.String() + "\")"
}
