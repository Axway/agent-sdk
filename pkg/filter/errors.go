package filter

import "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"

// Errors hit while parsing the APIMANAGER_FILTER
var (
	ErrFilterConfiguration = errors.Newf(1147, "error parsing filter in configuration. Syntax error:%s")
	ErrFilterExpression    = errors.New(1148, "error parsing filter in configuration. Unrecognized expression")
	ErrFilterGeneralParse  = errors.New(1149, "error parsing filter in configuration")
	ErrFilterArgument      = errors.New(1150, "error parsing filter in configuration. Invalid call argument")
	ErrFilterSelectorType  = errors.New(1151, "error parsing filter in configuration. Invalid selector type")
	ErrFilterSelectorExpr  = errors.New(1152, "error parsing filter in configuration. Invalid selector expression")
	ErrFilterOperator      = errors.New(1153, "error parsing filter in configuration. Invalid operator")
	ErrFilterCondition     = errors.New(1154, "error parsing filter in configuration. Unrecognized condition")
)
