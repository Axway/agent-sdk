package config

import (
	"encoding/json"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
)

const (
	accessToken              = "accessToken"
	client                   = "client"
	clientSecretBasic        = "client_secret_basic"
	clientSecretPost         = "client_secret_post"
	clientSecretJwt          = "client_secret_jwt"
	privateKeyJwt            = "private_key_jwt"
	tlsClientAuth            = "tls_client_auth"
	selfSignedTlsClientAuth  = "self_signed_tls_client_auth"
	propInsecureSkipVerify   = "insecureSkipVerify"
	pathExternalIDP          = "agentFeatures.idp"
	fldName                  = "name"
	fldTitle                 = "title"
	fldType                  = "type"
	fldMetadataURL           = "metadataUrl"
	fldExtraProperties       = "extraProperties"
	fldScope                 = "scope"
	fldGrantType             = "grantType"
	fldAuthMethod            = "authMethod"
	fldAuthResponseType      = "authResponseType"
	fldAuthType              = "auth.type"
	fldAuthAccessToken       = "auth.accessToken"
	fldAuthClientID          = "auth.clientId"
	fldAuthClientSecret      = "auth.clientSecret"
	fldAuthClientScope       = "auth.clientScope"
	fldAuthPrivateKey        = "auth.privateKey"
	fldAuthPublicKey         = "auth.publicKey"
	fldAuthKeyPassword       = "auth.keyPassword"
	fldSSLInsecureSkipVerify = "ssl." + propInsecureSkipVerify
	fldSSLRootCACertPath     = "ssl.rootCACertPath"
	fldSSLClientCertPath     = "ssl.clientCertPath"
	fldSSLClientKeyPath      = "ssl.clientKeyPath"
)

var configProperties = []string{
	fldName,
	fldTitle,
	fldType,
	fldMetadataURL,
	fldExtraProperties,
	fldScope,
	fldGrantType,
	fldAuthMethod,
	fldAuthResponseType,
	fldAuthType,
	fldAuthAccessToken,
	fldAuthClientID,
	fldAuthClientSecret,
	fldAuthClientScope,
	fldSSLInsecureSkipVerify,
	fldSSLRootCACertPath,
	fldSSLClientCertPath,
	fldSSLClientKeyPath,
	fldAuthPrivateKey,
	fldAuthPublicKey,
	fldAuthKeyPassword,
}

var validIDPAuthType = map[string]bool{
	accessToken:             true,
	client:                  true,
	clientSecretBasic:       true,
	clientSecretPost:        true,
	clientSecretJwt:         true,
	privateKeyJwt:           true,
	selfSignedTlsClientAuth: true,
	tlsClientAuth:           true,
}

