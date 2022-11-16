package migrate

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

var defEnvName = config.NewTestCentralConfig(config.DiscoveryAgent).GetEnvironmentName()

func createRevisionsResponse(serviceName string, numRevs int) []*apiv1.ResourceInstance {
	revs := []*apiv1.ResourceInstance{}
	for i := 1; i <= numRevs; i++ {
		rev := management.NewAPIServiceRevision(fmt.Sprintf("%v.%v", serviceName, i), defEnvName)
		rev.Spec.ApiService = serviceName

		ri, _ := rev.AsInstance()
		revs = append(revs, ri)
	}
	return revs
}

func createInstanceResponse(serviceName string, numRevs int) []*apiv1.ResourceInstance {
	insts := []*apiv1.ResourceInstance{}
	for i := 1; i <= numRevs; i++ {
		inst := management.NewAPIServiceInstance(fmt.Sprintf("%v.%v", serviceName, i), defEnvName)
		inst.Spec.ApiServiceRevision = fmt.Sprintf("%v.%v", serviceName, i)

		ri, _ := inst.AsInstance()
		insts = append(insts, ri)
	}

	rand.Shuffle(len(insts), func(i, j int) {
		insts[i], insts[j] = insts[j], insts[i]
	})

	return insts
}

func TestAPISIMigration(t *testing.T) {
	tests := []struct {
		name              string
		resource          apiv1.Interface
		expectErr         bool
		turnOff           bool
		migrationComplete bool
		setMigCompelete   bool
		revisions         []*apiv1.ResourceInstance
		instances         []*apiv1.ResourceInstance
		expectedDeletes   int
	}{
		{
			name:     "called with non-apiservice returns without error",
			resource: management.NewAccessRequestDefinition("asdf", defEnvName),
		},
		{
			name:     "called with apiservice and config off returns without error",
			resource: management.NewAPIService("asdf", defEnvName),
			turnOff:  true,
		},
		{
			name:              "called with apiservice that previously was migrated",
			resource:          management.NewAPIService("asdf", defEnvName),
			setMigCompelete:   true,
			migrationComplete: true,
		},
		{
			name:              "called with apiservice and with no revisions returns without error",
			resource:          management.NewAPIService("asdf", defEnvName),
			migrationComplete: true,
		},
		{
			name:              "called with apiservice, revisions, and instances of same stage returns without error",
			resource:          management.NewAPIService("apisi", defEnvName),
			revisions:         createRevisionsResponse("apisi", 10),
			instances:         createInstanceResponse("apisi", 10),
			expectedDeletes:   9,
			migrationComplete: true,
		},
		{
			name:              "called with apiservice, revisions, and instances of diff stages returns without error",
			resource:          management.NewAPIService("apisi", defEnvName),
			revisions:         append(createRevisionsResponse("apisi-stage1", 5), createRevisionsResponse("apisi-stage2", 5)...),
			instances:         append(createInstanceResponse("apisi-stage1", 5), createInstanceResponse("apisi-stage2", 5)...),
			expectedDeletes:   8,
			migrationComplete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &mockAPISIMigClient{
				revisions: tt.revisions,
				instances: tt.instances,
			}

			cfg := config.NewTestCentralConfig(config.DiscoveryAgent)
			cfg.(*config.CentralConfiguration).MigrationSettings.(*config.MigrationSettings).CleanInstances = !tt.turnOff
			mig := NewAPISIMigration(c, cfg)

			resInst, _ := tt.resource.AsInstance()
			if tt.setMigCompelete {
				util.SetAgentDetailsKey(resInst, definitions.InstanceMigration, definitions.MigrationCompleted)
			}
			ri, err := mig.Migrate(resInst)
			if tt.expectErr {
				assert.NotNil(t, err)
				return
			}

			assert.Nil(t, err)
			migVal, _ := util.GetAgentDetailsValue(ri, definitions.InstanceMigration)
			if tt.migrationComplete {
				assert.Equal(t, definitions.MigrationCompleted, migVal)
			} else {
				assert.Equal(t, "", migVal)
			}
			assert.Equal(t, tt.expectedDeletes, c.deleteCalls)
		})
	}
}

type mockAPISIMigClient struct {
	sync.Mutex
	deleteCalls      int
	revisions        []*apiv1.ResourceInstance
	instances        []*apiv1.ResourceInstance
	instanceReturned bool
}

func (m *mockAPISIMigClient) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	return nil, nil
}

func (m *mockAPISIMigClient) GetAPIV1ResourceInstancesWithPageSize(query map[string]string, url string, pageSize int) ([]*apiv1.ResourceInstance, error) {
	m.Lock()
	defer m.Unlock()
	if m.instanceReturned {
		return nil, nil
	} else if strings.Contains(url, "instances") {
		m.instanceReturned = true
		return m.instances, nil
	}
	return m.revisions, nil
}

func (m *mockAPISIMigClient) UpdateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
	r, err := ri.AsInstance()
	return r, err
}

func (m *mockAPISIMigClient) CreateOrUpdateResource(data apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return nil, nil
}

func (m *mockAPISIMigClient) CreateSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) error {
	return nil
}

func (m *mockAPISIMigClient) DeleteResourceInstance(ri apiv1.Interface) error {
	m.deleteCalls++
	return nil
}

func (m *mockAPISIMigClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	return nil, nil
}
