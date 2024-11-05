package agent

import (
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
)

var supportedIDPGrantTypes = map[string]bool{
	oauth.GrantTypeClientCredentials: true,
	oauth.GrantTypeAuthorizationCode: true}

var supportedIDPTokenAuthMethods = map[string]bool{
	config.ClientSecretBasic:       true,
	config.ClientSecretPost:        true,
	config.ClientSecretJWT:         true,
	config.PrivateKeyJWT:           true,
	config.TLSClientAuth:           true,
	config.SelfSignedTLSClientAuth: true,
}

var tlsAuthCertificateMetadata = []string{
	oauth.TLSClientAuthSubjectDN,
	oauth.TLSClientAuthSanDNS,
	oauth.TLSClientAuthSanEmail,
	oauth.TLSClientAuthSanIP,
	oauth.TLSClientAuthSanURI,
}

// createOrUpdateCredentialRequestDefinition -
func createOrUpdateCredentialRequestDefinition(data *management.CredentialRequestDefinition) (*management.CredentialRequestDefinition, error) {
	ri, err := createOrUpdateDefinition(data)
	if ri == nil || err != nil {
		return nil, err
	}
	err = data.FromInstance(ri)
	return data, err
}

// createOrUpdateDefinition -
func createOrUpdateDefinition(data v1.Interface) (*v1.ResourceInstance, error) {

	ri, err := agent.apicClient.CreateOrUpdateResource(data)
	if err != nil {
		return nil, err
	}

	var existingRI *v1.ResourceInstance

	switch ri.Kind {
	case management.AccessRequestDefinitionGVK().Kind:
		existingRI, _ = agent.cacheManager.GetAccessRequestDefinitionByName(ri.Name)
	case management.CredentialRequestDefinitionGVK().Kind:
		existingRI, _ = agent.cacheManager.GetCredentialRequestDefinitionByName(ri.Name)
	}

	// if not existing, go ahead and add the request definition
	if existingRI == nil {
		switch ri.Kind {
		case management.AccessRequestDefinitionGVK().Kind:
			agent.cacheManager.AddAccessRequestDefinition(ri)
		case management.CredentialRequestDefinitionGVK().Kind:
			agent.cacheManager.AddCredentialRequestDefinition(ri)
		}
	}

	return ri, nil
}

type crdBuilderOptions struct {
	name               string
	title              string
	renewable          bool
	suspendable        bool
	deprovisionExpired bool
	expirationDays     int
	provProps          []provisioning.PropertyBuilder
	reqProps           []provisioning.PropertyBuilder
	registerFunc       provisioning.RegisterCredentialRequestDefinition
}

// NewCredentialRequestBuilder - called by the agents to build and register a new credential request definition
func NewCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	thisCred := &crdBuilderOptions{
		renewable:    false,
		provProps:    make([]provisioning.PropertyBuilder, 0),
		reqProps:     make([]provisioning.PropertyBuilder, 0),
		registerFunc: createOrUpdateCredentialRequestDefinition,
	}

	if agent.cfg != nil {
		thisCred.expirationDays = agent.cfg.GetCredentialConfig().GetExpirationDays()
		thisCred.deprovisionExpired = agent.cfg.GetCredentialConfig().ShouldDeprovisionExpired()
	}

	for _, o := range options {
		o(thisCred)
	}

	provSchema := provisioning.NewSchemaBuilder()
	for _, provProp := range thisCred.provProps {
		provSchema.AddProperty(provProp)
	}

	reqSchema := provisioning.NewSchemaBuilder()
	for _, props := range thisCred.reqProps {
		reqSchema.AddProperty(props)
	}

	builder := provisioning.NewCRDBuilder(thisCred.registerFunc).
		SetName(thisCred.name).
		SetTitle(thisCred.title).
		SetProvisionSchema(provSchema).
		SetRequestSchema(reqSchema).
		SetExpirationDays(thisCred.expirationDays)

	if thisCred.renewable {
		builder.IsRenewable()
	}

	if thisCred.suspendable {
		builder.IsSuspendable()
	}

	if thisCred.deprovisionExpired {
		builder.SetDeprovisionExpired()
	}

	return builder
}

// WithCRDName - set another name for the CRD
func WithCRDName(name string) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.name = name
	}
}

