package apic

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// used by oas spec parsers to start the builder
func newSpecSecurityBuilder(oasMajorVersion int) SecurityBuilder {
	return &specSecurity{
		oasMajorVersion: oasMajorVersion,
	}
}

// first select the type of security we are building
type SecurityBuilder interface {
	IsHTTPBasic() HTTPBasicSecurityBuilder
	IsAPIKey() APIKeySecurityBuilder
	IsOAuth() OAuthSecurityBuilder
	IsBearer() BearerSecurityBuilder
	IsOpenID() OpenIDSecurityBuilder
}

type specSecurity struct {
	oasMajorVersion int
}

func (s *specSecurity) IsHTTPBasic() HTTPBasicSecurityBuilder {
	return &httpBasicSecurity{
		specSecurity: s,
	}
}

type HTTPBasicSecurityBuilder interface {
	Build() map[string]interface{}
}

type httpBasicSecurity struct {
	*specSecurity
}

// create http basic scheme
func (s *httpBasicSecurity) Build() map[string]interface{} {
	const (
		name  = "basicAuth"
		basic = "basic"
	)

	if s.oasMajorVersion == 2 {
		return map[string]interface{}{
			name: &openapi2.SecurityScheme{
				Type: basic,
			},
		}
	}
	return map[string]interface{}{
		name: &openapi3.SecurityScheme{
			Type:   "http",
			Scheme: basic,
		},
	}
}

func (s *specSecurity) IsAPIKey() APIKeySecurityBuilder {
	return &apiKeySecurity{
		specSecurity: s,
		locations:    []string{},
	}
}

type APIKeySecurityBuilder interface {
	Build() map[string]interface{}
	InCookie() APIKeySecurityBuilder // quests are the same for cookie vs api key in query or header
	InHeader() APIKeySecurityBuilder
	InQueryParam() APIKeySecurityBuilder
	SetArgumentName(argName string) APIKeySecurityBuilder
}

type apiKeySecurity struct {
	*specSecurity
	locations []string
	argName   string
}

func (s *apiKeySecurity) InCookie() APIKeySecurityBuilder {
	s.locations = append(s.locations, "cookie")
	return s
}

func (s *apiKeySecurity) InHeader() APIKeySecurityBuilder {
	s.locations = append(s.locations, "header")
	return s
}

func (s *apiKeySecurity) InQueryParam() APIKeySecurityBuilder {
	s.locations = append(s.locations, "query")
	return s
}

func (s *apiKeySecurity) SetArgumentName(argName string) APIKeySecurityBuilder {
	s.argName = argName
	return s
}

// create api key security type
func (s *apiKeySecurity) Build() map[string]interface{} {
	const apiKey = "apiKey"

	output := map[string]interface{}{}

	for _, location := range s.locations {
		name := "apiKeyHeader"
		if location == "query" {
			name = "apiKeyQueryParam"
		} else if location == "cookie" {
			name = "apiKeyCookie"
		}

		if s.oasMajorVersion == 2 {
			if location == "cookie" {
				// only supported on oas3, return empty for oas2
				continue
			}

			output[name] = &openapi2.SecurityScheme{
				Name: s.argName,
				In:   location,
				Type: apiKey,
			}
			continue
		}

		output[name] = &openapi3.SecurityScheme{
			Name: s.argName,
			In:   location,
			Type: apiKey,
		}
	}

	return output
}

type oAuthFlow struct {
	flow       string
	authURL    string
	tokenURL   string
	refreshURL string
	scopes     map[string]string
}

func NewOAuthFlowBuilder() OAuthFlowBuilder {
	return &oAuthFlow{}
}

// oauth flow options, setting flow type should be last, not all other methods are required
type OAuthFlowBuilder interface {
	SetScopes(map[string]string) OAuthFlowBuilder
	AddScope(scope, description string) OAuthFlowBuilder
	SetAuthorizationURL(url string) OAuthFlowBuilder
	SetRefreshURL(url string) OAuthFlowBuilder
	SetTokenURL(url string) OAuthFlowBuilder
	IsImplicit() *oAuthFlow
	IsPassword() *oAuthFlow
	IsAuthorizationCode() *oAuthFlow
	IsClientCredentials() *oAuthFlow
}

func (s *oAuthFlow) SetScopes(scopes map[string]string) OAuthFlowBuilder {
	s.scopes = scopes
	return s
}

func (s *oAuthFlow) AddScope(scope, description string) OAuthFlowBuilder {
	s.scopes[scope] = description
	return s
}

func (s *oAuthFlow) SetTokenURL(url string) OAuthFlowBuilder {
	s.tokenURL = url
	return s
}

