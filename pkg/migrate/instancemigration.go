package migrate

import (
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

type InstanceMigration struct {
	migration
}

// NewInstanceMigration -
func NewInstanceMigration(client client, cfg config.CentralConfig) *InstanceMigration {
	return &InstanceMigration{
		migration: migration{
			client: client,
			cfg:    cfg,
		},
	}
}

func (im *InstanceMigration) Migrate(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	if ri.Kind != management.APIServiceInstanceGVK().Kind {
		return ri, nil
	}

	ri.Finalizers = make([]apiv1.Finalizer, 0)

	return im.migration.client.UpdateResourceInstance(ri)
}
