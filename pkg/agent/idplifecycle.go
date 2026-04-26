package agent

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/api"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func manageIDPResource(idpLogger log.FieldLogger, idp config.IDPConfig) string {
	metadataURL := idp.GetMetadataURL()
	envName := agent.cfg.GetEnvironmentName()
	idpLogger = idpLogger.WithField("environmentName", envName)

	idpURL := agent.cfg.GetEnvironmentURL() + "/" + management.IdentityProviderResourceName
	existing, err := agent.apicClient.GetAPIV1ResourceInstances(
		map[string]string{"query": fmt.Sprintf("spec.metadataUrl==\"%s\"", metadataURL)},
		idpURL,
	)
	if err != nil {
		idpLogger.WithError(err).Warn("unable to query for existing IdentityProvider resources; will attempt creation")
	}

	if len(existing) > 0 {
		name := existing[0].Name
		GetAuthProviderRegistry().SetIDPResourceName(metadataURL, name)
		idpLogger.WithField("name", name).Debug("reusing existing IdentityProvider resource")
		return name
	}

	createdName, err := createIDPResource(idpLogger, idp, metadataURL)
	if err != nil {
		return ""
	}
	return createdName
}

func createIDPResource(idpLogger log.FieldLogger, idp config.IDPConfig, metadataURL string) (string, error) {
	idpResource, err := buildIdentityProvider(idpLogger, idp, metadataURL)
	if err != nil {
		return "", err
	}

	envPolicies := getEnvCredentialPolicies(idpLogger)

	ri, err := agent.apicClient.CreateOrUpdateResource(idpResource)
	if err != nil {
		idpLogger.WithError(err).Error("unable to create IdentityProvider resource in Engage")
		return "", err
	}
	if err = idpResource.FromInstance(ri); err != nil {
		idpLogger.WithError(err).Error("failed to parse created IdentityProvider resource")
		return "", err
	}
	createdName := idpResource.Name

	applyIDPPolicies(idpLogger, idpResource, envPolicies)
	createIDPMetadataResource(idpLogger, idp, createdName)

	GetAuthProviderRegistry().SetIDPResourceName(metadataURL, createdName)
	idpLogger.WithField("name", createdName).Info("successfully created IdentityProvider resource")
	return createdName, nil
}

func buildIdentityProvider(idpLogger log.FieldLogger, idp config.IDPConfig, metadataURL string) (*management.IdentityProvider, error) {
	if s := agent.idpResourceSupplier; s != nil {
		res, err := s.GetIdentityProvider(idp)
		if err != nil {
			idpLogger.WithError(err).Error("supplier failed to build IdentityProvider resource")
			return nil, err
		}
		return res, nil
	}
	name := util.NormalizeNameForCentral(idp.GetIDPName())
	res := management.NewIdentityProvider(name)
	res.Spec = management.IdentityProviderSpec{
		MetadataUrl:  metadataURL,
		ProviderType: idp.GetIDPType(),
	}
	return res, nil
}

func getEnvCredentialPolicies(idpLogger log.FieldLogger) management.EnvironmentPoliciesCredentials {
	env, err := agent.apicClient.GetEnvironment()
	if err != nil {
		idpLogger.WithError(err).Warn("unable to retrieve environment credential policies; IdP will be created without policies")
		return management.EnvironmentPoliciesCredentials{}
	}
	return env.Policies.Credentials
}

func applyIDPPolicies(idpLogger log.FieldLogger, idpResource *management.IdentityProvider, envPolicies management.EnvironmentPoliciesCredentials) {
	if envPolicies.Expiry.Period == 0 && envPolicies.Visibility.Period == 0 {
		return
	}
	policies := management.IdentityProviderPolicies{
		Credentials: management.IdentityProviderPoliciesCredentials{
			Expiry: management.IdentityProviderPoliciesCredentialsExpiry{
				Period: envPolicies.Expiry.Period,
				Action: envPolicies.Expiry.Action,
				Notifications: management.IdentityProviderPoliciesCredentialsExpiryNotifications{
					DaysBefore: envPolicies.Expiry.Notifications.DaysBefore,
				},
			},
			Visibility: management.IdentityProviderPoliciesCredentialsVisibility{
				Period: envPolicies.Visibility.Period,
			},
		},
	}
	if err := agent.apicClient.CreateSubResource(idpResource.ResourceMeta, map[string]interface{}{
		management.IdentityProviderPoliciesSubResourceName: policies,
	}); err != nil {
		idpLogger.WithField("name", idpResource.Name).WithError(err).Warn("unable to set credential policies on IdentityProvider resource")
	}
}

func createIDPMetadataResource(idpLogger log.FieldLogger, idp config.IDPConfig, idpName string) {
	httpClient := api.NewClient(agent.cfg.GetTLSConfig(), agent.cfg.GetProxyURL())
	serverMetadata, err := oauth.FetchMetadata(httpClient, idp.GetMetadataURL())
	if err != nil {
		idpLogger.WithError(err).Error("unable to fetch IdP metadata; IdentityProviderMetadata resource will not be created")
		return
	}

	var idpMetadata *management.IdentityProviderMetadata
	if s := agent.idpResourceSupplier; s != nil {
		idpMetadata, err = s.GetIdentityProviderMetadata(idp, serverMetadata)
		if err != nil {
			idpLogger.WithError(err).Error("supplier failed to build IdentityProviderMetadata resource")
			return
		}
	} else {
		idpMetadata = NewIdentityProviderMetadataFromServerMetadata(idpName, idpName, serverMetadata)
	}

	if _, err = agent.apicClient.CreateOrUpdateResource(idpMetadata); err != nil {
		idpLogger.WithField("name", idpName).WithError(err).Error("unable to create IdentityProviderMetadata resource in Engage")
	}
}
