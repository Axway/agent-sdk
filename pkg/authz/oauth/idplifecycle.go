package oauth

import (
	"fmt"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// IDPClient is the subset of apic.Client used by the IdP lifecycle manager,
// defined here to avoid a circular import with pkg/apic.
type IDPClient interface {
	GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*apiv1.ResourceInstance, error)
	CreateOrUpdateResource(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	CreateSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) error
}

// IDPResourceBuilder is an optional override for building IdentityProvider and
// IdentityProviderMetadata resources. Agent implementations that supply custom
// resources set this via WithResourceBuilder.
type IDPResourceBuilder interface {
	GetIdentityProvider(cfg corecfg.IDPConfig) (*management.IdentityProvider, error)
	GetIdentityProviderMetadata(cfg corecfg.IDPConfig, metadata *AuthorizationServerMetadata) (*management.IdentityProviderMetadata, error)
}

// IDPEngageLifecycle manages IdentityProvider and IdentityProviderMetadata resources in Engage.
type IDPEngageLifecycle interface {
	// CreateEngageResourcesFromMetadata creates or reuses IdentityProvider and IdentityProviderMetadata
	// resources in Engage using pre-resolved metadata — no Provider or outbound HTTP fetch required.
	// idpCfg is optional (may be nil for the v7 path); when set it is passed to the IDPResourceBuilder.
	// Returns the Engage IdentityProvider resource name.
	CreateEngageResourcesFromMetadata(idpLogger log.FieldLogger, idpCfg corecfg.IDPConfig, idpName string, metadata *AuthorizationServerMetadata, envURL string, envPolicies management.EnvironmentPoliciesCredentials) (string, error)
}

// LifecycleOption configures an idpEngageLifecycle.
type LifecycleOption func(*idpEngageLifecycle)

// WithResourceBuilder injects an optional custom builder for IdP and metadata resources.
func WithResourceBuilder(b IDPResourceBuilder) LifecycleOption {
	return func(l *idpEngageLifecycle) { l.builder = b }
}

type idpEngageLifecycle struct {
	client  IDPClient
	builder IDPResourceBuilder
}

// NewIDPEngageLifecycle returns an IDPEngageLifecycle backed by the given Engage client.
func NewIDPEngageLifecycle(client IDPClient, opts ...LifecycleOption) IDPEngageLifecycle {
	l := &idpEngageLifecycle{client: client}
	for _, o := range opts {
		o(l)
	}
	return l
}

func (l *idpEngageLifecycle) CreateEngageResourcesFromMetadata(idpLogger log.FieldLogger, idpCfg corecfg.IDPConfig, idpName string, metadata *AuthorizationServerMetadata, envURL string, envPolicies management.EnvironmentPoliciesCredentials) (string, error) {
	metadataURL := metadata.Issuer
	if idpCfg != nil {
		metadataURL = idpCfg.GetMetadataURL()
	}

	idpLogger.Debug("querying Engage for existing IdentityProvider resource")
	existing, err := l.client.GetAPIV1ResourceInstances(
		map[string]string{"query": fmt.Sprintf("spec.metadataUrl==\"%s\"", metadataURL)},
		envURL+"/"+management.IdentityProviderResourceName,
	)
	if err != nil {
		return "", err
	}

	if len(existing) > 0 {
		name := existing[0].Name
		idpLogger.WithField("name", name).Info("reusing existing IdentityProvider resource")
		return name, nil
	}

	idpResource, err := l.buildIdentityProviderFromMetadata(idpLogger, idpCfg, idpName, metadataURL)
	if err != nil {
		return "", err
	}

	ri, err := l.client.CreateOrUpdateResource(idpResource)
	if err != nil {
		idpLogger.WithError(err).Error("unable to create IdentityProvider resource in Engage")
		return "", err
	}
	if err = idpResource.FromInstance(ri); err != nil {
		idpLogger.WithError(err).Error("failed to parse created IdentityProvider resource")
		return "", err
	}
	createdName := idpResource.Name

	if err = l.applyPolicies(idpLogger, idpResource, envPolicies); err != nil {
		return "", err
	}

	if err = l.createMetadataFromServerMetadata(idpLogger, idpCfg, createdName, metadata); err != nil {
		return "", err
	}

	idpLogger.WithField("name", createdName).Info("IdentityProvider resource created successfully")
	return createdName, nil
}

func (l *idpEngageLifecycle) buildIdentityProviderFromMetadata(idpLogger log.FieldLogger, idpCfg corecfg.IDPConfig, idpName, metadataURL string) (*management.IdentityProvider, error) {
	if l.builder != nil && idpCfg != nil {
		idpLogger.Debug("building IdentityProvider resource via supplier")
		res, err := l.builder.GetIdentityProvider(idpCfg)
		if err != nil {
			idpLogger.WithError(err).Error("supplier failed to build IdentityProvider resource")
			return nil, err
		}
		return res, nil
	}

	name := util.NormalizeNameForCentral(idpName)
	res := management.NewIdentityProvider(name)
	res.Spec = management.IdentityProviderSpec{
		MetadataUrl: metadataURL,
	}
	return res, nil
}

func (l *idpEngageLifecycle) createMetadataFromServerMetadata(idpLogger log.FieldLogger, idpCfg corecfg.IDPConfig, idpName string, metadata *AuthorizationServerMetadata) error {
	idpLogger.WithField("name", idpName).Debug("creating IdentityProviderMetadata resource")

	idpMetadata := newIdentityProviderMetadata(idpName, idpName, metadata)

	if l.builder != nil && idpCfg != nil {
		idpLogger.Debug("building IdentityProviderMetadata resource via supplier")
		supplied, err := l.builder.GetIdentityProviderMetadata(idpCfg, metadata)
		if err != nil {
			idpLogger.WithError(err).Warn("supplier failed to build IdentityProviderMetadata resource; skipping metadata creation")
			return nil
		}
		idpMetadata = supplied
	}

	if _, err := l.client.CreateOrUpdateResource(idpMetadata); err != nil {
		idpLogger.WithField("name", idpName).WithError(err).Warn("unable to create IdentityProviderMetadata resource in Engage")
		return err
	}
	idpLogger.WithField("name", idpName).Info("IdentityProviderMetadata resource created successfully")
	return nil
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
	res.Spec = management.IdentityProviderMetadataSpec{
		Issuer:                m.Issuer,
		AuthorizationEndpoint: m.AuthorizationEndpoint,
		TokenEndpoint:         m.TokenEndpoint,
		IntrospectionEndpoint: m.IntrospectionEndpoint,
		JwksUri:               m.JwksURI,
	}
	return res
}
