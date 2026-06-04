package oauth

import (
	"fmt"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	defaultIDPClientTimeoutSeconds = 60
)

type idpCache interface {
	GetIdentityProviderMetadataByTokenUrl(tokenURL string) *apiv1.ResourceInstance
	AddIdentityProviderMetadata(resource *apiv1.ResourceInstance)
}

// IDPClient is the subset of apic.Client used by the IdP lifecycle manager,
// defined here to avoid a circular import with pkg/apic.
type IDPClient interface {
	GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*apiv1.ResourceInstance, error)
	CreateOrUpdateResource(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	CreateSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) error
	GetResource(url string) (*apiv1.ResourceInstance, error)
}

// IDPEngageLifecycle manages IdentityProvider and IdentityProviderMetadata resources in Engage.
type IDPEngageLifecycle interface {
	// CreateEngageResourcesFromMetadata creates or reuses IdentityProvider and IdentityProviderMetadata
	// resources in Engage using pre-resolved metadata — no Provider or outbound HTTP fetch required.
	// Returns the Engage IdentityProvider resource name.
	CreateEngageResourcesFromMetadata(idpLogger log.FieldLogger, idpCfg corecfg.IDPConfig, idpType, idpName string, metadata *AuthorizationServerMetadata, baseURL string, envPolicies management.EnvironmentPoliciesCredentials) (string, error)
}

// LifecycleOption configures an idpEngageLifecycle.
type LifecycleOption func(*idpEngageLifecycle)

type idpEngageLifecycle struct {
	client   IDPClient
	idpCache idpCache
}

// NewIDPEngageLifecycle returns an IDPEngageLifecycle backed by the given Engage client.
func NewIDPEngageLifecycle(client IDPClient, cache idpCache, opts ...LifecycleOption) IDPEngageLifecycle {
	l := &idpEngageLifecycle{client: client, idpCache: cache}
	for _, o := range opts {
		o(l)
	}
	return l
}

func (l *idpEngageLifecycle) CreateEngageResourcesFromMetadata(idpLogger log.FieldLogger, idpCfg corecfg.IDPConfig, idpType, idpName string, metadata *AuthorizationServerMetadata, baseURL string, envPolicies management.EnvironmentPoliciesCredentials) (string, error) {
	if metadata == nil || l.idpCache == nil {
		return "", fmt.Errorf("metadata and cache are required to manage IdentityProvider resources")
	}

	tokenEndpoint := metadata.TokenEndpoint

	// Attempt to find an existing IdentityProviderMetadat in cache
	idpMetadata := l.idpCache.GetIdentityProviderMetadataByTokenUrl(tokenEndpoint)
	if idpMetadata != nil {
		idpLogger.WithField("name", idpMetadata.GetMetadata().Scope.Name).Debug("reusing existing IdentityProvider resource")
		return idpMetadata.GetMetadata().Scope.Name, nil
	}

	idpLogger.Debug("querying Engage for existing IdentityProvider resource")
	existing, err := l.client.GetAPIV1ResourceInstances(
		map[string]string{"query": fmt.Sprintf("spec.tokenEndpoint==\"%s\"", tokenEndpoint)},
		baseURL+"/"+management.NewIdentityProviderMetadata("", "").PluralName(),
	)
	if err != nil {
		return "", err
	}

	if len(existing) > 0 {
		name := existing[0].GetMetadata().Scope.Name
		l.idpCache.AddIdentityProviderMetadata(existing[0])
		idpLogger.WithField("name", name).Debug("reusing existing IdentityProvider resource")
		return name, nil
	}

	idpResource, err := l.buildIdentityProviderFromMetadata(idpLogger, idpCfg, idpType, idpName)
	if err != nil {
		return "", err
	}

	// Check if an IdentityProvider with the same name already exists
	ri, err := l.client.GetResource(idpResource.GetSelfLink())
	if err != nil {
		return "", err
	}

	if ri == nil {
		ri, err = l.client.CreateOrUpdateResource(idpResource)
		if err != nil {
			idpLogger.WithError(err).Error("unable to create IdentityProvider resource in Engage")
			return "", err
		}
		if err = idpResource.FromInstance(ri); err != nil {
			idpLogger.WithError(err).Error("failed to parse created IdentityProvider resource")
			return "", err
		}
		if err = l.applyPolicies(idpLogger, idpResource, envPolicies); err != nil {
			return "", err
		}
		idpLogger.WithField("name", ri.GetName()).Info("IdentityProvider resource created successfully")
	}

	createdName := idpResource.Name
	if ri != nil {
		createdName = ri.GetName()
	}

	if idpMetadata, err = l.createMetadataFromServerMetadata(idpLogger, idpCfg, createdName, metadata); err != nil {
		return "", err
	}
	idpLogger.WithField("idpName", createdName).
		WithField("name", idpMetadata.GetName()).
		Info("IdentityProviderMetadata resource created successfully")
	return createdName, nil
}

