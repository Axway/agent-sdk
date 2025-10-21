package config

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
)

const (
	AccessToken             = "accessToken"
	Client                  = "client"
	ClientSecretBasic       = "client_secret_basic"
	ClientSecretPost        = "client_secret_post"
	ClientSecretJWT         = "client_secret_jwt"
	PrivateKeyJWT           = "private_key_jwt"
	TLSClientAuth           = "tls_client_auth"
	SelfSignedTLSClientAuth = "self_signed_tls_client_auth"

	propInsecureSkipVerify      = "insecureSkipVerify"
	propUseCachedToken          = "useCachedToken"
	propUseRegistrationToken    = "useRegistrationToken"
	pathExternalIDP             = "agentFeatures.idp"
	fldName                     = "name"
	fldTitle                    = "title"
	fldType                     = "type"
	fldMetadataURL              = "metadataUrl"
	fldExtraProperties          = "extraProperties"
	fldRequestHeaders           = "requestHeaders"
	fldQueryParams              = "queryParams"
	fldScope                    = "scope"
	fldGrantType                = "grantType"
	fldAuthMethod               = "authMethod"
	fldAuthResponseType         = "authResponseType"
	fldAuthType                 = "auth.type"
	fldAuthRequestHeaders       = "auth.requestHeaders"
	fldAuthQueryParams          = "auth.queryParams"
	fldAuthAccessToken          = "auth.accessToken"
	fldAuthClientID             = "auth.clientId"
	fldAuthClientSecret         = "auth.clientSecret"
	fldAuthClientScope          = "auth.clientScope"
	fldAuthPrivateKey           = "auth.privateKey"
	fldAuthPublicKey            = "auth.publicKey"
	fldAuthKeyPassword          = "auth.keyPassword"
	fldAuthTokenSigningMethod   = "auth.tokenSigningMethod"
	fldAuthUseCachedToken       = "auth." + propUseCachedToken
	fldAuthUseRegistrationToken = "auth." + propUseRegistrationToken
	fldSSLInsecureSkipVerify    = "ssl." + propInsecureSkipVerify
	fldSSLRootCACertPath        = "ssl.rootCACertPath"
	fldSSLClientCertPath        = "ssl.clientCertPath"
	fldSSLClientKeyPath         = "ssl.clientKeyPath"
)

var configProperties = []string{
	fldName,
	fldTitle,
	fldType,
	fldMetadataURL,
	fldExtraProperties,
	fldRequestHeaders,
	fldQueryParams,
	fldScope,
	fldGrantType,
	fldAuthMethod,
	fldAuthResponseType,
	fldAuthType,
	fldAuthRequestHeaders,
	fldAuthQueryParams,
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
	fldAuthTokenSigningMethod,
	fldAuthUseCachedToken,
	fldAuthUseRegistrationToken,
}

