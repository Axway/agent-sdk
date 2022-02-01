package log

import (
	"fmt"
	"regexp"
)

// ObscureArguments obscure/mask/redact values for a set of trailing arguments
func ObscureArguments(redactedFields []string, args ...interface{}) []interface{} {
	var obscuredParams []interface{}
	for _, arg := range args {
		obscuredParams = append(obscuredParams, obscureParams(fmt.Sprintf("%s", arg), redactedFields))
	}
	return obscuredParams

}

// obscureParams obscure/mask/redact a set of values in a json string
func obscureParams(jsn string, sensitiveParams []string) string {
	for _, param := range sensitiveParams {
		jsn = obscureParam(jsn, param)
	}
	return jsn
}

// obscureParam obscure/mask/redact a value in a json string
func obscureParam(jsn string, param string) string {
	rWithSlash := *regexp.MustCompile(`\\"` + param + `\\":.*?"(.*?)\\"`)
	jsn = rWithSlash.ReplaceAllString(jsn, `\"`+param+`\": \"**********\"`)

	rWithoutSlash := *regexp.MustCompile(`"` + param + `":.*?"(.*?)"`)
	return rWithoutSlash.ReplaceAllString(jsn, `"`+param+`": "**********"`)
}
