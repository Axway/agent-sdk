package migrate

import (
	"context"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

// ArdMigration - used for migrating access request definitions
type ArdMigration struct {
	migration
}

// NewArdMigration creates a new ArdMigration
func NewArdMigration(client client, cfg config.CentralConfig) *ArdMigration {
	return &ArdMigration{
		migration: migration{
			client: client,
			cfg:    cfg,
		},
	}
}

// Migrate checks an AccessRequestDefinition for the "scopes" key in the schema, and removes it if it is found.
func (m *ArdMigration) Migrate(_ context.Context, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	if ri.Kind != management.AccessRequestDefinitionGVK().Kind {
		return ri, nil
	}

	ard := management.NewAccessRequestDefinition("", m.cfg.GetEnvironmentName())
	err := ard.FromInstance(ri)
	if err != nil {
		return ri, err
	}

	scopes := m.getScopes(ard.Spec.Schema)
	if scopes != nil {
		res, err := m.client.UpdateResourceInstance(ard)
		if err != nil {
			return ri, err
		}
		ri = res
	}

	return ri, nil
}

func (m *ArdMigration) getScopes(schema map[string]interface{}) interface{} {
	if properties, ok := schema["properties"]; ok {
		if props, ok := properties.(map[string]interface{}); ok {
			if scopes, ok := props["scopes"]; ok {
				delete(props, "scopes")
				return scopes
			}
		}
	}
	return nil
}
