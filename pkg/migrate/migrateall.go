package migrate

import (
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type migrateAll struct {
	migrations []Migrator
	logger     log.FieldLogger
}

// NewMigrateAll creates a single Migrator out of a list of Migrators
func NewMigrateAll(m ...Migrator) Migrator {
	logger := log.NewFieldLogger().
		WithPackage("migrate").
		WithComponent("migrateAll")
	return &migrateAll{
		migrations: m,
		logger:     logger,
	}
}

// Migrate passes the resource instance to each migrate func
func (m migrateAll) Migrate(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	var err error

	for _, mig := range m.migrations {
		var e error
		ri, e = mig.Migrate(ri)
		if e != nil {
			err = e
			m.logger.
				WithError(err).
				WithField("name", ri.Name).
				WithField("kind", ri.Kind).
				Error("failed to run migration for resource")
		}
	}

	return ri, err
}
