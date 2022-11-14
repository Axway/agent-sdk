package migrate

import (
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
	logger := log.NewFieldLogger().
		WithPackage("sdk.migrate").
		WithComponent("instance-migration")

	return &APISIMigration{
		migration: migration{
			client: client,
			cfg:    cfg,
		},
		logger: logger,
	}
}

// Migrate checks an APIServiceInstance for the "scopes" key in the schema, and removes it if it is found.
func (m *APISIMigration) Migrate(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	if ri.Kind != management.APIServiceGVK().Kind {
		return ri, nil
	}

	// skip migration if instance migration is not enabled
	if !m.cfg.GetMigrationSettings().ShouldCleanInstances() {
		return ri, nil
	}

	logger := m.logger.WithField(serviceName, ri.Name)

	if isMigrationCompleted(ri, definitions.InstanceMigration) {
		// migration ran already
		logger.Debugf("service instance migration already completed")
		return ri, nil
	}

	// get all revisions for this service (should only really return the latest one)
	revisions, err := m.getRevisions(ri)

	if err != nil {
		return ri, err
	}

	instances := []*apiv1.ResourceInstance{}

	if len(revisions) > 0 {
		revision := revisions[0]
		logger.WithField("revision", revision).Debugf("got revision %s", revision.Name)
		revisionInstances, _ := m.getInstances(revision)
		instances = append(instances, revisionInstances...)
		logger.WithField("instances", instances).Debug("all instances")
	}

	err = m.cleanInstances(logger, instances)
	if err != nil {
		return ri, err
	}

	// mark the migration as complete
	util.SetAgentDetailsKey(ri, definitions.InstanceMigration, definitions.MigrationCompleted)
	err = m.updateRI(ri)
	return ri, err
}

// updateRev gets a list of revisions for the service
func (m *APISIMigration) getRevisions(ri *apiv1.ResourceInstance) ([]*apiv1.ResourceInstance, error) {
	url := m.cfg.GetRevisionsURL()
	q := map[string]string{
		"query":    queryFuncByMetadataID(ri.Metadata.ID),
		"fields":   "id",
		"sort":     "metadata.audit.createTimestamp,DESC",
		"page":     strconv.Itoa(1),
		"pageSize": strconv.Itoa(1),
	}

	return m.getAllRI(url, q)
}

// getInstances gets a list of instances for the service
func (m *APISIMigration) getInstances(ri *apiv1.ResourceInstance) ([]*apiv1.ResourceInstance, error) {
	url := m.cfg.GetInstancesURL()
	q := map[string]string{
		"query": queryFuncByMetadataName(ri.Name),
	}

	return m.getAllRI(url, q)
}

func (m *APISIMigration) cleanInstances(logger log.FieldLogger, instances []*apiv1.ResourceInstance) error {
	logger.Tracef("cleaning instances")
	type instanceNameIndex struct {
		ri    *apiv1.ResourceInstance
		index int
	}

	re := regexp.MustCompile(`([-\.a-z0-9]*)\.([0-9]*$)`)

	// sort all instances into buckets based on name, removing any index number, noting the highest
	toKeep := map[string]instanceNameIndex{}
	for _, inst := range instances {
		iLog := logger.WithField(instanceName, inst.Name)
		iLog.Tracef("handling instances")
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
		iLog = iLog.WithField("service-group", group).WithField("instance-index", index)
		iLog.Tracef("parsed instance name")

		keepIndex := -1
		if i, ok := toKeep[group]; ok {
			keepIndex = i.index
		}

		thisNameIndex := instanceNameIndex{
			ri:    inst,
			index: index,
		}

		if keepIndex == -1 {
			iLog.Trace("first instance in group")
			toKeep[group] = thisNameIndex
		} else if keepIndex < index {
			iLog.Tracef("removing previous high instance with name: %s", toKeep[group].ri.Name)
			m.client.DeleteResourceInstance(toKeep[group].ri)
			toKeep[group] = thisNameIndex
		} else {
			iLog.Tracef("removing this instance")
			m.client.DeleteResourceInstance(thisNameIndex.ri)
		}
	}

	return nil
}
