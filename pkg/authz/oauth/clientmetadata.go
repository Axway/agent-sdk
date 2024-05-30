package oauth

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

// Time - time
type Time time.Time

// MarshalJSON - serialize time to unix timestamp
func (t *Time) MarshalJSON() ([]byte, error) {
	tt := (time.Time(*t)).Unix()
	return json.Marshal(tt)
}

// UnmarshalJSON - deserialize time to unix timestamp
func (t *Time) UnmarshalJSON(data []byte) error {
	var tt int64
	json.Unmarshal(data, &tt)
	*t = Time(time.Unix(tt, 0))
	return nil
}

// ClientMetadata - Interface for IdP client metadata representation
type ClientMetadata interface {
	GetClientName() string
	GetClientID() string
	GetClientSecret() string
	GetClientIDIssuedAt() *time.Time
	GetClientSecretExpiresAt() *time.Time
	GetScopes() []string
	GetGrantTypes() []string
	GetTokenEndpointAuthMethod() string
	GetResponseTypes() []string
	GetClientURI() string
	GetRedirectURIs() []string
	GetLogoURI() string
	GetJwksURI() string
	GetJwks() map[string]interface{}
	GetExtraProperties() map[string]string
	GetTLSClientAuthSanDNS() string
	GetTLSClientAuthSanEmail() string
	GetTLSClientAuthSanIP() string
	GetTLSClientAuthSanURI() string
	GetRegistrationAccessToken() string
}

type clientMetadata struct {
	ClientName            string `json:"client_name,omitempty"`
	ClientID              string `json:"client_id,omitempty"`
	ClientSecret          string `json:"client_secret,omitempty"`
	ClientIDIssuedAt      *Time  `json:"client_id_issued_at,omitempty"`
	ClientSecretExpiresAt *Time  `json:"client_secret_expires_at,omitempty"`

	Scope Scopes `json:"scope,omitempty"`

	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`

	ClientURI               string                 `json:"client_uri,omitempty"`
	RedirectURIs            []string               `json:"redirect_uris,omitempty"`
	JwksURI                 string                 `json:"jwks_uri,omitempty"`
	Jwks                    map[string]interface{} `json:"jwks,omitempty"`
	LogoURI                 string                 `json:"logo_uri,omitempty"`
	TLSClientAuthSubjectDN  string                 `json:"tls_client_auth_subject_dn,omitempty"`
	TLSClientAuthSanDNS     string                 `json:"tls_client_auth_san_dns,omitempty"`
	TLSClientAuthSanEmail   string                 `json:"tls_client_auth_san_email,omitempty"`
	TLSClientAuthSanIP      string                 `json:"tls_client_auth_san_ip,omitempty"`
	TLSClientAuthSanURI     string                 `json:"tls_client_auth_san_uri,omitempty"`
	RegistrationAccessToken string                 `json:"registration_access_token,omitempty"`
	extraProperties         map[string]string      `json:"-"`
}

var clientFields map[string]bool

func init() {
	clientFields = make(map[string]bool)
	t := reflect.TypeOf(clientMetadata{})

	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag != "" && tag != "-" {
			fieldName := tag
			if idx := strings.Index(tag, ","); idx > 0 {
				fieldName = tag[:idx]
			}
			clientFields[fieldName] = true
		}
	}
}

func (c *clientMetadata) GetClientName() string {
	return c.ClientName
}

func (c *clientMetadata) GetClientID() string {
	return c.ClientID
}

func (c *clientMetadata) GetClientSecret() string {
	return c.ClientSecret
}

func (c *clientMetadata) GetClientIDIssuedAt() *time.Time {
	if c.ClientIDIssuedAt == nil {
		return nil
	}
	tm := *c.ClientIDIssuedAt
	t := time.Time(tm)
	return &t
}

func (c *clientMetadata) GetClientSecretExpiresAt() *time.Time {
	if c.ClientSecretExpiresAt == nil {
		return nil
	}
	tm := *c.ClientSecretExpiresAt
	t := time.Time(tm)
	return &t
}

func (c *clientMetadata) GetScopes() []string {
	return c.Scope
}

func (c *clientMetadata) GetGrantTypes() []string {
	return c.GrantTypes
}

func (c *clientMetadata) GetResponseTypes() []string {
	return c.ResponseTypes
}

func (c *clientMetadata) GetTokenEndpointAuthMethod() string {
	return c.TokenEndpointAuthMethod
}

func (c *clientMetadata) GetClientURI() string {
	return c.ClientURI
}

func (c *clientMetadata) GetRedirectURIs() []string {
	return c.RedirectURIs
}

func (c *clientMetadata) GetLogoURI() string {
	return c.LogoURI
}

func (c *clientMetadata) GetJwksURI() string {
	return c.JwksURI
}

func (c *clientMetadata) GetJwks() map[string]interface{} {
	return c.Jwks
}

func (c *clientMetadata) GetExtraProperties() map[string]string {
	return c.extraProperties
}

func (c *clientMetadata) GetTLSClientAuthSanDNS() string {
	return c.TLSClientAuthSanDNS
}

func (c *clientMetadata) GetTLSClientAuthSanEmail() string {
	return c.TLSClientAuthSanEmail
}

func (c *clientMetadata) GetTLSClientAuthSanIP() string {
	return c.TLSClientAuthSanIP
}

func (c *clientMetadata) GetTLSClientAuthSanURI() string {
	return c.TLSClientAuthSanURI
}

func (c *clientMetadata) GetRegistrationAccessToken() string {
	return c.RegistrationAccessToken
}

// MarshalJSON serialize the client metadata with provider metadata
func (c *clientMetadata) MarshalJSON() ([]byte, error) {
	type alias clientMetadata
	v := &struct{ *alias }{
		alias: (*alias)(c),
	}

	buf, err := json.Marshal(v)
	if err != nil {
		return buf, err
	}

	allFields := map[string]interface{}{}
	err = json.Unmarshal(buf, &allFields)
	if err != nil {
		return buf, nil
	}

	for k, v := range c.extraProperties {
		allFields[k] = v
	}

	return json.Marshal(allFields)
}

// UnmarshalJSON deserialize the client metadata with provider metadata
func (c *clientMetadata) UnmarshalJSON(data []byte) error {
	type alias clientMetadata
	v := &struct{ *alias }{
		alias: (*alias)(c),
	}

	v.Scope = make([]string, 0)
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	allFields := map[string]interface{}{}
	err = json.Unmarshal(data, &allFields)
	if err != nil {
		return err
	}

	v.extraProperties = make(map[string]string)
	for key, value := range allFields {
		if _, ok := clientFields[key]; !ok {
			if strValue, ok := value.(string); ok {
				v.extraProperties[key] = strValue
			}
		}
	}

	return nil
}
