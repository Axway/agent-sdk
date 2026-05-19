package agent

import (
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func GetIDPCredentialExpiryPolicy(idpResourceName string) management.IdentityProviderPoliciesCredentialsExpiry {
	expPolicies := management.IdentityProviderPoliciesCredentialsExpiry{}
	if !ManageIDPResourcesEnabled() {
		return expPolicies
	}

	if idpResourceName == "" || agent.cacheManager == nil {
		return expPolicies
	}

	ri := agent.cacheManager.GetIdentityProviderByName(idpResourceName)
	if ri != nil {
		idp := management.NewIdentityProvider("")
		if err := idp.FromInstance(ri); err == nil {
			return idp.Policies.Credentials.Expiry
		}
	}

	ri, err := agent.apicClient.GetResource(management.NewIdentityProvider(idpResourceName).GetSelfLink())
	if err != nil || ri == nil {
		return expPolicies
	}

	idp := management.NewIdentityProvider("")
	if err = idp.FromInstance(ri); err != nil {
		return expPolicies
	}

	agent.cacheManager.AddIdentityProvider(ri)
	return idp.Policies.Credentials.Expiry
}

func manageIDPResource(idpLogger log.FieldLogger, idp config.IDPConfig) string {
	if !ManageIDPResourcesEnabled() {
		return ""
	}

	idpLogger = idpLogger.WithField("environmentName", agent.cfg.GetEnvironmentName())

	provider, err := GetAuthProviderRegistry().GetProviderByName(idp.GetIDPName())
	if err != nil {
		idpLogger.WithError(err).Error("unable to retrieve registered IdP provider; cannot manage IdentityProvider resource")
		return ""
	}

	name := manageIDPResourceFromMetadata(idpLogger, nil, idp.GetIDPName(), provider.GetMetadata())
	if name != "" {
		GetAuthProviderRegistry().SetIDPResourceName(idp.GetMetadataURL(), name)
	}
	return name
}

func ManageIDPResourcesEnabled() bool {
	return agent.agentFeaturesCfg != nil && agent.agentFeaturesCfg.ManageIDPResourcesEnabled()
}

// ManageIDPResource creates or reuses an IdentityProvider resource in Engage using
// pre-resolved metadata. Public entry point for agents like v7 that supply metadata
// directly without a discovery URL.
// Returns the Engage IdentityProvider resource name, or empty string on failure.
func ManageIDPResource(idpLogger log.FieldLogger, idpName string, metadata *oauth.AuthorizationServerMetadata) string {
	if !ManageIDPResourcesEnabled() {
		return ""
	}

	if metadata == nil {
		idpLogger.Error("metadata is nil; cannot manage IdentityProvider resource")
		return ""
	}

	return manageIDPResourceFromMetadata(idpLogger, nil, idpName, metadata)
}

// manageIDPResourceFromMetadata is the shared internal entry point for both paths.
func manageIDPResourceFromMetadata(idpLogger log.FieldLogger, idpCfg config.IDPConfig, idpName string, metadata *oauth.AuthorizationServerMetadata) string {
	if metadata == nil {
		idpLogger.Error("metadata is nil; cannot manage IdentityProvider resource")
		return ""
	}

	idpLogger = idpLogger.WithField("issuer", metadata.Issuer)

	idpType := oauth.TypeGeneric
	if idpCfg != nil {
		idpName = idpCfg.GetIDPName()
		idpType = idpCfg.GetIDPType()
	}

	idpLifecycle := oauth.NewIDPEngageLifecycle(agent.apicClient, agent.cacheManager)
	name, err := idpLifecycle.CreateEngageResourcesFromMetadata(idpLogger, idpCfg, idpType, idpName, metadata, agent.cfg.GetAPIServerVersionURL(), getEnvCredentialPolicies(idpLogger))
	if err != nil {
		idpLogger.WithError(err).Warn("unable to create or find IdentityProvider resource in Engage")
		return ""
	}
	return name
}

func getEnvCredentialPolicies(idpLogger log.FieldLogger) management.EnvironmentPoliciesCredentials {
	env, err := agent.apicClient.GetEnvironment()
	if err != nil {
		idpLogger.WithError(err).Warn("unable to retrieve environment credential policies; IdP will be created without policies")
		return management.EnvironmentPoliciesCredentials{}
	}
	return env.Policies.Credentials
}
