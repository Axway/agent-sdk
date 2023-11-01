package agent

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
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

// credential request definitions
// createOrUpdateDefinition -
func createOrUpdateDefinition(data v1.Interface, marketplaceMigration migrate.Migrator) (*v1.ResourceInstance, error) {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil, nil
	}

	ri, err := agent.apicClient.CreateOrUpdateResource(data)
	if err != nil {
		return nil, err
	}

	if runMarketplaceMigration(ri, marketplaceMigration) {
		migrateMarketPlace(marketplaceMigration, ri)
	}

	return ri, nil
}

func runMarketplaceMigration(ri *v1.ResourceInstance, marketplaceMigration migrate.Migrator) bool {
	// check if the KIND and ID combo have an item in the cache
	var existingRI *v1.ResourceInstance

	switch ri.Kind {
	case management.AccessRequestDefinitionGVK().Kind:
		existingRI, _ = agent.cacheManager.GetAccessRequestDefinitionByName(ri.Name)
	case management.CredentialRequestDefinitionGVK().Kind:
		existingRI, _ = agent.cacheManager.GetCredentialRequestDefinitionByName(ri.Name)
	}

	return existingRI == nil && marketplaceMigration != nil
}

// migrateMarketPlace -
func migrateMarketPlace(marketplaceMigration migrate.Migrator, ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	switch ri.Kind {
	case management.AccessRequestDefinitionGVK().Kind:
		agent.cacheManager.AddAccessRequestDefinition(ri)
	case management.CredentialRequestDefinitionGVK().Kind:
		agent.cacheManager.AddCredentialRequestDefinition(ri)
	}

	apiSvcResources := make([]*v1.ResourceInstance, 0)

	cache := agent.cacheManager.GetAPIServiceCache()

	for _, key := range cache.GetKeys() {
		item, _ := cache.Get(key)
		if item == nil {
			continue
		}

		svc, ok := item.(*v1.ResourceInstance)
		if ok {
			apiSvcResources = append(apiSvcResources, svc)
		}
	}

	for _, svc := range apiSvcResources {
		var err error

		mig := marketplaceMigration.(*migrate.MarketplaceMigration)

		logger.Tracef("update apiserviceinstances with request definition %s: %s", ri.Kind, ri.Name)

		mig.UpdateService(context.Background(), svc)

		// Mark marketplace migration completed here in provisioning
		util.SetAgentDetailsKey(svc, definitions.MarketplaceMigration, definitions.MigrationCompleted)
		ri, err = GetCentralClient().UpdateResourceInstance(svc)
		if err != nil {
			return nil, err
		}
		//update sub resources
		inst, err := svc.AsInstance()
		if xagentdetails, found := inst.SubResources[definitions.XAgentDetails]; found && err == nil {
			err = GetCentralClient().CreateSubResource(ri.ResourceMeta, map[string]interface{}{definitions.XAgentDetails: xagentdetails})
			if err != nil {
				return nil, err
			}
			log.Debugf("updated x-agent-details with marketplace-migration: completed")
		}

	}
	return ri, nil
}

// createOrUpdateCredentialRequestDefinition -
func createOrUpdateCredentialRequestDefinition(data *management.CredentialRequestDefinition) (*management.CredentialRequestDefinition, error) {
	ri, err := createOrUpdateDefinition(data, agent.marketplaceMigration)
	if ri == nil || err != nil {
		return nil, err
	}
	err = data.FromInstance(ri)
	return data, err
}

type crdBuilderOptions struct {
	name        string
	title       string
	renewable   bool
	suspendable bool
	provProps   []provisioning.PropertyBuilder
	reqProps    []provisioning.PropertyBuilder
}

