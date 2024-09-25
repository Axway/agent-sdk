package filter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
)

var (
	dashMatchReg   = regexp.MustCompile(`tag\.MatchRegEx\("([\w+\.\*]+-)+[\w+\.\*]+"\)`)
	dashTagNameReg = regexp.MustCompile(`tag\.(\w+-)+\w+`)
)

// ConditionParser - Represents the filter condition parser
type ConditionParser struct {
	err           error
	newConditions []Condition
}

// NewConditionParser - Create a new instance of condition parser
func NewConditionParser() *ConditionParser {
	return &ConditionParser{
		newConditions: make([]Condition, 0),
	}
}

// Parse - parses the AST tree to filter condition
func (f *ConditionParser) Parse(filterConfig string) ([]Condition, error) {
	parsedConditions, err := f.parseCondition(strings.TrimSpace(f.preProcessCondition(filterConfig)))
	if err != nil {
		return nil, err
	}

	return parsedConditions, nil
}

func (f *ConditionParser) preProcessCondition(filterCondition string) string {
	filterCondition = applyDashPlaceholder(dashMatchReg, filterCondition)
	return applyDashPlaceholder(dashTagNameReg, filterCondition)
}

func applyDashPlaceholder(re *regexp.Regexp, filterCondition string) string {
	return re.ReplaceAllStringFunc(filterCondition, func(s string) string {
		return strings.ReplaceAll(s, Dash, DashPlaceHolder)
	})
}

func (f *ConditionParser) parseCondition(filterCodition string) ([]Condition, error) {
	if filterCodition == "" {
		return nil, nil
	}
	src := "package main\nvar b bool = " + filterCodition
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "filter_config", []byte(src), parser.AllErrors)
	// ast.Fprint(os.Stdout, fset, node, nil)
	if err != nil {
		errSegments := strings.Split(err.Error(), ":")
		errMsg := errSegments[len(errSegments)-1]
		return nil, ErrFilterConfiguration.FormatError(errMsg)
	}

	ast.Inspect(node, f.parseConditionExpr)
	if f.err != nil {
		return nil, f.err
	}
	return f.newConditions, f.err
}

func (f *ConditionParser) parseConditionExpr(node ast.Node) bool {
	valueSpec, ok := node.(*ast.ValueSpec)
	if !ok {
		return true
	}
	for _, valueEx := range valueSpec.Values {
		newCondition, err := f.parseExpr(valueEx)
		f.newConditions = append(f.newConditions, newCondition)
		f.err = err
		if f.err != nil {
			return false
		}
	}
	return true
}

func (f *ConditionParser) isConditionalExpr(expr *ast.BinaryExpr) bool {
	op := expr.Op
	opPrecendence := op.Precedence()
	return (opPrecendence <= 3)
}

func (f *ConditionParser) isSimpleExpr(expr *ast.BinaryExpr) bool {
	op := expr.Op
	opPrecendence := op.Precedence()
	return (opPrecendence == 3)
}

func (f *ConditionParser) parseExpr(expr ast.Expr) (Condition, error) {
	bexpr, ok := expr.(*ast.BinaryExpr)
	if ok {
		return f.parseBinaryExpr(bexpr)
	} else if callExpr, ok := expr.(*ast.CallExpr); ok {
		ce, err := f.parseCallExpr(callExpr)
		if err != nil {
			return nil, err
		}
		return &SimpleCondition{
			LHSExpr: ce,
		}, nil
	}
	return nil, ErrFilterExpression
}

func (f *ConditionParser) parseCallExpr(expr *ast.CallExpr) (CallExpr, error) {
	funcSelectorExprt, ok := expr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, ErrFilterExpression
	}
	funcSelector, err := f.parseSelectorExpr(funcSelectorExprt)
	if err != nil {
		return nil, err
	}
	selectorType, selector, err := f.parseSelector(funcSelector)
	if err != nil {
		return nil, err
	}
	funcName := selector[strings.LastIndex(selector, ".")+1:]

	callType, err := GetCallType(funcName)
	if err != nil {
		return nil, utilerrors.Wrap(ErrFilterGeneralParse, err.Error())
	}

	var callArguments []interface{}
	if expr.Args != nil {
		callArguments, err = f.parseCallArguments(expr.Args)
		if err != nil {
			return nil, err
		}
	}

	name := ""
	lastSelectorIndex := strings.LastIndex(selector, ".")
	if lastSelectorIndex != -1 {
		name = selector[:lastSelectorIndex]
	}
	callExpr, err := newCallExpr(callType, selectorType, name, callArguments)
	if err != nil {
		return nil, utilerrors.Wrap(ErrFilterGeneralParse, err.Error())
	}
	return callExpr, nil
}

