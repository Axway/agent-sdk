package oauth

import (
	"context"
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

type ConfigOption func(config.IDPConfig) error

type IdPRegistry interface {
	// RegisterProvider - registers the provider using the config
	RegisterProvider(ctx context.Context, idp corecfg.IDPConfig, tlsCfg corecfg.TLSConfig, proxyURL string, clientTimeout time.Duration) error
	// UnregisterProvider - un-registers the provider
	UnregisterProvider(ctx context.Context, provider Provider) error
	// GetProviderByName - returns the provider from registry based on the name
	GetProviderByName(ctx context.Context, name string, opts ...ConfigOption) (Provider, error)
	// GetProviderByIssuer - returns the provider from registry based on the IDP issuer
	GetProviderByIssuer(ctx context.Context, issuer string, opts ...ConfigOption) (Provider, error)
	// GetProviderByTokenEndpoint - returns the provider from registry based on the IDP token endpoint
	GetProviderByTokenEndpoint(ctx context.Context, tokenEndpoint string, opts ...ConfigOption) (Provider, error)
	// GetProviderByAuthorizationEndpoint - returns the provider from registry based on the IDP authorization endpoint
	GetProviderByAuthorizationEndpoint(ctx context.Context, authEndpoint string, opts ...ConfigOption) (Provider, error)
	// GetProviderByMetadataURL - returns the provider from registry based on the IDP metadata URL
	GetProviderByMetadataURL(ctx context.Context, metadataURL string, opts ...ConfigOption) (Provider, error)
}

type idpRegistry struct {
	registry ProviderRegistry
}
type IdpRegistryOption func(r *idpRegistry)

func WithProviderRegistry(providerRegistry ProviderRegistry) IdpRegistryOption {
	return func(r *idpRegistry) {
		r.registry = providerRegistry
	}
}

// NewProviderRegistry - create a new provider registry
func NewIdpRegistry(opts ...IdpRegistryOption) IdPRegistry {
	r := &idpRegistry{
		registry: NewProviderRegistry(),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

func (p *idpRegistry) RegisterProvider(_ context.Context, idp corecfg.IDPConfig, tlsCfg corecfg.TLSConfig, proxyURL string, clientTimeout time.Duration) error {
	return p.registry.RegisterProvider(idp, tlsCfg, proxyURL, clientTimeout)
}

func (p *idpRegistry) UnregisterProvider(_ context.Context, provider Provider) error {
	return fmt.Errorf("not implemented")
}

func (p *idpRegistry) GetProviderByName(_ context.Context, name string, _ ...ConfigOption) (Provider, error) {
	return p.registry.GetProviderByName(name)
}

func (p *idpRegistry) GetProviderByIssuer(_ context.Context, issuer string, _ ...ConfigOption) (Provider, error) {
	return p.registry.GetProviderByIssuer(issuer)
}

func (p *idpRegistry) GetProviderByTokenEndpoint(_ context.Context, tokenEndpoint string, _ ...ConfigOption) (Provider, error) {
	return p.registry.GetProviderByTokenEndpoint(tokenEndpoint)
}

func (p *idpRegistry) GetProviderByAuthorizationEndpoint(_ context.Context, authEndpoint string, _ ...ConfigOption) (Provider, error) {
	return p.registry.GetProviderByAuthorizationEndpoint(authEndpoint)
}

func (p *idpRegistry) GetProviderByMetadataURL(_ context.Context, metadataURL string, _ ...ConfigOption) (Provider, error) {
	return p.registry.GetProviderByMetadataURL(metadataURL)
}
