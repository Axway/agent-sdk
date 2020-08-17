package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var name1Val = "value 1"
var name2Val = "value 2"
var name3Val = "value 3"
var name4Val = "value 4"

var filterData = map[string]string{
	"name1": name1Val,
	"name2": name2Val,
	"name3": name3Val,
	"name4": name4Val,
}

var filterDataWithStringPointer = map[string]*string{
	"name1": &name1Val,
	"name2": &name2Val,
	"name3": &name3Val,
	"name4": &name4Val,
}

var filterDataWithStringArrary = map[string][]string{
	"name1": {name1Val, "v 1", "v-1"},
	"name2": {name2Val, "v 2", "v-2"},
	"name3": {name3Val, "v 3", "v-3"},
	"name4": {name4Val, "v 4", "v-4"},
}

func TestSimpleFilter(t *testing.T) {
	assertFilter(t, "tag.name == \"value 1\"", filterData, false)
	assertFilter(t, "tag.name1 == \"value 1\"", filterData, true)
	assertFilter(t, "tag.name1 == \"value 1,v 1,v-1\"", filterDataWithStringArrary, true)
	assertFilter(t, "tag.name1 != \"value 1\"", filterData, false)

	assertFilter(t, "tag.Any() == \"value 1\"", filterDataWithStringPointer, true)
	assertFilter(t, "tag.Any() == \"somevalue\"", filterDataWithStringPointer, false)
	assertFilter(t, "tag.Any() != \"value 1\"", filterDataWithStringPointer, false)
	assertFilter(t, "tag.Any() != \"somevalue\"", filterDataWithStringPointer, true)

	assertFilter(t, "tag.name1.Exists()", filterData, true)
	assertFilter(t, "tag.name.Exists()", filterData, false)
	assertFilter(t, "tag.name1.Exists() == false", filterData, false)
	assertFilter(t, "tag.name.Exists() != true", filterData, true)

	assertFilter(t, "tag.name1.Contains(\"val\")", filterData, true)
	assertFilter(t, "tag.name1.Contains(\"someval\")", filterData, false)
	assertFilter(t, "tag.name1.Contains(\"val\") == true", filterData, true)
	assertFilter(t, "tag.name1.Contains(\"val\") != true", filterData, false)

	assertFilter(t, "tag.name1.Contains(\"v-1\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.name1.Contains(\"value 1,v 1\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.name2.Contains(\"value 1,v 1\")", filterDataWithStringArrary, false)

	assertFilter(t, "tag.name1.MatchRegEx(\"(val){1}\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.name1.MatchRegEx(\"(na){1}\")", filterDataWithStringArrary, false)
	assertFilter(t, "tag.name1.MatchRegEx(\".*(value 1|value 2).*\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.name2.MatchRegEx(\".*(value 1|value 2).*\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.name3.MatchRegEx(\".*(value 1|value 2).*\")", filterDataWithStringArrary, false)

	assertFilter(t, "tag.name1.MatchRegEx(\"(val){1}\") == true", filterDataWithStringArrary, true)
	assertFilter(t, "tag.name1.MatchRegEx(\"(val){1}\") != true", filterDataWithStringArrary, false)
}

