package apic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

var methods = [5]string{"get", "post", "put", "patch", "delete"} // RestAPI methods

const (
	apikey      = "verify-api-key"
	passthrough = "pass-through"
	oauth       = "verify-oauth-token"
)

func determineAuthPolicyFromSwagger(swagger *[]byte) string {
	// Traverse the swagger looking for any route that has security set
	// return the security of the first route, if none- found return passthrough
	var authPolicy = passthrough

	gjson.GetBytes(*swagger, "paths").ForEach(func(_, pathObj gjson.Result) bool {
		for _, method := range methods {
			if pathObj.Get(fmt.Sprint(method, ".security.#.api_key")).Exists() {
				authPolicy = apikey
				return false
			}
			if pathObj.Get(fmt.Sprint(method, ".securityDefinitions.OAuthImplicit")).Exists() {
				authPolicy = oauth
				return false
			}
		}
		return authPolicy == passthrough // Return from path loop anonymous func, true = go to next item
	})

	if gjson.GetBytes(*swagger, "securityDefinitions.OAuthImplicit").Exists() {
		authPolicy = oauth
	}

	return authPolicy
}

func TestSanitizeAPIName(t *testing.T) {
	name := sanitizeAPIName("Abc.Def")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName(".Abc.Def")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName(".Abc...De/f")
	assert.Equal(t, "abc--.de-f", name)
	name = sanitizeAPIName("Abc.D-ef")
	assert.Equal(t, "abc.d-ef", name)
	name = sanitizeAPIName("Abc.Def=")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName("A..bc.Def")
	assert.Equal(t, "a--bc.def", name)
}
