package apic

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeySecuritySchemeBuilder(t *testing.T) {
	// OAS2
	b := newSpecSecurityBuilder(2)

	// api key builder
	oas2Schemes := b.APIKey().InCookie().InHeader().InQueryParam().SetArgumentName("api-key").Build()
	assert.Len(t, oas2Schemes, 2)
	assert.NotContains(t, oas2Schemes, "apiKeyCookie")
	oas2Header, ok := oas2Schemes["apiKeyHeader"].(*openapi2.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi2.SecurityScheme type")
	}
	assert.Equal(t, "apiKey", oas2Header.Type)
	assert.Equal(t, "header", oas2Header.In)
	assert.Equal(t, "api-key", oas2Header.Name)
	oas2Query, ok := oas2Schemes["apiKeyQuery"].(*openapi2.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi2.SecurityScheme type")
	}
	assert.Equal(t, "apiKey", oas2Query.Type)
	assert.Equal(t, "query", oas2Query.In)
	assert.Equal(t, "api-key", oas2Query.Name)

	// OAS3
	b = newSpecSecurityBuilder(3)

	// api key builder
	oas3Schemes := b.APIKey().InCookie().InHeader().InQueryParam().SetArgumentName("api-key").Build()
	assert.Len(t, oas3Schemes, 3)
	assert.Contains(t, oas3Schemes, "apiKeyCookie")
	oas3Header, ok := oas3Schemes["apiKeyHeader"].(*openapi3.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi3.SecurityScheme type")
	}
	assert.Equal(t, "apiKey", oas3Header.Type)
	assert.Equal(t, "header", oas3Header.In)
	assert.Equal(t, "api-key", oas3Header.Name)
	oas3Query, ok := oas3Schemes["apiKeyQuery"].(*openapi3.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi3.SecurityScheme type")
	}
	assert.Equal(t, "apiKey", oas3Query.Type)
	assert.Equal(t, "query", oas3Query.In)
	assert.Equal(t, "api-key", oas3Query.Name)
	oas3Cookie, ok := oas3Schemes["apiKeyCookie"].(*openapi3.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi3.SecurityScheme type")
	}
	assert.Equal(t, "apiKey", oas3Cookie.Type)
	assert.Equal(t, "cookie", oas3Cookie.In)
	assert.Equal(t, "api-key", oas3Cookie.Name)
}

func TestHTTPBasicSecuritySchemeBuilder(t *testing.T) {
	// OAS2
	b := newSpecSecurityBuilder(2)

	oas2Schemes := b.HTTPBasic().Build()
	assert.Len(t, oas2Schemes, 1)
	oas2Basic, ok := oas2Schemes["basicAuth"].(*openapi2.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi2.SecurityScheme type")
	}
	assert.Equal(t, "basic", oas2Basic.Type)

	// OAS3
	b = newSpecSecurityBuilder(3)

	oas3Schemes := b.HTTPBasic().Build()
	assert.Len(t, oas3Schemes, 1)
	oas3Basic, ok := oas3Schemes["basicAuth"].(*openapi3.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi3.SecurityScheme type")
	}
	assert.Equal(t, "http", oas3Basic.Type)
	assert.Equal(t, "basic", oas3Basic.Scheme)
}

func TestBearerSecuritySchemeBuilder(t *testing.T) {
	// OAS2
	b := newSpecSecurityBuilder(2)

	oas2Schemes := b.Bearer().SetFormat("jwt").Build()
	assert.Len(t, oas2Schemes, 0)

	// OAS3
	b = newSpecSecurityBuilder(3)

	oas3Schemes := b.Bearer().SetFormat("jwt").Build()
	assert.Len(t, oas3Schemes, 1)
	oas3Bearer, ok := oas3Schemes["bearerAuth"].(*openapi3.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi3.SecurityScheme type")
	}
	assert.Equal(t, "http", oas3Bearer.Type)
	assert.Equal(t, "bearer", oas3Bearer.Scheme)
	assert.Equal(t, "jwt", oas3Bearer.BearerFormat)
}

func TestOpenIDSecuritySchemeBuilder(t *testing.T) {
	// OAS2
	b := newSpecSecurityBuilder(2)

	oas2Schemes := b.OpenID().SetURL("http://test.com").Build()
	assert.Len(t, oas2Schemes, 0)

	// OAS3
	b = newSpecSecurityBuilder(3)

	oas3Schemes := b.OpenID().SetURL("http://test.com").Build()
	assert.Len(t, oas3Schemes, 1)
	oas3Bearer, ok := oas3Schemes["openId"].(*openapi3.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi3.SecurityScheme type")
	}
	assert.Equal(t, "openIdConnect", oas3Bearer.Type)
	assert.Equal(t, "http://test.com", oas3Bearer.OpenIdConnectUrl)
}