// ExternalIDPConfig -
type ExternalIDPConfig interface {
	GetIDPList() []IDPConfig
	ValidateCfg() (err error)
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

func (e *externalIDPConfig) ValidateCfg() (err error) {
	for _, idpCfg := range e.IDPConfigs {
		exception.Block{
			Try: func() {
				idpCfg.validate()
			},
			Catch: func(e error) {
				err = e
			},
		}.Do()
	}
	return err
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
	// GetClientScope - Scopes used for requesting the token using the ID client
	GetClientScope() string
	// validate - Validates the IDP auth configuration
	validate(isMTLS bool)
	// GetPrivateKey() - private key to be used for private_key_jwt authentication
	GetPrivateKey() string
	// GetPublicKey() - public key to be used for private_key_jwt authentication
	GetPublicKey() string
	// GetKeyPassword() - public key to be used for private_key_jwt authentication
	GetKeyPassword() string
}

// IDPConfig - interface for IdP provider config
type IDPConfig interface {
	// GetMetadataURL - URL exposed by OAuth authorization server to provide metadata information
	GetMetadataURL() string
	// GetIDPType - IDP type ("generic" or "okta")
	GetIDPType() string
	// GetIDPName - for the identity provider
	GetIDPName() string
	// GetIDPTitle - for the identity provider friendly name
	GetIDPTitle() string
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
	// GetTLSConfig - tls config for IDP connection
	GetTLSConfig() TLSConfig
	// validate - Validates the IDP configuration
	validate()
}

// IDPAuthConfiguration - Structure to hold the IdP provider auth config
type IDPAuthConfiguration struct {
	Type         string `json:"type,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
	ClientID     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	ClientScope  string `json:"clientScope,omitempty"`
	PrivateKey   string `json:"privateKey,omitempty"`
	PublicKey    string `json:"publicKey,omitempty"`
	KeyPwd       string `json:"keyPassword,omitempty"`
}

// IDPConfiguration - Structure to hold the IdP provider config
type IDPConfiguration struct {
	Name             string          `json:"name,omitempty"`
	Title            string          `json:"title,omitempty"`
	Type             string          `json:"type,omitempty"`
	MetadataURL      string          `json:"metadataUrl,omitempty"`
	AuthConfig       IDPAuthConfig   `json:"auth,omitempty"`
	ClientScopes     string          `json:"scope,omitempty"`
	GrantType        string          `json:"grantType,omitempty"`
	AuthMethod       string          `json:"authMethod,omitempty"`
	AuthResponseType string          `json:"authResponseType,omitempty"`
	ExtraProperties  ExtraProperties `json:"extraProperties,omitempty"`
	TLSConfig        TLSConfig       `json:"ssl,omitempty"`
}

// GetIDPName - for the identity provider
func (i *IDPConfiguration) GetIDPName() string {
	return i.Name
}

// GetIDPName - for the identity provider frinedly name
func (i *IDPConfiguration) GetIDPTitle() string {
	return i.Title
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

// GetTLSConfig - tls config for IDP connection
func (i *IDPConfiguration) GetTLSConfig() TLSConfig {
	return i.TLSConfig
}

// validate - Validates the IDP configuration
func (i *IDPConfiguration) validate() {
	if i.Name == "" {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldName))
	}
	if i.Title == "" {
		i.Title = i.Name
	}

	if i.MetadataURL == "" {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldMetadataURL))
	}

	isMTLS := false
	if i.TLSConfig != nil {
		isMTLS = i.TLSConfig.(*TLSConfiguration).ClientCertificatePath != "" && i.TLSConfig.(*TLSConfiguration).ClientKeyPath != ""
	}
	i.AuthConfig.validate(isMTLS)
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

// GetClientScope - scopes used for requesting the token using the ID client
func (i *IDPAuthConfiguration) GetClientScope() string {
	return i.ClientScope
}

// GetPrivateKey -
func (i *IDPAuthConfiguration) GetPrivateKey() string {
	return i.PrivateKey
}

// GetPublicKey -
func (i *IDPAuthConfiguration) GetPublicKey() string {
	return i.PublicKey
}

// GetKeyPassword -
func (i *IDPAuthConfiguration) GetKeyPassword() string {
	return i.KeyPwd
}

// validate - Validates the IDP auth configuration
func (i *IDPAuthConfiguration) validate(isMTLS bool) {
	if ok := validIDPAuthType[i.GetType()]; !ok {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldAuthType))
	}

	if i.GetType() == accessToken && i.GetAccessToken() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldAuthAccessToken))
	}

	if i.GetType() == client {
		if i.GetClientID() == "" {
			exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldAuthClientID))
		}
		if !isMTLS {
			if i.GetClientSecret() == "" {
				exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldAuthClientSecret))
			}
		}
	}
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
		if entry, ok := envIdpCfg["ssl"]; ok && entry != nil {
			idpCfg.TLSConfig = parseExternalIDPTLSConfig(entry)
		}
		cfg.IDPConfigs[idpCfg.Name] = idpCfg
	}

	return cfg, nil
}

func parseExternalIDPTLSConfig(idpTLSCfg interface{}) TLSConfig {
	tlsCfg := NewTLSConfig()
	tlsConfig, ok := idpTLSCfg.(map[string]interface{})
	if ok {
		buf, _ := json.Marshal(tlsConfig)
		json.Unmarshal(buf, tlsCfg)
		v := tlsConfig[propInsecureSkipVerify]
		if s, ok := v.(string); ok && s == "true" {
			tlsCfg.(*TLSConfiguration).InsecureSkipVerify = true
		}
	}
	return tlsCfg
}
