package migrate

import (
	"regexp"
	"strconv"
	"sync"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
		WithComponent("APIServiceInstance Migration")

	return &APISIMigration{
		migration: migration{
			client: client,
			cfg:    cfg,
		},
		logger: logger,
	}
}

// Migrate checks an APIServiceInstance for the "scopes" key in the schema, and removes it if it is found.
func (m *APISIMigration) Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	if ri.Kind != mv1a.APIServiceGVK().Kind {
		return ri, nil
	}

	// skip migration if instance migration is not enabled
	// if !m.cfg.ShouldMigrateInstances() {
	// 	return ri, nil
	// }

	logger := m.logger.WithField(serviceName, ri.Name)

	// skip migration if instance migration has been completed
	details := util.GetAgentDetails(ri)
	if len(details) > 0 {
		completed := details[definitions.InstanceMigration]
		if completed == definitions.MigrationCompleted {
			// migration ran already
			logger.Debugf("service instance migration already completed")
			return ri, nil
		}
	}

	// get all revisions for this service
	revisions, err := m.getRevisions(ri)
	if err != nil {
		return ri, err
	}
	logger.WithField("revisions", revisions).Debug("all revisions")

	// get all instances for each revision
	wg := &sync.WaitGroup{}
	errCh := make(chan error, len(revisions))
	instances := []*v1.ResourceInstance{}
	instancesLock := sync.RWMutex{}

	for _, rev := range revisions {
		wg.Add(1)

		go func(r *v1.ResourceInstance) {
			defer wg.Done()

			revisionInstances, err := m.getInstances(r)

			instancesLock.Lock()
			defer instancesLock.Unlock()
			instances = append(instances, revisionInstances...)

			errCh <- err
		}(rev)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return ri, e
		}
	}
	logger.WithField("instances", instances).Debug("all instances")

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
func (m *APISIMigration) getRevisions(ri *v1.ResourceInstance) ([]*v1.ResourceInstance, error) {
	url := m.cfg.GetRevisionsURL()
	q := map[string]string{
		"query": queryFunc(ri.Name),
	}

	return m.getAllRI(url, q)
}

// updateRev gets a list of revisions for the service
func (m *APISIMigration) getInstances(ri *v1.ResourceInstance) ([]*v1.ResourceInstance, error) {
	url := m.cfg.GetInstancesURL()
	q := map[string]string{
		"query": queryFunc(ri.Name),
	}

	return m.getAllRI(url, q)
}

func (m *APISIMigration) cleanInstances(logger log.FieldLogger, instances []*v1.ResourceInstance) error {
	logger.Tracef("cleaning instances")
	type instanceNameIndex struct {
		ri    *v1.ResourceInstance
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