var validIDPAuthType = map[string]bool{
	AccessToken:             true,
	Client:                  true,
	ClientSecretBasic:       true,
	ClientSecretPost:        true,
	ClientSecretJWT:         true,
	PrivateKeyJWT:           true,
	SelfSignedTLSClientAuth: true,
	TLSClientAuth:           true,
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
type ExtraProperties map[string]interface{}

// UnmarshalJSON - deserializes extra properties from env config
func (e *ExtraProperties) UnmarshalJSON(data []byte) error {
	// Parse the JSON string wrapper if present
	buf, err := strconv.Unquote(string(data))
	if err != nil {
		// Not a quoted string, use as-is
		buf = string(data)
	}

	// Unmarshal into the map, preserving types
	m := make(map[string]interface{})
	if err := json.Unmarshal([]byte(buf), &m); err != nil {
		return err
	}

	for key, val := range m {
		(*e)[key] = val
	}
	return nil
}

// IDPRequestHeaders - additional request headers for calls to IdP
type IDPRequestHeaders map[string]string

// UnmarshalJSON - deserializes request header from env config
func (e *IDPRequestHeaders) UnmarshalJSON(data []byte) error {
	headers := map[string]string(*e)
	return parseKeyValuePairs(headers, data)
}

// IDPQueryParams - additional query params for calls to IdP
type IDPQueryParams map[string]string

// UnmarshalJSON - deserializes query parameters from env config
func (e *IDPQueryParams) UnmarshalJSON(data []byte) error {
	qp := map[string]string(*e)
	return parseKeyValuePairs(qp, data)
}

func parseKeyValuePairs(kv map[string]string, data []byte) error {
	m := make(map[string]string)
	// try parsing the data as json string, ground agent config setup as json string
	buf, err := strconv.Unquote(string(data))
	if err != nil {
		// parse as json map in case data is not json string
		buf = string(data)
	}
	err = json.Unmarshal([]byte(buf), &m)
	if err != nil {
		return err
	}

	for key, val := range m {
		kv[key] = val
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
	validate(tlsCfg TLSConfig)
	// GetPrivateKey() - private key to be used for private_key_jwt authentication
	GetPrivateKey() string
	// GetPublicKey() - public key to be used for private_key_jwt authentication
	GetPublicKey() string
	// GetKeyPassword() - public key to be used for private_key_jwt authentication
	GetKeyPassword() string
	// GetSigningMethod() - the token signing method for private_key_jwt authentication
	GetTokenSigningMethod() string
	// UseTokenCache() - return flag to indicate if the auth client to get new token on each request
	UseTokenCache() bool
	// UseRegistrationAccessToken - return flag to indicate if the auth client to use registration access token
	UseRegistrationAccessToken() bool
	// GetRequestHeaders - set of additional request headers to be applied when registering the client
	GetRequestHeaders() map[string]string
	// GetQueryParams - set of additional query parameters to be applied when registering the client
	GetQueryParams() map[string]string
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
	GetExtraProperties() map[string]interface{}
	// GetRequestHeaders - set of additional request headers to be applied when registering the client
	GetRequestHeaders() map[string]string
	// GetQueryParams - set of additional query parameters to be applied when registering the client
	GetQueryParams() map[string]string
	// GetTLSConfig - tls config for IDP connection
	GetTLSConfig() TLSConfig
	// validate - Validates the IDP configuration
	validate()
}

// IDPAuthConfiguration - Structure to hold the IdP provider auth config
type IDPAuthConfiguration struct {
	Type                 string            `json:"type,omitempty"`
	RequestHeaders       IDPRequestHeaders `json:"requestHeaders,omitempty"`
	QueryParams          IDPQueryParams    `json:"queryParams,omitempty"`
	AccessToken          string            `json:"accessToken,omitempty"`
	ClientID             string            `json:"clientId,omitempty"`
	ClientSecret         string            `json:"clientSecret,omitempty"`
	ClientScope          string            `json:"clientScope,omitempty"`
	PrivateKey           string            `json:"privateKey,omitempty"`
	PublicKey            string            `json:"publicKey,omitempty"`
	KeyPwd               string            `json:"keyPassword,omitempty"`
	TokenSigningMethod   string            `json:"tokenSigningMethod,omitempty"`
	UseCachedToken       bool              `json:"-"`
	UseRegistrationToken bool              `json:"-"`
}

// IDPConfiguration - Structure to hold the IdP provider config
type IDPConfiguration struct {
	Name             string            `json:"name,omitempty"`
	Title            string            `json:"title,omitempty"`
	Type             string            `json:"type,omitempty"`
	MetadataURL      string            `json:"metadataUrl,omitempty"`
	AuthConfig       IDPAuthConfig     `json:"auth,omitempty"`
	ClientScopes     string            `json:"scope,omitempty"`
	GrantType        string            `json:"grantType,omitempty"`
	AuthMethod       string            `json:"authMethod,omitempty"`
	AuthResponseType string            `json:"authResponseType,omitempty"`
	ExtraProperties  ExtraProperties   `json:"extraProperties,omitempty"`
	RequestHeaders   IDPRequestHeaders `json:"requestHeaders,omitempty"`
	QueryParams      IDPQueryParams    `json:"queryParams,omitempty"`
	TLSConfig        TLSConfig         `json:"-"`
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
func (i *IDPConfiguration) GetExtraProperties() map[string]interface{} {
	return i.ExtraProperties
}

// GetRequestHeaders - set of additional request headers to be applied when registering the client
func (i *IDPConfiguration) GetRequestHeaders() map[string]string {
	return i.RequestHeaders
}

// GetQueryParams - set of additional query params to be applied when registering the client
func (i *IDPConfiguration) GetQueryParams() map[string]string {
	return i.QueryParams
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

// UnmarshalJSON - custom unmarshaler for IDPConfiguration struct
func (i *IDPConfiguration) UnmarshalJSON(data []byte) error {
	type Alias IDPConfiguration
	i.RequestHeaders = make(IDPRequestHeaders)
	i.QueryParams = make(IDPQueryParams)
	i.ExtraProperties = make(ExtraProperties)

	i.AuthConfig = &IDPAuthConfiguration{
		RequestHeaders: make(IDPRequestHeaders),
		QueryParams:    make(IDPQueryParams),
	}
	if err := json.Unmarshal(data, &struct{ *Alias }{Alias: (*Alias)(i)}); err != nil {
		return err
	}

	var allFields interface{}
	json.Unmarshal(data, &allFields)
	b := allFields.(map[string]interface{})

	if v, ok := b["auth"]; ok {
		buf, _ := json.Marshal(v)
		json.Unmarshal(buf, i.AuthConfig)
	}

	return nil
}

// MarshalJSON - custom marshaler for IDPConfiguration struct
func (i *IDPConfiguration) MarshalJSON() ([]byte, error) {
	type Alias IDPConfiguration

	idp, err := json.Marshal(&struct{ *Alias }{Alias: (*Alias)(i)})
	if err != nil {
		return nil, err
	}
	var allFields interface{}
	json.Unmarshal(idp, &allFields)
	b := allFields.(map[string]interface{})

	idpAuthCfg, err := json.Marshal(i.AuthConfig)
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	json.Unmarshal(idpAuthCfg, &m)
	b["auth"] = m

	return json.Marshal(b)
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

	i.AuthConfig.validate(i.TLSConfig)
}

// GetType - type of authentication mechanism to use "accessToken" or "client"
func (i *IDPAuthConfiguration) GetType() string {
	return i.Type
}

// GetRequestHeaders - set of additional request headers to be applied when registering the client
func (i *IDPAuthConfiguration) GetRequestHeaders() map[string]string {
	return i.RequestHeaders
}

// GetQueryParams - set of additional query params to be applied when registering the client
func (i *IDPAuthConfiguration) GetQueryParams() map[string]string {
	return i.QueryParams
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

// GetTokenSigningMethod -
func (i *IDPAuthConfiguration) GetTokenSigningMethod() string {
	return i.TokenSigningMethod
}

// UseTokenCache - return flag to indicate if the auth client to get new token on each request
func (i *IDPAuthConfiguration) UseTokenCache() bool {
	return i.UseCachedToken
}

// UseRegistrationAccessToken - return flag to indicate if the auth client to use registration access token
func (i *IDPAuthConfiguration) UseRegistrationAccessToken() bool {
	return i.UseRegistrationToken
}

// UnmarshalJSON - custom unmarshaler for IDPAuthConfiguration struct
func (i *IDPAuthConfiguration) UnmarshalJSON(data []byte) error {
	type Alias IDPAuthConfiguration // Create an intermittent type to unmarshal the base attributes

	if err := json.Unmarshal(data, &struct{ *Alias }{Alias: (*Alias)(i)}); err != nil {
		return err
	}

	var allFields interface{}
	json.Unmarshal(data, &allFields)
	b := allFields.(map[string]interface{})

	// Default to use the cached token
	i.UseCachedToken = true
	if v, ok := b[propUseCachedToken]; ok {
		i.UseCachedToken = (v == "true")
	}
	// Default to use not use registration access token
	i.UseRegistrationToken = false
	if v, ok := b[propUseRegistrationToken]; ok {
		i.UseRegistrationToken = (v == "true")
	}
	return nil
}

// MarshalJSON - custom marshaler for Application struct
func (i *IDPAuthConfiguration) MarshalJSON() ([]byte, error) {
	type Alias IDPAuthConfiguration // Create an intermittent type to marshal the base attributes

	app, err := json.Marshal(&struct{ *Alias }{Alias: (*Alias)(i)})
	if err != nil {
		return nil, err
	}

	// decode it back to get a map
	var allFields interface{}
	json.Unmarshal(app, &allFields)
	b := allFields.(map[string]interface{})

	if i.UseCachedToken {
		b[propUseCachedToken] = "true"
	}

	// Return encoding of the map
	return json.Marshal(b)
}

// validate - Validates the IDP auth configuration
func (i *IDPAuthConfiguration) validate(tlsCfg TLSConfig) {
	if ok := validIDPAuthType[i.GetType()]; !ok {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldAuthType))
	}

	switch i.GetType() {
	case AccessToken:
		i.validateAccessTokenAuthConfig()
	case Client:
		fallthrough
	case ClientSecretBasic:
		fallthrough
	case ClientSecretPost:
		fallthrough
	case ClientSecretJWT:
		i.validateClientSecretAuthConfig()
	case PrivateKeyJWT:
		i.validatePrivateKeyJwtAuthConfig()
	case TLSClientAuth:
		fallthrough
	case SelfSignedTLSClientAuth:
		i.validateTLSClientAuthConfig(tlsCfg)
	}
}

func (i *IDPAuthConfiguration) validateAccessTokenAuthConfig() {
	if i.GetAccessToken() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldAuthAccessToken))
	}
}

