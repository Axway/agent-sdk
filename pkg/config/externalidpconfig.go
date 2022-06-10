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
	// GetType - type of authentication mechanism to use "accessToken" or "client"
	GetType() string
	// GetAccessToken - token(initial access token/Admin API Token etc) to be used by Agent SDK to authenticate with IdP
	GetAccessToken() string
	// GetClientID - Identifier of the client in IdP that can used to create new OAuth clients
	GetClientID() string
	// GetClientSecret - Secret for the client in IdP that can used to create new OAuth clients
	GetClientSecret() string
}

// IDPConfig - interface for IdP provider config
type IDPConfig interface {
	// GetMetadataURL - URL exposed by OAuth authorization server to provide metadata information
	GetMetadataURL() string
	// GetIDPType - IDP type ("generic" or "okta")
	GetIDPType() string
	// GetIDPName - for the identity provider
	GetIDPName() string
	// GetAuthConfig - to be used for authentication with IDP
	GetAuthConfig() IDPAuthConfig
	// GetClientScopes - default list of scopes that are included in the client metadata request to IDP
	GetClientScopes() string
	// GetGrantType - default grant type to be used when creating the client. (default :  "client_credentials")
	GetGrantType() string
	// GetAuthMethod - default token endpoint authentication method(default : "client_secret_basic")
	GetAuthMethod() string
	// GetAuthResponseType - default token response type to be used when registering the client
	GetAuthResponseType() string
	// GetExtraProperties - set of additional properties to be applied when registering the client
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

// GetIDPName - for the identity provider
func (i *IDPConfiguration) GetIDPName() string {
	return i.Name
}

// GetIDPType - IDP type ("generic" or "okta")
func (i *IDPConfiguration) GetIDPType() string {
	return i.Type
}

// GetAuthConfig - to be used for authentication with IDP
func (i *IDPConfiguration) GetAuthConfig() IDPAuthConfig {
	return i.AuthConfig
}

// GetMetadataURL - URL exposed by OAuth authorization server to provide metadata information
func (i *IDPConfiguration) GetMetadataURL() string {
	return i.MetadataURL
}

// GetExtraProperties - set of additional properties to be applied when registering the client
func (i *IDPConfiguration) GetExtraProperties() map[string]string {
	return i.ExtraProperties
}

// GetClientScopes - default list of scopes that are included in the client metadata request to IDP
func (i *IDPConfiguration) GetClientScopes() string {
	return i.ClientScopes
}

// GetGrantType - default grant type to be used when creating the client. (default :  "client_credentials")
func (i *IDPConfiguration) GetGrantType() string {
	return i.GrantType
}

// GetAuthMethod - default token endpoint authentication method(default : "client_secret_basic")
func (i *IDPConfiguration) GetAuthMethod() string {
	return i.AuthMethod
}

// GetAuthResponseType - default token response type to be used when registering the client
func (i *IDPConfiguration) GetAuthResponseType() string {
	return i.AuthResponseType
}

// GetType - type of authentication mechanism to use "accessToken" or "client"
func (i *IDPAuthConfiguration) GetType() string {
	return i.Type
}

// GetAccessToken - token(initial access token/Admin API Token etc) to be used by Agent SDK to authenticate with IdP
func (i *IDPAuthConfiguration) GetAccessToken() string {
	return i.AccessToken
}

// GetClientID - Identifier of the client in IdP that can used to create new OAuth client
func (i *IDPAuthConfiguration) GetClientID() string {
	return i.ClientID
}

// GetClientSecret - Secret for the client in IdP that can used to create new OAuth clients
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
			ClientScopes:     "resource.READ resource.WRITE",
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