func (s *oAuthFlow) SetAuthorizationURL(url string) OAuthFlowBuilder {
	s.authURL = url
	return s
}

func (s *oAuthFlow) SetRefreshURL(url string) OAuthFlowBuilder {
	s.refreshURL = url
	return s
}

func (s *oAuthFlow) IsImplicit() *oAuthFlow {
	s.flow = "implicit"
	return s
}

func (s *oAuthFlow) IsPassword() *oAuthFlow {
	s.flow = "password"
	return s
}

func (s *oAuthFlow) IsAuthorizationCode() *oAuthFlow {
	s.flow = "authorizationCode"
	return s
}

func (s *oAuthFlow) IsClientCredentials() *oAuthFlow {
	s.flow = "clientCredentials"
	return s
}

type OAuthSecurityBuilder interface {
	AddFlow(flow *oAuthFlow) OAuthSecurityBuilder
	Build() map[string]interface{}
}

type oAuthSecurity struct {
	*specSecurity
	flows []*oAuthFlow
}

func (s *specSecurity) IsOAuth() OAuthSecurityBuilder {
	return &oAuthSecurity{
		specSecurity: s,
		flows:        []*oAuthFlow{},
	}
}

func (s *oAuthSecurity) AddFlow(flow *oAuthFlow) OAuthSecurityBuilder {
	s.flows = append(s.flows, flow)
	return s
}

func (s *oAuthSecurity) Build() map[string]interface{} {
	const oauth2 = "oauth2"

	if s.oasMajorVersion == 2 {
		// create separate scheme for each flow type
		oauthFlows := map[string]interface{}{}
		for _, f := range s.flows {

			// adjust the name of the flow for oas2 support
			if f.flow == "authorizationCode" {
				f.flow = "accessCode"
			} else if f.flow == "clientCredentials" {
				f.flow = "application"
			}

			fName := fmt.Sprintf("%v%v", oauth2, cases.Title(language.English).String(f.flow))
			oauthFlows[fName] = &openapi2.SecurityScheme{
				Type:             oauth2,
				Flow:             f.flow,
				Scopes:           f.scopes,
				AuthorizationURL: f.authURL,
				TokenURL:         f.tokenURL,
			}
		}
		return oauthFlows
	}

	// create single scheme with all flows
	oauthFlows := &openapi3.OAuthFlows{}
	for _, f := range s.flows {
		oFlow := &openapi3.OAuthFlow{
			AuthorizationURL: f.authURL,
			RefreshURL:       f.refreshURL,
			TokenURL:         f.tokenURL,
			Scopes:           f.scopes,
		}
		switch f.flow {
		case "authorizationCode":
			oauthFlows.AuthorizationCode = oFlow
		case "password":
			oauthFlows.Password = oFlow
		case "clientCredentials":
			oauthFlows.ClientCredentials = oFlow
		case "implicit":
			oauthFlows.Implicit = oFlow
		}
	}
	return map[string]interface{}{
		oauth2: &openapi3.SecurityScheme{
			Type:  oauth2,
			Flows: oauthFlows,
		},
	}
}

func (s *specSecurity) IsBearer() BearerSecurityBuilder {
	return &bearerSecurity{
		specSecurity: s,
	}
}

type BearerSecurityBuilder interface {
	Build() map[string]interface{}
	SetFormat(format string) BearerSecurityBuilder
}

type bearerSecurity struct {
	*specSecurity
	format string
}

func (s *bearerSecurity) SetFormat(format string) BearerSecurityBuilder {
	s.format = format
	return s
}

func (s *bearerSecurity) Build() map[string]interface{} {
	const (
		name   = "bearerAuth"
		bearer = "bearer"
	)

	if s.oasMajorVersion == 2 {
		// only supported on oas3, return empty for oas2
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		name: &openapi3.SecurityScheme{
			Type:         "http",
			Scheme:       bearer,
			BearerFormat: s.format,
		},
	}
}

func (s *specSecurity) IsOpenID() OpenIDSecurityBuilder {
	return &openIDSecurity{
		specSecurity: s,
	}
}

type OpenIDSecurityBuilder interface {
	Build() map[string]interface{}
	SetURL(url string) OpenIDSecurityBuilder
}

type openIDSecurity struct {
	*specSecurity
	url string
}

func (s *openIDSecurity) SetURL(url string) OpenIDSecurityBuilder {
	s.url = url
	return s
}

func (s *openIDSecurity) Build() map[string]interface{} {
	const (
		name = "openId"
	)

	if s.oasMajorVersion == 2 {
		// only supported on oas3, return empty for oas2
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		name: &openapi3.SecurityScheme{
			Type:             "openIdConnect",
			OpenIdConnectUrl: s.url,
		},
	}
}