func (i *IDPAuthConfiguration) validateClientIDConfig() {
	if i.GetClientID() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldAuthClientID))
	}
}

func (i *IDPAuthConfiguration) validateClientSecretConfig() {
	if i.GetClientSecret() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldAuthClientSecret))
	}
}

func (i *IDPAuthConfiguration) validateClientSecretAuthConfig() {
	i.validateClientIDConfig()
	i.validateClientSecretConfig()
}

func (i *IDPAuthConfiguration) validatePrivateKeyJwtAuthConfig() {
	i.validateClientIDConfig()

	validateAuthFileConfig(pathExternalIDP+"."+fldAuthPrivateKey, i.PrivateKey, "", "private key")
	validateAuthFileConfig(pathExternalIDP+"."+fldAuthPublicKey, i.PublicKey, "", "public key")
}

func (i *IDPAuthConfiguration) validateTLSClientAuthConfig(tlsCfg TLSConfig) {
	i.validateClientIDConfig()

	if tlsCfg == nil {
		exception.Throw(ErrBadConfig.FormatError(pathExternalIDP + "." + fldSSLClientCertPath))
	}
	validateAuthFileConfig(pathExternalIDP+"."+fldSSLClientCertPath, tlsCfg.(*TLSConfiguration).ClientCertificatePath, "", "tls client certificate")
	validateAuthFileConfig(pathExternalIDP+"."+fldSSLClientKeyPath, tlsCfg.(*TLSConfiguration).ClientKeyPath, "", "tls client key")
}