// WithCRDTitle - set the title for the CRD
func WithCRDTitle(title string) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.title = title
	}
}

// WithCRDIsRenewable - set the flag for renewable credential
func WithCRDIsRenewable() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.renewable = true
	}
}

// WithCRDIsSuspendable - set the flag for suspendable credential
func WithCRDIsSuspendable() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.suspendable = true
	}
}

// WithCRDExpirationDays - set the expiration days
func WithCRDExpirationDays(expirationDays int) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.expirationDays = expirationDays
	}
}

// WithCRDDeprovisionExpired - set the flag for deprovisioning expired credential
func WithCRDDeprovisionExpired() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.deprovisionExpired = true
	}
}

// WithCRDProvisionSchemaProperty - add more provisioning properties
func WithCRDProvisionSchemaProperty(prop provisioning.PropertyBuilder) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.provProps = append(c.provProps, prop)
	}
}

// WithCRDRequestSchemaProperty - add more request properties
func WithCRDRequestSchemaProperty(prop provisioning.PropertyBuilder) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.reqProps = append(c.reqProps, prop)
	}
}

// WithCRDRegisterFunc - use the provided registration function for creating CRD
func WithCRDRegisterFunc(registerFunc provisioning.RegisterCredentialRequestDefinition) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.registerFunc = registerFunc
	}
}

func idpUsesPrivateKeyJWTAuth(tokenAuthMethods []string) bool {
	for _, s := range tokenAuthMethods {
		if s == config.PrivateKeyJWT {
			return true
		}
	}
	return false
}

func idpUsesTLSClientAuth(tokenAuthMethods []string) bool {
	for _, s := range tokenAuthMethods {
		if s == config.TLSClientAuth || s == config.SelfSignedTLSClientAuth {
			return true
		}
	}
	return false
}

// WithCRDForIDP - set the schema properties using the provider metadata
func WithCRDForIDP(p oauth.Provider, scopes []string) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		if c.name == "" {
			name := util.ConvertToDomainNameCompliant(p.GetName())
			c.name = name + "-" + provisioning.OAuthIDPCRD
			c.title = "OAuth" + p.GetName()
		}

		setIDPClientSecretSchemaProperty(c)
		setIDPTokenURLSchemaProperty(p, c)
		setIDPScopesSchemaProperty(scopes, c)
		setIDPGrantTypesSchemaProperty(p, c)
		setIDPTokenAuthMethodSchemaProperty(p, c)
		setIDPRedirectURIsSchemaProperty(c)
	}
}

func setIDPClientSecretSchemaProperty(c *crdBuilderOptions) {
	c.provProps = append(c.provProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthClientSecret).
			SetLabel("Client Secret").
			IsString().
			IsEncrypted())
}

func setIDPTokenURLSchemaProperty(p oauth.Provider, c *crdBuilderOptions) {
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.IDPTokenURL).
			SetRequired().
			SetLabel("Token URL").
			SetReadOnly().
			IsString().
			SetDefaultValue(p.GetTokenEndpoint()))
}

func setIDPScopesSchemaProperty(scopes []string, c *crdBuilderOptions) {
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthScopes).
			SetLabel("Scopes").
			IsArray().
			AddItem(
				provisioning.NewSchemaPropertyBuilder().
					SetName("scope").
					IsString().SetEnumValues(scopes).SetSortEnumValues()))
}

func setIDPGrantTypesSchemaProperty(p oauth.Provider, c *crdBuilderOptions) {
	grantType, defaultGrantType := removeUnsupportedTypes(
		p.GetSupportedGrantTypes(), supportedIDPGrantTypes, oauth.GrantTypeClientCredentials)

	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthGrantType).
			SetLabel("Grant Type").
			IsString().
			SetDefaultValue(defaultGrantType).
			SetEnumValues(grantType))
}

func removeUnsupportedTypes(values []string, supportedTypes map[string]bool, defaultType string) ([]string, string) {
	var result []string
	defaultSupportedType := ""
	defaultExists := false
	for _, s := range values {
		if ok := supportedTypes[s]; ok {
			if s == defaultType {
				defaultExists = true
			}
			if defaultSupportedType == "" {
				defaultSupportedType = s
			}
			result = append(result, s)
		}
	}

	if !defaultExists {
		defaultType = defaultSupportedType
	}
	return result, defaultType
}

