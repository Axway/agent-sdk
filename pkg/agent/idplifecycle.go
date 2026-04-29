package agent

import (
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func manageIDPResource(idpLogger log.FieldLogger, idp config.IDPConfig) string {
	idpLogger = idpLogger.WithField("environmentName", agent.cfg.GetEnvironmentName())

	provider, err := GetAuthProviderRegistry().GetProviderByName(idp.GetIDPName())
	if err != nil {
		idpLogger.WithError(err).Error("unable to retrieve registered IdP provider; cannot manage IdentityProvider resource")
		return ""
	}

	name, err := newLifecycle().CreateEngageResources(idpLogger, provider, agent.cfg.GetEnvironmentURL(), getEnvCredentialPolicies(idpLogger))
	if err != nil {
		idpLogger.WithError(err).Warn("unable to create or find IdentityProvider resource in Engage")
		return ""
	}
	GetAuthProviderRegistry().SetIDPResourceName(idp.GetMetadataURL(), name)
	return name
}

func manageIDPResourceWithMetadata(idpLogger log.FieldLogger, idp config.IDPConfig, metadata *oauth.AuthorizationServerMetadata) string {
	if metadata == nil ||
		metadata.Issuer == "" ||
		metadata.AuthorizationEndpoint == "" ||
		metadata.TokenEndpoint == "" ||
		metadata.IntrospectionEndpoint == "" ||
		metadata.JwksURI == "" {
		idpLogger.Error("agent-supplied metadata is missing one or more required fields: issuer, authorizationEndpoint, tokenEndpoint, introspectionEndpoint, jwksUri")
		return ""
	}

	idpLogger = idpLogger.WithField("environmentName", agent.cfg.GetEnvironmentName()).WithField("issuer", metadata.Issuer)

	tlsCfg := idp.GetTLSConfig()
	if tlsCfg == nil {
		tlsCfg = agent.cfg.GetTLSConfig()
	}
	if err := GetAuthProviderRegistry().RegisterProviderWithMetadata(idp, metadata, tlsCfg, agent.cfg.GetProxyURL(), agent.cfg.GetClientTimeout()); err != nil {
		idpLogger.WithError(err).Error("unable to register IdP provider with supplied metadata; cannot manage IdentityProvider resource")
		return ""
	}

	provider, err := GetAuthProviderRegistry().GetProviderByName(idp.GetIDPName())
	if err != nil {
		idpLogger.WithError(err).Error("unable to retrieve registered IdP provider; cannot manage IdentityProvider resource")
		return ""
	}

	name, err := newLifecycle().CreateEngageResources(idpLogger, provider, agent.cfg.GetEnvironmentURL(), getEnvCredentialPolicies(idpLogger))
	if err != nil {
		idpLogger.WithError(err).Warn("unable to create or find IdentityProvider resource in Engage")
		return ""
	}
	GetAuthProviderRegistry().SetIDPResourceName(idp.GetMetadataURL(), name)
	return name
}

// newLifecycle builds an IDPEngageLifecycle, injecting the optional supplier if one is registered.
func newLifecycle() oauth.IDPEngageLifecycle {
	opts := []oauth.LifecycleOption{}
	if s := agent.idpResourceSupplier; s != nil {
		opts = append(opts, oauth.WithResourceBuilder(s))
	}
	return oauth.NewIDPEngageLifecycle(agent.apicClient, opts...)
}

func getEnvCredentialPolicies(idpLogger log.FieldLogger) management.EnvironmentPoliciesCredentials {
	env, err := agent.apicClient.GetEnvironment()
	if err != nil {
		idpLogger.WithError(err).Warn("unable to retrieve environment credential policies; IdP will be created without policies")
		return management.EnvironmentPoliciesCredentials{}
	}
	return env.Policies.Credentials
}
