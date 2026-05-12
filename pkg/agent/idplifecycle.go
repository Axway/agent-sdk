package agent

import (
	"sync"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// idpMetadataResourceCache is a local cache for v7 IdP metadata resource name lookups,
// keyed by token endpoint. Separate from ProviderRegistry — v7 agents have no registered Provider.
var (
	idpMetadataResourceCache     = map[string]string{}
	idpMetadataResourceCacheLock sync.RWMutex
)

func getIDPMetadataResourceName(tokenEndpoint string) (string, bool) {
	idpMetadataResourceCacheLock.RLock()
	defer idpMetadataResourceCacheLock.RUnlock()
	name, ok := idpMetadataResourceCache[tokenEndpoint]
	return name, ok
}

func setIDPMetadataResourceName(tokenEndpoint, name string) {
	idpMetadataResourceCacheLock.Lock()
	defer idpMetadataResourceCacheLock.Unlock()
	idpMetadataResourceCache[tokenEndpoint] = name
}

func manageIDPResource(idpLogger log.FieldLogger, idp config.IDPConfig) string {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.ManageIDPResourcesEnabled() {
		return ""
	}

	idpLogger = idpLogger.WithField("environmentName", agent.cfg.GetEnvironmentName())

	provider, err := GetAuthProviderRegistry().GetProviderByName(idp.GetIDPName())
	if err != nil {
		idpLogger.WithError(err).Error("unable to retrieve registered IdP provider; cannot manage IdentityProvider resource")
		return ""
	}

	name := manageIDPResourceFromMetadata(idpLogger, idp, "", provider.GetMetadata())
	if name == "" {
		idpLogger.Warn("IdentityProvider resource could not be created or found; CRD will be registered without an IdentityProvider reference")
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

	idpLogger = idpLogger.WithField("environmentName", agent.cfg.GetEnvironmentName())

	name := manageIDPResourceFromMetadata(idpLogger, idp, "", metadata)
	if name != "" {
		GetAuthProviderRegistry().SetIDPResourceName(idp.GetMetadataURL(), name)
	}
	return name
}

// ManageIDPResource creates or reuses an IdentityProvider resource in Engage using
// pre-resolved metadata. Public entry point for agents like v7 that supply metadata
// directly without a discovery URL.
// Returns the Engage IdentityProvider resource name, or empty string on failure.
func ManageIDPResource(idpLogger log.FieldLogger, idpName string, metadata *oauth.AuthorizationServerMetadata) string {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.ManageIDPResourcesEnabled() {
		return ""
	}

	if metadata == nil {
		idpLogger.Error("metadata is nil; cannot manage IdentityProvider resource")
		return ""
	}

	// find existing idp with token url — check local cache first
	if name, ok := getIDPMetadataResourceName(metadata.TokenEndpoint); ok {
		idpLogger.WithField("name", name).Debug("found IdentityProvider resource name in local cache")
		return name
	}

	// not found in cache — create the IdP resource and cache it by token endpoint
	name := manageIDPResourceFromMetadata(idpLogger, nil, idpName, metadata)
	if name != "" {
		setIDPMetadataResourceName(metadata.TokenEndpoint, name)
	}
	return name
}

// manageIDPResourceFromMetadata is the shared internal entry point for both paths.
// idpCfg is passed to the IDPResourceBuilder when a supplier is registered; may be nil for the v7 path.
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

	name, err := newLifecycle().CreateEngageResourcesFromMetadata(idpLogger, idpCfg, idpType, idpName, metadata, agent.cfg.GetAPIServerVersionURL(), getEnvCredentialPolicies(idpLogger))
	if err != nil {
		idpLogger.WithError(err).Warn("unable to create or find IdentityProvider resource in Engage")
		return ""
	}
	return name
}

// newLifecycle builds an IDPEngageLifecycle, injecting the optional supplier if one is registered.
func newLifecycle() oauth.IDPEngageLifecycle {
	var opts []oauth.LifecycleOption
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