func setIDPTokenAuthMethodSchemaProperty(p oauth.Provider, c *crdBuilderOptions) []string {
	tokenAuthMethods, defaultTokenMethod := removeUnsupportedTypes(
		p.GetSupportedTokenAuthMethods(), supportedIDPTokenAuthMethods, config.ClientSecretBasic)

	tmBuilder := provisioning.NewSchemaPropertyBuilder().
		SetName(provisioning.OauthTokenAuthMethod).
		SetLabel("Token Auth Method").
		IsString().
		SetDefaultValue(defaultTokenMethod).
		SetEnumValues(tokenAuthMethods)

	if idpUsesPrivateKeyJWTAuth(tokenAuthMethods) {
		setIDPJWKSURISchemaProperty(config.PrivateKeyJWT, tmBuilder)
		setIDPJWKSSchemaProperty(config.PrivateKeyJWT, tmBuilder)
	}

	if idpUsesTLSClientAuth(tokenAuthMethods) {
		setIDPJWKSURISchemaProperty(config.TLSClientAuth, tmBuilder)
		setIDPTLSClientAuthSchemaProperty(tmBuilder)
	}

	c.reqProps = append(c.reqProps, tmBuilder)
	return tokenAuthMethods
}

func setIDPRedirectURIsSchemaProperty(c *crdBuilderOptions) {
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthRedirectURIs).
			SetLabel("Redirect URLs").
			IsArray().
			AddItem(
				provisioning.NewSchemaPropertyBuilder().
					SetName("URL").
					IsString()))
}

func setIDPJWKSURISchemaProperty(depValue string, propBuilder provisioning.StringPropertyBuilder) {
	propBuilder.AddDependency(
		depValue,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthJwksURI).
			SetLabel("JWKS URI").
			IsString())
}

func setIDPJWKSSchemaProperty(depValue string, propBuilder provisioning.StringPropertyBuilder) {
	propBuilder.AddDependency(
		depValue,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthJwks).
			SetLabel("Public Key").
			IsString())
}

func setIDPTLSClientAuthSchemaProperty(propBuilder provisioning.StringPropertyBuilder) {
	propBuilder.AddDependency(
		config.TLSClientAuth,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthCertificate).
			SetLabel("Public Certificate").
			IsString())

	certMetadataBuilder := provisioning.NewSchemaPropertyBuilder().
		SetName(provisioning.OauthCertificateMetadata).
		SetLabel("Certificate Metadata").
		IsString().
		SetDefaultValue(oauth.TLSClientAuthSubjectDN).
		SetEnumValues(tlsAuthCertificateMetadata)

	propBuilder.AddDependency(
		config.TLSClientAuth,
		certMetadataBuilder,
	)
	certMetadataBuilder.AddDependency(
		oauth.TLSClientAuthSanDNS,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTLSAuthSANDNS).
			SetLabel("Certificate Subject Alternative Name, DNS").
			IsString())

	certMetadataBuilder.AddDependency(
		oauth.TLSClientAuthSanEmail,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTLSAuthSANEmail).
			SetLabel("Certificate Subject Alternative Name, Email").
			IsString())

	certMetadataBuilder.AddDependency(
		oauth.TLSClientAuthSanIP,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTLSAuthSANIP).
			SetLabel("Certificate Subject Alternative Name, IP address").
			IsString())

	certMetadataBuilder.AddDependency(
		oauth.TLSClientAuthSanURI,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTLSAuthSANURI).
			SetLabel("Certificate Subject Alternative Name, URI").
			IsString())
}

// WithCRDOAuthSecret - set that the Oauth cred is secret based
func WithCRDOAuthSecret() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		if c.name == "" {
			c.name = provisioning.OAuthSecretCRD
			c.title = "OAuth Client ID & Secret"
		}
		c.provProps = append(c.provProps,
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.OauthClientSecret).
				SetLabel("Client Secret").
				SetRequired().
				IsString().
				IsEncrypted())
	}
}

// WithCRDOAuthPublicKey - set that the Oauth cred is key based
func WithCRDOAuthPublicKey() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		if c.name == "" {
			c.name = provisioning.OAuthPublicKeyCRD
			c.title = "OAuth Client ID & Private Key"
		}

		c.reqProps = append(c.reqProps,
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.OauthPublicKey).
				SetLabel("Public Key").
				SetRequired().
				IsString())
	}
}