// NewCredentialRequestBuilder - called by the agents to build and register a new credential reqest definition
func NewCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	thisCred := &crdBuilderOptions{
		renewable: false,
		provProps: make([]provisioning.PropertyBuilder, 0),
		reqProps:  make([]provisioning.PropertyBuilder, 0),
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

	builder := provisioning.NewCRDBuilder(createOrUpdateCredentialRequestDefinition).
		SetName(thisCred.name).
		SetTitle(thisCred.title).
		SetProvisionSchema(provSchema).
		SetRequestSchema(reqSchema).
		SetExpirationDays(agent.cfg.GetCredentialConfig().GetExpirationDays())

	if thisCred.renewable {
		builder.IsRenewable()
	}

	if thisCred.suspendable {
		builder.IsSuspendable()
	}

	if agent.cfg.GetCredentialConfig().ShouldDeprovisionExpired() {
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

// WithCRDName - set another name for the CRD
func WithCRDTitle(title string) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.title = title
	}
}

// WithCRDIsRenewable - set another name for the CRD
func WithCRDIsRenewable() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.renewable = true
	}
}

// WithCRDIsSuspendable - set another name for the CRD
func WithCRDIsSuspendable() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.suspendable = true
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
		setIDPScopesSchemaProperty(p, scopes, c)
		setIDPGrantTypesSchemaProperty(p, c)
		tokenAuthMethods := setIDPTokenAuthMethodSchemaProperty(p, c)
		setIDPRedirectURIsSchemaProperty(p, c)

		usePrivateKeyJWTAuth := idpUsesPrivateKeyJWTAuth(tokenAuthMethods)
		useTLSClientAuth := idpUsesTLSClientAuth(tokenAuthMethods)
		if usePrivateKeyJWTAuth || useTLSClientAuth {
			setIDPJWKSURISchemaProperty(p, c)
		}

		if usePrivateKeyJWTAuth {
			setIDPJWKSSchemaProperty(p, c)
		}

		if useTLSClientAuth {
			setIDPTLSClientAuthSchemaProperty(p, c)
		}
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

func setIDPScopesSchemaProperty(p oauth.Provider, scopes []string, c *crdBuilderOptions) {
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

	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTokenAuthMethod).
			SetLabel("Token Auth Method").
			IsString().
			SetDefaultValue(defaultTokenMethod).
			SetEnumValues(tokenAuthMethods))
	return tokenAuthMethods
}

func setIDPRedirectURIsSchemaProperty(p oauth.Provider, c *crdBuilderOptions) {
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

func setIDPJWKSURISchemaProperty(p oauth.Provider, c *crdBuilderOptions) {
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthJwksURI).
			SetLabel("JWKS URI").
			IsString())
}

func setIDPJWKSSchemaProperty(p oauth.Provider, c *crdBuilderOptions) {
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthJwks).
			SetLabel("Public Key").
			IsString())

}

func setIDPTLSClientAuthSchemaProperty(p oauth.Provider, c *crdBuilderOptions) {
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthCertificate).
			SetLabel("Public Certificate").
			IsString())
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthCertificateMetadata).
			SetLabel("Certificate Metadata").
			IsString().
			SetDefaultValue(oauth.TLSClientAuthSubjectDN).
			SetEnumValues(tlsAuthCertificateMetadata))
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTLSAuthSANDNS).
			SetLabel("Certificate Subject Alternative Name, DNS").
			IsString())
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTLSAuthSANEmail).
			SetLabel("Certificate Subject Alternative Name, Email").
			IsString())
	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTLSAuthSANIP).
			SetLabel("Certificate Subject Alternative Name, IP address").
			IsString())
	c.reqProps = append(c.reqProps,
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
	ri, err := createOrUpdateDefinition(data, agent.marketplaceMigration)
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
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return
	}
	agent.provisioner = provisioner

	if agent.cfg.GetAgentType() == config.DiscoveryAgent || agent.cfg.GetAgentType() == config.GovernanceAgent {
		agent.proxyResourceHandler.RegisterTargetHandler(
			"accessrequesthandler",
			handler.NewAccessRequestHandler(agent.provisioner, agent.cacheManager, agent.apicClient),
		)
		agent.proxyResourceHandler.RegisterTargetHandler(
			"managedappHandler",
			handler.NewManagedApplicationHandler(agent.provisioner, agent.cacheManager, agent.apicClient),
		)
		agent.proxyResourceHandler.RegisterTargetHandler(
			"credentialHandler",
			handler.NewCredentialHandler(agent.provisioner, agent.apicClient, agent.authProviderRegistry),
		)
	}
}
