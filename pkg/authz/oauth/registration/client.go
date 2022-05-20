package registration

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

// Client - Interface for IdP client representation
type Client interface {
	GetClientName() string
	GetClientID() string
	GetClientSecret() string
	GetClientIDIssuedAt() *time.Time
	GetClientSecretExpiresAt() *time.Time
	GetScopes() []string
	GetGrantTypes() []string
	GetResponseTypes() []string
	GetClientURI() string
	GetRedirectURIs() []string
	GetLogoURI() string
	GetJwksURI() string
	GetExtraProperties() map[string]string
}

type client struct {
	ClientName            string `json:"client_name,omitempty"`
	ClientID              string `json:"client_id,omitempty"`
	ClientSecret          string `json:"client_secret,omitempty"`
	ClientIDIssuedAt      *Time  `json:"client_id_issued_at,omitempty"`
	ClientSecretExpiresAt *Time  `json:"client_secret_expires_at,omitempty"`

	Scope Scopes `json:"scope,omitempty"`

	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`

	ClientURI       string            `json:"client_uri,omitempty"`
	RedirectURIs    []string          `json:"redirect_uris,omitempty"`
	JwksURI         string            `json:"jwks_uri,omitempty"`
	LogoURI         string            `json:"logo_uri,omitempty"`
	extraProperties map[string]string `json:"-"`
}

var clientFields map[string]bool

func init() {
	clientFields = make(map[string]bool)
	t := reflect.TypeOf(client{})

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

func (c *client) GetClientName() string {
	return c.ClientName
}

func (c *client) GetClientID() string {
	return c.ClientID
}

func (c *client) GetClientSecret() string {
	return c.ClientSecret
}

func (c *client) GetClientIDIssuedAt() *time.Time {
	if c.ClientIDIssuedAt == nil {
		return nil
	}
	tm := *c.ClientIDIssuedAt
	t := time.Time(tm)
	return &t
}

func (c *client) GetClientSecretExpiresAt() *time.Time {
	if c.ClientSecretExpiresAt == nil {
		return nil
	}
	tm := *c.ClientSecretExpiresAt
	t := time.Time(tm)
	return &t
}

func (c *client) GetScopes() []string {
	return c.Scope
}

func (c *client) GetGrantTypes() []string {
	return c.GrantTypes
}

func (c *client) GetResponseTypes() []string {
	return c.ResponseTypes
}

func (c *client) GetClientURI() string {
	return c.ClientURI
}

func (c *client) GetRedirectURIs() []string {
	return c.RedirectURIs
}

func (c *client) GetLogoURI() string {
	return c.LogoURI
}

func (c *client) GetJwksURI() string {
	return c.JwksURI
}

func (c *client) GetExtraProperties() map[string]string {
	return c.extraProperties
}

// MarshalJSON serialize the client metadata with provider metadata
func (c *client) MarshalJSON() ([]byte, error) {
	type alias client
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
func (c *client) UnmarshalJSON(data []byte) error {
	type alias client
	v := &struct{ *alias }{
		alias: (*alias)(c),
	}

	v.Scope = make([]string, 0)
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}

	allFields := map[string]interface{}{}
	err = json.Unmarshal(buf, &allFields)
	if err != nil {
		return err
	}

	v.extraProperties = make(map[string]string)

	for key, value := range allFields {
		if _, ok := clientFields[key]; !ok {
			if strValue, ok := value.(string); ok {
				c.extraProperties[key] = strValue
			}
		}
	}

	return nil
}
