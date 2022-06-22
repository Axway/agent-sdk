package migrate

import (
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
)

// ArdMigration - used for migrating access request definitions
type ArdMigration struct {
	client client
	cfg    config.CentralConfig
}

// NewArdMigration creates a new ArdMigration
func NewArdMigration(client client, cfg config.CentralConfig) *ArdMigration {
	return &ArdMigration{
		client: client,
		cfg:    cfg,
	}
}

// Migrate checks an AccessRequestDefinition for the "scopes" key in the schema, and removes it if it is found.
func (m *ArdMigration) Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	if ri.Kind != mv1a.AccessRequestDefinitionGVK().Kind {
		return ri, nil
	}

	ard := mv1a.NewAccessRequestDefinition("", m.cfg.GetEnvironmentName())
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
