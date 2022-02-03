package log

import (
	"encoding/json"
	"regexp"
)

// ObscureArguments obscure/mask/redact values for a set of trailing arguments
func ObscureArguments(redactedFields []string, args ...interface{}) []interface{} {
	var obscuredParams []interface{}
	for _, arg := range args {
		b, _ := json.Marshal(arg)
		obscuredParams = append(obscuredParams, obscureParams(string(b), redactedFields))
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

	rWithNumberSlash := *regexp.MustCompile(`"` + param + `": ?([0-9]+)`)
	jsn = rWithNumberSlash.ReplaceAllString(jsn, `"`+param+`": "[redacted]"`)

	rWithNumberNoSlash := *regexp.MustCompile(`\\"` + param + `\\":.*?([0-9]+)`)
	jsn = rWithNumberNoSlash.ReplaceAllString(jsn, `"`+param+`": "[redacted]"`)

	rWithSlash := *regexp.MustCompile(`\\"` + param + `\\":.*?"(.*?)\\"`)
	jsn = rWithSlash.ReplaceAllString(jsn, `\"`+param+`\": "[redacted]"`)

	rWithoutSlash := *regexp.MustCompile(`"` + param + `":.*?"(.*?)"`)
	return rWithoutSlash.ReplaceAllString(jsn, `"`+param+`": "[redacted]"`)
}