func TestOAuthSecuritySchemeBuilder(t *testing.T) {
	// OAS2
	b := newSpecSecurityBuilder(2)

	oas2Schemes := b.OAuth().
		AddFlow(NewOAuthFlowBuilder().
			SetScopes(map[string]string{"scope1": ""}).
			AddScope("scope2", "").
			SetAuthorizationURL("http://authurl.com").
			Implicit()).
		AddFlow(NewOAuthFlowBuilder().
			SetScopes(map[string]string{"scope1": ""}).
			AddScope("scope2", "").
			SetTokenURL("http://tokenurl.com").
			ClientCredentials()).
		AddFlow(NewOAuthFlowBuilder().
			SetScopes(map[string]string{"scope1": ""}).
			AddScope("scope2", "").
			SetAuthorizationURL("http://authurl.com").
			SetTokenURL("http://tokenurl.com").
			AuthorizationCode()).
		AddFlow(NewOAuthFlowBuilder().
			SetScopes(map[string]string{"scope1": ""}).
			AddScope("scope2", "").
			SetTokenURL("http://tokenurl.com").
			Password()).
		Build()
	assert.Len(t, oas2Schemes, 4)

	oas2Implicit, ok := oas2Schemes["oauth2Implicit"].(*openapi2.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi2.SecurityScheme type")
	}
	assert.Equal(t, "oauth2", oas2Implicit.Type)
	assert.Equal(t, "implicit", oas2Implicit.Flow)
	assert.Equal(t, "http://authurl.com", oas2Implicit.AuthorizationURL)
	assert.Len(t, oas2Implicit.Scopes, 2)

	oas2Application, ok := oas2Schemes["oauth2Application"].(*openapi2.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi2.SecurityScheme type")
	}
	assert.Equal(t, "oauth2", oas2Application.Type)
	assert.Equal(t, "application", oas2Application.Flow)
	assert.Equal(t, "http://tokenurl.com", oas2Application.TokenURL)
	assert.Len(t, oas2Application.Scopes, 2)

	oas2AccessCode, ok := oas2Schemes["oauth2Accesscode"].(*openapi2.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi2.SecurityScheme type")
	}
	assert.Equal(t, "oauth2", oas2AccessCode.Type)
	assert.Equal(t, "accessCode", oas2AccessCode.Flow)
	assert.Equal(t, "http://authurl.com", oas2AccessCode.AuthorizationURL)
	assert.Equal(t, "http://tokenurl.com", oas2AccessCode.TokenURL)
	assert.Len(t, oas2AccessCode.Scopes, 2)

	oas2Password, ok := oas2Schemes["oauth2Password"].(*openapi2.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi2.SecurityScheme type")
	}
	assert.Equal(t, "oauth2", oas2Password.Type)
	assert.Equal(t, "password", oas2Password.Flow)
	assert.Equal(t, "http://tokenurl.com", oas2Password.TokenURL)
	assert.Len(t, oas2Password.Scopes, 2)

	// OAS3
	b = newSpecSecurityBuilder(3)

	oas3Schemes := b.OAuth().
		AddFlow(NewOAuthFlowBuilder().
			SetScopes(map[string]string{"scope1": ""}).
			AddScope("scope2", "").
			SetAuthorizationURL("http://authurl.com").
			SetRefreshURL("http://refreshurl.com").
			Implicit()).
		AddFlow(NewOAuthFlowBuilder().
			SetScopes(map[string]string{"scope1": ""}).
			AddScope("scope2", "").
			SetRefreshURL("http://refreshurl.com").
			SetTokenURL("http://tokenurl.com").
			ClientCredentials()).
		AddFlow(NewOAuthFlowBuilder().
			SetScopes(map[string]string{"scope1": ""}).
			AddScope("scope2", "").
			SetAuthorizationURL("http://authurl.com").
			SetRefreshURL("http://refreshurl.com").
			SetTokenURL("http://tokenurl.com").
			AuthorizationCode()).
		AddFlow(NewOAuthFlowBuilder().
			SetScopes(map[string]string{"scope1": ""}).
			AddScope("scope2", "").
			SetRefreshURL("http://refreshurl.com").
			SetTokenURL("http://tokenurl.com").
			Password()).
		Build()
	assert.Len(t, oas3Schemes, 1)

	oas3Auth, ok := oas3Schemes["oauth2"].(*openapi3.SecurityScheme)
	if !ok {
		assert.FailNow(t, "interface was not an *openapi3.SecurityScheme type")
	}

	assert.Equal(t, "oauth2", oas3Auth.Type)
	assert.Equal(t, "http://authurl.com", oas3Auth.Flows.Implicit.AuthorizationURL)
	assert.Equal(t, "http://refreshurl.com", oas3Auth.Flows.Implicit.RefreshURL)
	assert.Len(t, oas3Auth.Flows.Implicit.Scopes, 2)

	assert.Equal(t, "http://authurl.com", oas3Auth.Flows.AuthorizationCode.AuthorizationURL)
	assert.Equal(t, "http://tokenurl.com", oas3Auth.Flows.AuthorizationCode.TokenURL)
	assert.Equal(t, "http://refreshurl.com", oas3Auth.Flows.AuthorizationCode.RefreshURL)
	assert.Len(t, oas3Auth.Flows.AuthorizationCode.Scopes, 2)

	assert.Equal(t, "http://tokenurl.com", oas3Auth.Flows.Password.TokenURL)
	assert.Equal(t, "http://refreshurl.com", oas3Auth.Flows.Password.RefreshURL)
	assert.Len(t, oas3Auth.Flows.Password.Scopes, 2)

	assert.Equal(t, "http://tokenurl.com", oas3Auth.Flows.ClientCredentials.TokenURL)
	assert.Equal(t, "http://refreshurl.com", oas3Auth.Flows.ClientCredentials.RefreshURL)
	assert.Len(t, oas3Auth.Flows.ClientCredentials.Scopes, 2)
}