func (l *idpEngageLifecycle) buildIdentityProviderFromMetadata(idpLogger log.FieldLogger, idpCfg corecfg.IDPConfig, idpType, idpName string) (*management.IdentityProvider, error) {
	name := util.NormalizeNameForCentral(idpName)
	res := management.NewIdentityProvider(name)
	res.Title = name
	res.Spec = management.IdentityProviderSpec{
		ProviderType:  idpType,
		ClientTimeout: defaultIDPClientTimeoutSeconds,
	}
	return res, nil
}

func (l *idpEngageLifecycle) createMetadataFromServerMetadata(idpLogger log.FieldLogger, idpCfg corecfg.IDPConfig, idpName string, metadata *AuthorizationServerMetadata) (*apiv1.ResourceInstance, error) {
	idpLogger.WithField("name", idpName).Debug("creating IdentityProviderMetadata resource")

	idpMetadata := newIdentityProviderMetadata("", idpName, metadata)

	ri, err := l.client.CreateOrUpdateResource(idpMetadata)
	if err != nil {
		idpLogger.WithField("name", idpName).WithError(err).Warn("unable to create IdentityProviderMetadata resource in Engage")
		return nil, err
	}
	l.idpCache.AddIdentityProviderMetadata(ri)

	idpLogger.WithField("name", idpName).Info("IdentityProviderMetadata resource created successfully")
	return ri, nil
}

func (l *idpEngageLifecycle) applyPolicies(idpLogger log.FieldLogger, idpResource *management.IdentityProvider, envPolicies management.EnvironmentPoliciesCredentials) error {
	if envPolicies.Expiry.Period == 0 && envPolicies.Visibility.Period == 0 {
		return nil
	}

	idpLogger.WithField("name", idpResource.Name).Debug("applying credential policies to IdentityProvider")
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
	if err := l.client.CreateSubResource(idpResource.ResourceMeta, map[string]interface{}{
		management.IdentityProviderPoliciesSubResourceName: policies,
	}); err != nil {
		idpLogger.WithField("name", idpResource.Name).WithError(err).Error("unable to set credential policies on IdentityProvider resource")
		return err
	}
	idpLogger.WithField("name", idpResource.Name).Info("credential policies applied to IdentityProvider")
	return nil
}

func newIdentityProviderMetadata(name, scopeName string, m *AuthorizationServerMetadata) *management.IdentityProviderMetadata {
	res := management.NewIdentityProviderMetadata(name, scopeName)
	res.Title = name
	res.Spec = management.IdentityProviderMetadataSpec{
		Issuer:                m.Issuer,
		AuthorizationEndpoint: m.AuthorizationEndpoint,
		TokenEndpoint:         m.TokenEndpoint,
		IntrospectionEndpoint: m.IntrospectionEndpoint,
		JwksUri:               m.JwksURI,
	}
	return res
}