// NewAPIKeyCredentialRequestBuilder - add api key base properties for provisioning schema
func NewAPIKeyCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	apiKeyOptions := []func(*crdBuilderOptions){
		WithCRDName(provisioning.APIKeyCRD),
		WithCRDTitle("API Key"),
		WithCRDProvisionSchemaProperty(
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.APIKey).
				SetLabel("API Key").
				SetRequired().
				IsString().
				IsEncrypted()),
	}

	apiKeyOptions = append(apiKeyOptions, options...)

	return NewCredentialRequestBuilder(apiKeyOptions...)
}

// NewBasicAuthCredentialRequestBuilder - add basic auth base properties for provisioning schema
func NewBasicAuthCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	basicAuthOptions := []func(*crdBuilderOptions){
		WithCRDName(provisioning.BasicAuthCRD),
		WithCRDTitle("Basic Auth"),
		WithCRDProvisionSchemaProperty(
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.BasicAuthUsername).
				SetLabel("Username").
				SetRequired().
				IsString().
				IsEncrypted()),
		WithCRDProvisionSchemaProperty(
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.BasicAuthPassword).
				SetLabel("Password").
				SetRequired().
				IsString().
				IsEncrypted()),
	}

	basicAuthOptions = append(basicAuthOptions, options...)

	return NewCredentialRequestBuilder(basicAuthOptions...)
}

// NewOAuthCredentialRequestBuilder - add oauth base properties for provisioning schema
func NewOAuthCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	oauthOptions := []func(*crdBuilderOptions){
		WithCRDProvisionSchemaProperty(
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.OauthClientID).
				SetLabel("Client ID").
				SetRequired().
				IsString().
				IsCopyable()),
	}

	oauthOptions = append(oauthOptions, options...)

	return NewCredentialRequestBuilder(oauthOptions...)
}

// access request definitions

// createOrUpdateAccessRequestDefinition -
func createOrUpdateAccessRequestDefinition(data *management.AccessRequestDefinition) (*management.AccessRequestDefinition, error) {
	ri, err := createOrUpdateDefinition(data)
	if ri == nil || err != nil {
		return nil, err
	}
	err = data.FromInstance(ri)
	return data, err
}

// NewAccessRequestBuilder - called by the agents to build and register a new access request definition
func NewAccessRequestBuilder() provisioning.AccessRequestBuilder {
	return provisioning.NewAccessRequestBuilder(createOrUpdateAccessRequestDefinition)
}

// NewBasicAuthAccessRequestBuilder - called by the agents
func NewBasicAuthAccessRequestBuilder() provisioning.AccessRequestBuilder {
	return NewAccessRequestBuilder().SetName(provisioning.BasicAuthARD)
}

// NewAPIKeyAccessRequestBuilder - called by the agents
func NewAPIKeyAccessRequestBuilder() provisioning.AccessRequestBuilder {
	return NewAccessRequestBuilder().SetName(provisioning.APIKeyARD)
}

// provisioner

// RegisterProvisioner - allow the agent to register a provisioner
func RegisterProvisioner(provisioner provisioning.Provisioning) {
	if agent.agentFeaturesCfg == nil {
		return
	}
	agent.provisioner = provisioner

	if agent.cfg.GetAgentType() == config.DiscoveryAgent || agent.cfg.GetAgentType() == config.GovernanceAgent {
		agent.proxyResourceHandler.RegisterTargetHandler(
			"accessrequesthandler",
			handler.NewAccessRequestHandler(agent.provisioner, agent.cacheManager, agent.apicClient, agent.customUnitMetricServerManager),
		)
		agent.proxyResourceHandler.RegisterTargetHandler(
			"managedappHandler",
			handler.NewManagedApplicationHandler(agent.provisioner, agent.cacheManager, agent.apicClient),
		)
		registry := oauth.NewIdpRegistry(oauth.WithProviderRegistry(GetAuthProviderRegistry()))
		agent.proxyResourceHandler.RegisterTargetHandler(
			"credentialHandler",
			handler.NewCredentialHandler(agent.provisioner, agent.apicClient, registry),
		)
	}
}