func TestCompundFilter(t *testing.T) {
	assertFilter(t, "tag.name1 == \"value 1\" || tag.name2 == \"value 2\"", filterData, true)
	assertFilter(t, "tag.name1 == \"missing\" || tag.name2 == \"value 2\"", filterData, true)
	assertFilter(t, "tag.name1 == \"missing\" || tag.name2 == \"missing\"", filterData, false)

	assertFilter(t, "tag.name1 == \"value 1\" && tag.name2 == \"value 2\"", filterData, true)
	assertFilter(t, "tag.name1 == \"missing\" && tag.name2 == \"value 2\"", filterData, false)
	assertFilter(t, "tag.name1 == \"missing\" && tag.name2 == \"missing\"", filterData, false)

	assertFilter(t, "tag.Any() == \"value 1\" || tag.name2 == \"missing\" && tag.name2 == \"missing\"", filterData, true)
	assertFilter(t, "tag.Any() == \"missing\" || tag.name2 == \"missing\" && tag.name2 == \"missing\"", filterData, false)

	assertFilter(t, "tag.name1.Exists() || tag.name2 == \"value 2\"", filterData, true)
	assertFilter(t, "tag.name1.Exists() && tag.name2 == \"value 2\"", filterData, true)
	assertFilter(t, "tag.name1.Exists() || tag.name2 == \"value 1\"", filterData, true)
	assertFilter(t, "tag.name1.Exists() == false && tag.name2 == \"value 2\"", filterData, false)
	assertFilter(t, "tag.name.Exists() != true && tag.Any() == \"value 2\"", filterData, true)

	assertFilter(t, "tag.name1.Exists() && tag.name1.Contains(\"val\")", filterData, true)
	assertFilter(t, "tag.name1.Exists() && tag.name1.Contains(\"something\")", filterData, false)
	assertFilter(t, "tag.name.Exists() && tag.name1.Contains(\"val\")", filterData, false)

	assertFilter(t, "tag.Any() == \"value 1,v 1,v-1\" || tag.name2.Exists() || tag.name3.Contains(\"v-3\") || tag.name4.MatchRegEx(\"(val){1}\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.Any() == \"someotherval\" || tag.name2.Exists() || tag.name3.Contains(\"v-3\") || tag.name4.MatchRegEx(\"(val){1}\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.Any() == \"someotherval\" || tag.name22.Exists() || tag.name3.Contains(\"v-3\") || tag.name4.MatchRegEx(\"(val){1}\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.Any() == \"someotherval\" || tag.name22.Exists() || tag.name3.Contains(\"somevalue\") || tag.name4.MatchRegEx(\"(val){1}\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.Any() == \"someotherval\" || tag.name22.Exists() || tag.name3.Contains(\"somevalue\") || tag.name4.MatchRegEx(\"(something){1}\")", filterDataWithStringArrary, false)

	assertFilter(t, "tag.Any() == \"value 1,v 1,v-1\" && tag.name2.Exists() && tag.name3.Contains(\"v-3\") && tag.name4.MatchRegEx(\"(val){1}\")", filterDataWithStringArrary, true)
	assertFilter(t, "tag.Any() == \"someotherval\" || tag.name2.Exists() && tag.name3.Contains(\"v-3\") || tag.name4.MatchRegEx(\"(val){1}\")", filterDataWithStringArrary, true)
}

func assertFilter(t *testing.T, filterConfig string, filterData interface{}, expectedResult bool) {
	agentFilter, err := NewFilter(filterConfig)
	assert.NotNil(t, agentFilter)
	assert.Nil(t, err)
	b := agentFilter.Evaluate(filterData)
	assert.Equal(t, expectedResult, b)
}

func TestFilterParsingError(t *testing.T) {
	// golang Syntax OK, but have filter syntax errors
	assertFilterSyntaxErr(t, "a == b", "Unrecognized condition")
	assertFilterSyntaxErr(t, "something.name1 == \"value 1\"", "Invalid selector type")
	assertFilterSyntaxErr(t, "tag.name1", "Unrecognized expression")
	assertFilterSyntaxErr(t, "\"value\" == \"value\"", "Unrecognized condition")
	assertFilterSyntaxErr(t, "tag.name1 & \"value\"", "Invalid operator")

	// Unsupported condition grouping
	assertFilterSyntaxErr(t, "tag.name1 == \"value 1\" && (tag.name1 == \"value 1\")", "Unrecognized expression")
	assertFilterSyntaxErr(t, "\"tag.name1 == value\"", "Unrecognized expression")

	// Syntax Errors
	assertFilterSyntaxErr(t, "tag.name1 = \"value 1\"", "Syntax error: expected '==', found '='")
	assertFilterSyntaxErr(t, "tag.name1 == \"value 1\" tag.name1 != \"value 1\"", "Syntax error: expected ';', found tag")
	assertFilterSyntaxErr(t, "tag.name1 == ", "Syntax error: expected ';', found 'EOF'")

	// Call Expression Sytax Error
	assertFilterSyntaxErr(t, "tag.Match(\"something\")", "Unsupported call")

	// Additional arguments
	assertFilterSyntaxErr(t, "tag.Any(\"something\") == \"something\"", "Syntax Error, unrecognized argument(s)")
	assertFilterSyntaxErr(t, "tag.name.Exists(\"something\")", "Syntax Error, unrecognized argument(s)")
	assertFilterSyntaxErr(t, "tag.name.Contains(\"one\", \"two\")", "Syntax Error, unrecognized argument(s)")
	assertFilterSyntaxErr(t, "tag.name.MatchRegEx(\"one\", \"two\")", "Syntax Error, unrecognized argument(s)")

	// Missing arguments
	assertFilterSyntaxErr(t, "tag.name.Contains()", "Syntax Error, missing argument")
	assertFilterSyntaxErr(t, "tag.name.MatchRegEx()", "Syntax Error, missing argument")

	// Invalid Regular expression
	assertFilterSyntaxErr(t, "tag.name.MatchRegEx(\".*[\")", "Invalid regular expression")

}

func assertFilterSyntaxErr(t *testing.T, filterConfig, expectedErr string) {
	agentFilter, err := NewFilter(filterConfig)
	assert.Nil(t, agentFilter)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), expectedErr)
}
