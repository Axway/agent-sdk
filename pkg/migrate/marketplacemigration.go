package migrate

import (
	"context"
	"fmt"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Migrator interface for performing a migration on a ResourceInstance
type Migrator interface {
	Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
}

type ardCache interface {
	GetCredentialRequestDefinitionByName(name string) (*v1.ResourceInstance, error)
	AddAccessRequestDefinition(resource *v1.ResourceInstance)
	GetAccessRequestDefinitionByName(name string) (*v1.ResourceInstance, error)
}

// NewMarketplaceMigration - creates a new MarketplaceMigration
func NewMarketplaceMigration(client client, cfg config.CentralConfig, cache ardCache) *MarketplaceMigration {
	logger := log.NewFieldLogger().
		WithPackage("sdk.migrate").
		WithComponent("MarketplaceMigration")

	return &MarketplaceMigration{
		Logger: logger,
		Client: client,
		Cfg:    cfg,
		Cache:  cache,
	}
}

// MarketplaceMigration - used for migrating attributes to subresource
type MarketplaceMigration struct {
	Logger log.FieldLogger
	Client client
	Cfg    config.CentralConfig
	Cache  ardCache
}

// Migrate -
func (m *MarketplaceMigration) Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	if ri.Kind != mv1a.APIServiceGVK().Kind {
		return ri, nil
	}

	ctx := context.WithValue(context.Background(), serviceName, ri.Name)

	// check resource to see if this apiservice has already been run through migration
	apiSvc, err := ri.AsInstance()
	if err != nil {
		return nil, err
	}

	// get x-agent-details and determine if we need to process this apiservice for marketplace provisioning
	details := util.GetAgentDetails(apiSvc)
	if len(details) > 0 {
		completed := details[definitions.MarketplaceMigration]
		if completed == definitions.MigrationCompleted {
			// migration ran already
			m.Logger.
				WithField(serviceName, apiSvc).
				Debugf("marketplace provision migration already completed")
			return ri, nil
		}
	}

	m.Logger.
		WithField(serviceName, ri.Name).
		Tracef("perform marketplace provision")

	UpdateService(ctx, ri, m)
	// err = m.updateService(ctx, ri)
	if err != nil {
		return ri, fmt.Errorf("migration marketplace provisioning failed: %s", err)
	}

	return ri, nil
}