func addExternalIDPProperties(props properties.Properties) {
	props.AddObjectSliceProperty(pathExternalIDP, configProperties)
}

func ParseExternalIDPConfig(agentFeature AgentFeaturesConfig, props properties.Properties) error {
	af, ok := agentFeature.(*AgentFeaturesConfiguration)
	if !ok {
		return nil
	}
	envIDPCfgList := props.ObjectSlicePropertyValue(pathExternalIDP)

	cfg := &externalIDPConfig{
		IDPConfigs: make(map[string]IDPConfig),
	}

	for _, envIdpCfg := range envIDPCfgList {
		idpCfg := &IDPConfiguration{
			AuthConfig: &IDPAuthConfiguration{
				RequestHeaders: make(IDPRequestHeaders),
				QueryParams:    make(IDPQueryParams),
			},
			ExtraProperties:  make(ExtraProperties),
			RequestHeaders:   make(IDPRequestHeaders),
			QueryParams:      make(IDPQueryParams),
			ClientScopes:     "resource.READ resource.WRITE",
			GrantType:        "client_credentials",
			AuthMethod:       "client_secret_basic",
			AuthResponseType: "token",
		}

		buf, _ := json.Marshal(envIdpCfg)
		err := json.Unmarshal(buf, idpCfg)
		if err != nil {
			return fmt.Errorf("error parsing idp configuration, %s", err)
		}
		if entry, ok := envIdpCfg["ssl"]; ok && entry != nil {
			idpCfg.TLSConfig = parseExternalIDPTLSConfig(entry)
		}
		cfg.IDPConfigs[idpCfg.Name] = idpCfg
	}
	af.ExternalIDPConfig = cfg

	return nil
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
