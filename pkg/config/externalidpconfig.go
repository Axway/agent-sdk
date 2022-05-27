package config

import (
	"encoding/json"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
)

const (
	pathExternalIDP = "agentFeatures.idp"
)

var configProperties = []string{
	"name",
	"type",
	"metadataUrl",
	"extraProperties",
	"scope",
	"grantType",
	"authMethod",
	"authResponseType",
	"auth.type",
	"auth.type",
	"auth.accessToken",
	"auth.clientId",
	"auth.clientSecret",
}

// ExternalIDPConfig -
type ExternalIDPConfig interface {
	GetIDPList() []IDPConfig
}

type externalIDPConfig struct {
	IDPConfigs map[string]IDPConfig
}

func (e *externalIDPConfig) GetIDPList() []IDPConfig {
	list := make([]IDPConfig, 0)
	for _, idpCfg := range e.IDPConfigs {
		list = append(list, idpCfg)
	}
	return list
}

// ExtraProperties - type for representing extra IdP provider properties to be included in client request
type ExtraProperties map[string]string

// UnmarshalJSON - deserializes extra properties from env config
func (e *ExtraProperties) UnmarshalJSON(data []byte) error {
	m := make(map[string]string)
	buf, _ := strconv.Unquote(string(data))
	json.Unmarshal([]byte(buf), &m)

	em := map[string]string(*e)
	for key, val := range m {
		em[key] = val
	}

	return nil
}

// IDPAuthConfig - interface for IdP provider auth config
type IDPAuthConfig interface {
	GetType() string
	GetAccessToken() string
	GetClientID() string
	GetClientSecret() string
	// GetClientScopes() []string
}

// IDPConfig - interface for IdP provider config
type IDPConfig interface {
	GetMetadataURL() string
	GetIDPType() string
	GetIDPName() string
	GetAuthConfig() IDPAuthConfig
	GetClientScopes() string
	GetGrantType() string
	GetAuthMethod() string
	GetAuthResponseType() string
	GetExtraProperties() map[string]string
}

// IDPAuthConfiguration - Structure to hold the IdP provider auth config
type IDPAuthConfiguration struct {
	Type         string   `json:"type,omitempty"`
	AccessToken  string   `json:"accessToken,omitempty"`
	ClientID     string   `json:"clientId,omitempty"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	ClientScopes []string `json:"clientScopes,omitempty"`
}

// IDPConfiguration - Structure to hold the IdP provider config
type IDPConfiguration struct {
	Name             string          `json:"name,omitempty"`
	Type             string          `json:"type,omitempty"`
	MetadataURL      string          `json:"metadataUrl,omitempty"`
	AuthConfig       IDPAuthConfig   `json:"auth,omitempty"`
	ClientScopes     string          `json:"scope,omitempty"`
	GrantType        string          `json:"grantType,omitempty"`
	AuthMethod       string          `json:"authMethod,omitempty"`
	AuthResponseType string          `json:"authResponseType,omitempty"`
	ExtraProperties  ExtraProperties `json:"extraProperties,omitempty"`
}

// GetIDPName - returns the name of IdP provider
func (i *IDPConfiguration) GetIDPName() string {
	return i.Name
}

// GetIDPType - returns the IdP type
func (i *IDPConfiguration) GetIDPType() string {
	return i.Type
}

// GetAuthConfig - returns the IdP Auth config
func (i *IDPConfiguration) GetAuthConfig() IDPAuthConfig {
	return i.AuthConfig
}

// GetMetadataURL - returns the metadata URL for IdP
func (i *IDPConfiguration) GetMetadataURL() string {
	return i.MetadataURL
}

// GetExtraProperties - returns the IdP specific properties to be included in client request
func (i *IDPConfiguration) GetExtraProperties() map[string]string {
	return i.ExtraProperties
}

// GetClientScopes - returns the Client scopes to be used for registering IdP client
func (i *IDPConfiguration) GetClientScopes() string {
	return i.ClientScopes
}

// GetGrantType - returns the Client grant type to be used for registering IdP client
func (i *IDPConfiguration) GetGrantType() string {
	return i.GrantType
}

// GetAuthMethod - returns the Client auth method to be used for registering IdP client
func (i *IDPConfiguration) GetAuthMethod() string {
	return i.AuthMethod
}

// GetAuthResponseType - returns the Client auth response type to be used for registering IdP client
func (i *IDPConfiguration) GetAuthResponseType() string {
	return i.AuthResponseType
}

// GetType - returns the auth type to be used for IdP client registration APIs
func (i *IDPAuthConfiguration) GetType() string {
	return i.Type
}

// GetAccessToken - returns the access token to be used for IdP client registration APIs
func (i *IDPAuthConfiguration) GetAccessToken() string {
	return i.AccessToken
}

// GetClientID - returns the Client ID to be used for IdP client registration APIs
func (i *IDPAuthConfiguration) GetClientID() string {
	return i.ClientID
}

// GetClientSecret - returns the Client Secret to be used for IdP client registration APIs
func (i *IDPAuthConfiguration) GetClientSecret() string {
	return i.ClientSecret
}

func addExternalIDPProperties(props properties.Properties) {
	props.AddObjectSliceProperty(pathExternalIDP, configProperties)
}

func parseExternalIDPConfig(props properties.Properties) (ExternalIDPConfig, error) {
	envIDPCfgList := props.ObjectSlicePropertyValue(pathExternalIDP)

	cfg := &externalIDPConfig{
		IDPConfigs: make(map[string]IDPConfig),
	}

	for _, envIdpCfg := range envIDPCfgList {
		idpCfg := &IDPConfiguration{
			AuthConfig:       &IDPAuthConfiguration{},
			ExtraProperties:  make(ExtraProperties),
			ClientScopes:     "resource.Read resource.Write",
			GrantType:        "client_credentials",
			AuthMethod:       "client_secret_basic",
			AuthResponseType: "token",
		}

		buf, _ := json.Marshal(envIdpCfg)
		json.Unmarshal(buf, idpCfg)

		cfg.IDPConfigs[idpCfg.Name] = idpCfg
	}

	return cfg, nil
}
