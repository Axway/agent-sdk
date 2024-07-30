package migrate

import (
	"encoding/json"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
)

// migration - used for migrating resources
type migration struct {
	client client
	cfg    config.CentralConfig
}

// updateRI updates the resource, and the sub resource
func (m *migration) updateRI(apiInterface v1.Interface) error {
	ri, _ := apiInterface.AsInstance()
	_, err := m.client.UpdateResourceInstance(ri)
	if err != nil {
		return err
	}

	return m.createSubResource(ri)
}

func (m *migration) createSubResource(ri *v1.ResourceInstance) error {
	subResources := map[string]interface{}{
		defs.XAgentDetails: ri.SubResources[defs.XAgentDetails],
	}
	return m.client.CreateSubResource(ri.ResourceMeta, subResources)
}

func (m *migration) getRI(url string) (*v1.ResourceInstance, error) {
	response, err := m.client.ExecuteAPI(api.GET, url, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving ResourceInstance: %s", err)
	}

	resourceInstance := &v1.ResourceInstance{}
	err = json.Unmarshal(response, &resourceInstance)
	return resourceInstance, err
}

func (m *migration) getAllRI(url string, q map[string]string) ([]*v1.ResourceInstance, error) {
	resources, err := m.client.GetAPIV1ResourceInstances(q, url)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving all ResourceInstances: %s", err)
	}
	return resources, nil
}

func queryFuncByMetadataID(id string) string {
	return fmt.Sprintf("metadata.references.id==%s", id)
}

func isMigrationCompleted(h v1.Interface, migrationKey string) bool {
	details := util.GetAgentDetails(h)
	if len(details) > 0 {
		completed := details[migrationKey]
		if completed == defs.MigrationCompleted {
			return true
		}
	}
	return false
}
