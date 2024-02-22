package apic

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	oas2              = 2
	oas3              = 3
	httpScheme        = "http"
	cookie            = "cookie"
	header            = "header"
	query             = "query"
	implicit          = "implicit"
	authorizationCode = "authorizationCode"
	clientCredentials = "clientCredentials"
	password          = "password"
	accessCode        = "accessCode"
	application       = "application"
)

// used by oas spec parsers to start the builder
func newSpecSecurityBuilder(oasMajorVersion int) SecurityBuilder {
	return &specSecurity{
		oasMajorVersion: oasMajorVersion,
	}
}

// first select the type of security we are building
type SecurityBuilder interface {
	HTTPBasic() HTTPBasicSecurityBuilder
	APIKey() APIKeySecurityBuilder
	OAuth() OAuthSecurityBuilder
	Bearer() BearerSecurityBuilder
	OpenID() OpenIDSecurityBuilder
}

type specSecurity struct {
	oasMajorVersion int
}

func (s *specSecurity) HTTPBasic() HTTPBasicSecurityBuilder {
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
			Type:   httpScheme,
			Scheme: basic,
		},
	}
}

func (s *specSecurity) APIKey() APIKeySecurityBuilder {
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
	s.locations = append(s.locations, cookie)
	return s
}

func (s *apiKeySecurity) InHeader() APIKeySecurityBuilder {
	s.locations = append(s.locations, header)
	return s
}

func (s *apiKeySecurity) InQueryParam() APIKeySecurityBuilder {
	s.locations = append(s.locations, query)
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
		name := fmt.Sprintf("%v%v", apiKey, cases.Title(language.English).String(location))

		if s.oasMajorVersion == 2 {
			if location == cookie {
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
	Implicit() *oAuthFlow
	Password() *oAuthFlow
	AuthorizationCode() *oAuthFlow
	ClientCredentials() *oAuthFlow
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

func (s *oAuthFlow) Implicit() *oAuthFlow {
	s.flow = implicit
	return s
}

func (s *oAuthFlow) Password() *oAuthFlow {
	s.flow = password
	return s
}

func (s *oAuthFlow) AuthorizationCode() *oAuthFlow {
	s.flow = authorizationCode
	return s
}

func (s *oAuthFlow) ClientCredentials() *oAuthFlow {
	s.flow = clientCredentials
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

func (s *specSecurity) OAuth() OAuthSecurityBuilder {
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
			if f.flow == authorizationCode {
				f.flow = accessCode
			} else if f.flow == clientCredentials {
				f.flow = application
			}

			fName := fmt.Sprintf("%v%v", oauth2, cases.Title(language.English).String(f.flow))
			oauthFlows[fName] = &openapi2.SecurityScheme{
				Type:             oauth2,
				Flow:             f.flow,
				Scopes:           util.OrderStringsInMap(f.scopes),
				AuthorizationURL: f.authURL,
				TokenURL:         f.tokenURL,
			}
		}
		return util.OrderStringsInMap(oauthFlows)
	}

	// create single scheme with all flows
	oauthFlows := &openapi3.OAuthFlows{}
	for _, f := range s.flows {
		oFlow := &openapi3.OAuthFlow{
			AuthorizationURL: f.authURL,
			RefreshURL:       f.refreshURL,
			TokenURL:         f.tokenURL,
			Scopes:           util.OrderStringsInMap(f.scopes),
		}
		switch f.flow {
		case authorizationCode:
			oauthFlows.AuthorizationCode = oFlow
		case password:
			oauthFlows.Password = oFlow
		case clientCredentials:
			oauthFlows.ClientCredentials = oFlow
		case implicit:
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

func (s *specSecurity) Bearer() BearerSecurityBuilder {
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
			Type:         httpScheme,
			Scheme:       bearer,
			BearerFormat: s.format,
		},
	}
}

func (s *specSecurity) OpenID() OpenIDSecurityBuilder {
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
		name          = "openId"
		openIdConnect = "openIdConnect"
	)

	if s.oasMajorVersion == 2 {
		// only supported on oas3, return empty for oas2
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		name: &openapi3.SecurityScheme{
			Type:             openIdConnect,
			OpenIdConnectUrl: s.url,
		},
	}
}