func (f *ConditionParser) parseCallArguments(args []ast.Expr) ([]interface{}, error) {
	argsList := make([]interface{}, 0)
	for _, argExpr := range args {
		literal, ok := argExpr.(*ast.BasicLit)
		if !ok {
			return nil, ErrFilterArgument
		}
		arg := strings.Trim(literal.Value, `"`)
		argsList = append(argsList, arg)
	}
	return argsList, nil
}

func (f *ConditionParser) parseSelector(selector string) (selectorType, selectorPath string, err error) {
	selectorType = selector[0:strings.Index(selector, ".")]
	selectorPath = selector[strings.Index(selector, ".")+1:]
	if selectorType != filterTypeTag && selectorType != filterTypeAttr {
		err = ErrFilterSelectorType
	}
	return
}

func (f *ConditionParser) parseSelectorExpr(expr *ast.SelectorExpr) (string, error) {
	var xName string
	var err error
	x, ok := expr.X.(*ast.Ident)
	if ok {
		xName = x.Name
	} else if x, ok := expr.X.(*ast.SelectorExpr); ok {
		xName, err = f.parseSelectorExpr(x)
	} else {
		err = ErrFilterSelectorExpr
	}
	if err != nil {
		return "", err
	}

	return xName + "." + expr.Sel.Name, nil
}

func (f *ConditionParser) parseBinaryExpr(expr *ast.BinaryExpr) (Condition, error) {
	if !f.isConditionalExpr(expr) {
		return nil, ErrFilterOperator
	}

	if f.isSimpleExpr(expr) {
		return f.parseSimpleBinaryExpr(expr)
	}
	return f.parseCompoundBinaryExpr(expr)
}

func (f *ConditionParser) parseSimpleLHS(expr *ast.BinaryExpr) (CallExpr, error) {
	lhs, ok := expr.X.(*ast.SelectorExpr)
	if ok {
		return f.parseSelectorLHS(lhs)
	} else if lhs, ok := expr.X.(*ast.CallExpr); ok {
		return f.parseCallLHS(lhs)
	}
	return nil, ErrFilterCondition
}

func (f *ConditionParser) parseSelectorLHS(lhs *ast.SelectorExpr) (CallExpr, error) {
	s, err := f.parseSelectorExpr(lhs)
	if err != nil {
		return nil, err
	}
	filterType, filterName, err := f.parseSelector(s)
	if err != nil {
		return nil, err
	}
	return newCallExpr(GETVALUE, filterType, filterName, nil)
}

func (f *ConditionParser) parseCallLHS(lhs *ast.CallExpr) (CallExpr, error) {
	ce, err := f.parseCallExpr(lhs)
	if err != nil {
		return nil, err
	}
	return ce, nil
}

func (f *ConditionParser) parseSimpleRHS(expr *ast.BinaryExpr) (filterValue ComparableValue) {
	literal, ok := expr.Y.(*ast.BasicLit)
	if ok {
		filterValue = newStringRHSValue(strings.Trim(literal.Value, `"`))
	} else if identVal, ok := expr.Y.(*ast.Ident); ok {
		filterValue = newStringRHSValue(identVal.Name)
	}
	return
}

func (f *ConditionParser) parseSimpleBinaryExpr(expr *ast.BinaryExpr) (Condition, error) {
	filterNode := &SimpleCondition{
		Operator: expr.Op.String(),
	}
	var err error

	filterNode.LHSExpr, err = f.parseSimpleLHS(expr)
	if err != nil {
		return nil, err
	}
	filterNode.Value = f.parseSimpleRHS(expr)
	return filterNode, nil
}

func (f *ConditionParser) parseCompoundBinaryExpr(expr *ast.BinaryExpr) (Condition, error) {
	filterNode := &CompoundCondition{
		Operator: expr.Op.String(),
	}

	var err error
	filterNode.LHSCondition, err = f.parseExpr(expr.X)
	if err != nil {
		return nil, err
	}
	filterNode.RHSCondition, err = f.parseExpr(expr.Y)
	if err != nil {
		return nil, err
	}
	return filterNode, nil
}
