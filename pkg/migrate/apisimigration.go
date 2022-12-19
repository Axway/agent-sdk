package migrate

import (
	"context"
	"regexp"
	"strconv"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// APISIMigration - used for migrating API Service Instances
type APISIMigration struct {
	migration
	logger log.FieldLogger
}

// NewAPISIMigration creates a new APISIMigration
func NewAPISIMigration(client client, cfg config.CentralConfig) *APISIMigration {
	return &APISIMigration{
		migration: migration{
			client: client,
			cfg:    cfg,
		},
		logger: log.NewFieldLogger().
			WithPackage("sdk.migrate").
			WithComponent("instance-migration"),
	}
}

// Migrate checks an APIServiceInstance for the "scopes" key in the schema, and removes it if it is found.
func (m *APISIMigration) Migrate(ctx context.Context, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	if ri.Kind != management.APIServiceGVK().Kind {
		return ri, nil
	}
	logger := log.UpdateLoggerWithContext(ctx, m.logger)
	logger.Trace("api service migration")

	// skip migration if instance migration is not enabled
	if !m.cfg.GetMigrationSettings().ShouldCleanInstances() {
		return ri, nil
	}

	if isMigrationCompleted(ri, definitions.InstanceMigration) {
		// migration ran already
		logger.Debugf("service instance migration already completed")
		return ri, nil
	}

	instances, _ := m.getInstances(ri)
	logger.WithField("instances", instances).Debug("all instances")
	if err := m.cleanInstances(ctx, instances); err != nil {
		return ri, err
	}

	// mark the migration as complete
	util.SetAgentDetailsKey(ri, definitions.InstanceMigration, definitions.MigrationCompleted)
	return ri, m.updateRI(ri)
}

// getInstances gets a list of instances for the service
func (m *APISIMigration) getInstances(ri *apiv1.ResourceInstance) ([]*apiv1.ResourceInstance, error) {
	url := m.cfg.GetInstancesURL()
	q := map[string]string{
		"query": queryFuncByMetadataID(ri.Metadata.ID),
	}

	return m.getAllRI(url, q)
}

func (m *APISIMigration) cleanInstances(ctx context.Context, instances []*apiv1.ResourceInstance) error {
	logger := log.NewLoggerFromContext(ctx)

	logger.Tracef("cleaning instances")
	type instanceNameIndex struct {
		ri    *apiv1.ResourceInstance
		index int
	}

	re := regexp.MustCompile(`([-\.a-z0-9]*)\.([0-9]*$)`)

	// sort all instances into buckets based on name, removing any index number, noting the highest
	toKeep := map[string]instanceNameIndex{}
	for _, inst := range instances {
		logger := logger.WithField(instanceName, inst.Name)
		logger.Tracef("handling instances")
		name := inst.Name
		result := re.FindAllStringSubmatch(name, -1)
		group := name
		index := 0
		var err error
		if len(result) > 0 {
			group = result[0][1]
			index, err = strconv.Atoi(result[0][2])
			if err != nil {
				return err
			}
		}
		logger.WithField("service-group", group).WithField("instance-index", index).Tracef("parsed instance name")

		keepIndex := -1
		if i, ok := toKeep[group]; ok {
			keepIndex = i.index
		}

		thisNameIndex := instanceNameIndex{
			ri:    inst,
			index: index,
		}

		if keepIndex == -1 {
			logger.Trace("first instance in group")
			toKeep[group] = thisNameIndex
		} else if keepIndex < index {
			logger.Tracef("removing previous high instance with name: %s", toKeep[group].ri.Name)
			m.client.DeleteResourceInstance(toKeep[group].ri)
			toKeep[group] = thisNameIndex
		} else {
			logger.Tracef("removing this instance")
			m.client.DeleteResourceInstance(thisNameIndex.ri)
		}
	}

	return nil
}
