package agent

import (
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
	"client_credentials": true,
	"authorization_code": true}

var supportedIDPTokenAuthMethods = map[string]bool{
	"client_secret_basic": true,
	"client_secret_post":  true,
	"client_secret_jwt":   true,
	"private_key_jwt":     true}

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

	if marketplaceMigration != nil {
		migrateMarketPlace(marketplaceMigration, ri)
	}

	return ri, nil
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
		alreadyMigrated := mig.InstanceAlreadyMigrated(svc)

		// Check if migration already happened for apiservice
		if !alreadyMigrated {
			logger.Tracef("update apiserviceinstances with request definition %s: %s", ri.Kind, ri.Name)

			mig.UpdateService(svc)

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
	name      string
	renewable bool
	provProps []provisioning.PropertyBuilder
	reqProps  []provisioning.PropertyBuilder
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
		SetProvisionSchema(provSchema).
		SetRequestSchema(reqSchema).
		SetExpirationDays(agent.cfg.GetCredentialConfig().GetExpirationDays())

	if thisCred.renewable {
		builder.IsRenewable()
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

// WithCRDIsRenewable - set another name for the CRD
func WithCRDIsRenewable() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		c.renewable = true
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

// WithCRDForIDP - set the schema properties using the provider metadata
func WithCRDForIDP(p oauth.Provider, scopes []string) func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		if c.name == "" {
			name := util.ConvertToDomainNameCompliant(p.GetName())
			c.name = name + "-" + provisioning.OAuthIDPCRD
		}

		setIDPClientSecretSchemaProperty(c)
		setIDPTokenURLSchemaProperty(p, c)
		setIDPScopesSchemaProperty(p, scopes, c)
		setIDPGrantTypesSchemaProperty(p, c)
		setIDPTokenAuthMethodSchemaProperty(p, c)
		setIDPRedirectURIsSchemaProperty(p, c)
		setIDPJWKSURISchemaProperty(p, c)
		setIDPJWKSSchemaProperty(p, c)
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
					IsString().SetEnumValues(scopes)))
}

func setIDPGrantTypesSchemaProperty(p oauth.Provider, c *crdBuilderOptions) {
	grantType := removeUnsupportedTypes(
		p.GetSupportedGrantTypes(), supportedIDPGrantTypes)

	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthGrantType).
			SetLabel("Grant Type").
			IsString().
			SetDefaultValue("client_credentials").
			SetEnumValues(grantType))
}

func removeUnsupportedTypes(values []string, supportedTypes map[string]bool) []string {
	var result []string
	for _, s := range values {
		if ok := supportedTypes[s]; ok {
			result = append(result, s)
		}
	}
	return result
}

func setIDPTokenAuthMethodSchemaProperty(p oauth.Provider, c *crdBuilderOptions) {
	tokenAuthMethod := removeUnsupportedTypes(
		p.GetSupportedTokenAuthMethods(), supportedIDPTokenAuthMethods)

	c.reqProps = append(c.reqProps,
		provisioning.NewSchemaPropertyBuilder().
			SetName(provisioning.OauthTokenAuthMethod).
			SetLabel("Token Auth Method").
			IsString().
			SetDefaultValue("client_secret_basic").
			SetEnumValues(tokenAuthMethod))
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

// WithCRDOAuthSecret - set that the Oauth cred is secret based
func WithCRDOAuthSecret() func(c *crdBuilderOptions) {
	return func(c *crdBuilderOptions) {
		if c.name == "" {
			c.name = provisioning.OAuthSecretCRD
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

// NewOAuthCredentialRequestBuilder - add oauth base properties for provisioning schema
func NewOAuthCredentialRequestBuilder(options ...func(*crdBuilderOptions)) provisioning.CredentialRequestBuilder {
	oauthOptions := []func(*crdBuilderOptions){
		WithCRDProvisionSchemaProperty(
			provisioning.NewSchemaPropertyBuilder().
				SetName(provisioning.OauthClientID).
				SetLabel("Client ID").
				SetRequired().
				IsString()),
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
