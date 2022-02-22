package migrate

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
)

const queryByRefName = "metadata.references.name"

var oldAttrs = []string{
	defs.AttrPreviousAPIServiceRevisionID,
	defs.AttrExternalAPIID,
	defs.AttrExternalAPIPrimaryKey,
	defs.AttrExternalAPIName,
	defs.AttrExternalAPIStage,
	defs.AttrCreatedBy,
}

// AttrMigrator interface for performing an attribute migration
type AttrMigrator interface {
	Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
}

var regexes = make([]string, 0)

// AddPattern saves patterns to match against an attribute to migrate to the x-agent-details subresource
func AddPattern(pattern ...string) {
	regexes = append(regexes, pattern...)
}

// AddAttr saves attributes to migrate to the x-agent-details subresource
func AddAttr(attr ...string) {
	oldAttrs = append(oldAttrs, attr...)
}

type client interface {
	ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	GetAPIV1ResourceInstancesWithPageSize(query map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error)
	UpdateAPIV1ResourceInstance(url string, ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
	CreateSubResourceScoped(scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{}) error
}

type item struct {
	ri     *v1.ResourceInstance
	update bool
}

type migrateFunc func(ri *v1.ResourceInstance) error

// AttributeMigration - used for migrating attributes to subresource
type AttributeMigration struct {
	client client
	cfg    config.CentralConfig
}

// NewAttributeMigration creates a new AttributeMigration
func NewAttributeMigration(client client, cfg config.CentralConfig) *AttributeMigration {
	return &AttributeMigration{
		client: client,
		cfg:    cfg,
	}
}

// Migrate - receives an APIService as a ResourceInstance, and checks if an attribute migration should be performed.
// If a migration should occur, then the APIService, Instances, Revisions, and ConsumerInstances
// that refer to the APIService will all have their attributes updated.
func (m *AttributeMigration) Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	if ri.Kind != mv1a.APIServiceGVK().Kind {
		return ri, fmt.Errorf("expected resource instance kind to be api service")
	}

	// skip migration if x-agent-details is found for the service.
	details := util.GetAgentDetails(ri)
	if len(details) > 0 {
		return ri, nil
	}

	funcs := []migrateFunc{
		m.updateSvc,
		m.updateRev,
		m.updateInst,
		m.updateCI,
	}

	errCh := make(chan error)
	wg := &sync.WaitGroup{}

	for _, f := range funcs {
		wg.Add(1)

		go func(fun migrateFunc) {
			defer wg.Done()

			err := fun(ri)
			errCh <- err
		}(f)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return ri, e
		}
	}

	return ri, nil
}

// updateSvc updates the attributes on service in place, then updates on api server.
func (m *AttributeMigration) updateSvc(ri *v1.ResourceInstance) error {
	url := fmt.Sprintf("%s/%s", m.cfg.GetServicesURL(), ri.Name)
	ri, err := m.getRI(url)
	if err != nil {
		return err
	}
	item := updateAttrs(ri)
	if !item.update {
		return nil
	}

	return m.updateRI(url, ri)
}

// updateRev gets a list of revisions for the service and updates their attributes.
func (m *AttributeMigration) updateRev(ri *v1.ResourceInstance) error {
	url := m.cfg.GetRevisionsURL()
	q := map[string]string{
		queryByRefName: ri.Name,
	}

	return m.migrate(url, q)
}

// updateInst gets a list of instances for the service and updates their attributes.
func (m *AttributeMigration) updateInst(ri *v1.ResourceInstance) error {
	url := m.cfg.GetInstancesURL()
	q := map[string]string{
		queryByRefName: ri.Name,
	}

	return m.migrate(url, q)
}

// updateCI gets a list of consumer instances for the service and updates their attributes.
func (m *AttributeMigration) updateCI(ri *v1.ResourceInstance) error {
	url := m.cfg.GetConsumerInstancesURL()
	q := map[string]string{
		queryByRefName: ri.Name,
	}

	return m.migrate(url, q)
}

func (m *AttributeMigration) migrate(resourceURL string, query map[string]string) error {
	resources, err := m.client.GetAPIV1ResourceInstancesWithPageSize(query, resourceURL, 100)
	if err != nil {
		return err
	}

	items := make([]item, 0)

	for _, ri := range resources {
		item := updateAttrs(ri)
		items = append(items, item)
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error, len(items))

	for _, item := range items {
		if !item.update {
			continue
		}

		wg.Add(1)

		go func(ri *v1.ResourceInstance) {
			defer wg.Done()
			url := fmt.Sprintf("%s/%s", resourceURL, ri.Name)
			err := m.updateRI(url, ri)
			errCh <- err
		}(item.ri)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return e
		}
	}

	return nil
}

// updateRI updates the resource, and the sub resource
func (m *AttributeMigration) updateRI(url string, ri *v1.ResourceInstance) error {
	_, err := m.client.UpdateAPIV1ResourceInstance(url, ri)
	if err != nil {
		return err
	}

	return m.createSubResource(ri)
}

// getRI gets the resource instance
func (m *AttributeMigration) getRI(resUrl string) (*v1.ResourceInstance, error) {
	response, err := m.client.ExecuteAPI(api.GET, resUrl, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving ResourceInstance: %s", err)
	}

	resourceInstance := &v1.ResourceInstance{}
	err = json.Unmarshal(response, &resourceInstance)
	return resourceInstance, err
}

func (m *AttributeMigration) createSubResource(ri *v1.ResourceInstance) error {
	plural, err := getPlural(ri.Kind)
	if err != nil {
		return err
	}

	err = m.client.CreateSubResourceScoped(
		mv1a.EnvironmentResourceName,
		m.cfg.GetEnvironmentName(),
		plural,
		ri.Name,
		ri.Group,
		ri.APIVersion,
		ri.SubResources,
	)

	return err
}

func getPlural(kind string) (string, error) {
	switch kind {
	case mv1a.APIServiceGVK().Kind:
		return mv1a.APIServiceResourceName, nil
	case mv1a.APIServiceRevisionGVK().Kind:
		return mv1a.APIServiceRevisionResourceName, nil
	case mv1a.APIServiceInstanceGVK().Kind:
		return mv1a.APIServiceInstanceResourceName, nil
	case mv1a.ConsumerInstanceGVK().Kind:
		return mv1a.ConsumerInstanceResourceName, nil
	default:
		return "", fmt.Errorf("cannot get plural for %s", kind)
	}
}

func updateAttrs(ri *v1.ResourceInstance) item {
	details := util.GetAgentDetails(ri)
	if details == nil {
		details = make(map[string]interface{})
	}

	item := item{
		ri:     ri,
		update: false,
	}

	for _, attr := range oldAttrs {
		if _, ok := ri.Attributes[attr]; ok {
			details[attr] = ri.Attributes[attr]
			delete(ri.Attributes, attr)
			item.update = true
		}
	}

	for _, reg := range regexes {
		for attr := range ri.Attributes {
			if ok, _ := regexp.MatchString(reg, attr); ok {
				details[attr] = ri.Attributes[attr]
				delete(ri.Attributes, attr)
				item.update = true
			}
		}
	}

	util.SetAgentDetails(ri, details)

	return item
}
