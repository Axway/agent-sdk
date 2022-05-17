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
	"github.com/Axway/agent-sdk/pkg/util/log"
)

var oldAttrs = []string{
	defs.AttrPreviousAPIServiceRevisionID,
	defs.AttrExternalAPIID,
	defs.AttrExternalAPIPrimaryKey,
	defs.AttrExternalAPIName,
	defs.AttrExternalAPIStage,
	defs.AttrCreatedBy,
}

var regexes = make([]string, 0)

var tagRegexes = make([]string, 0)

// MatchAttrPattern matches attribute patterns to match against an attribute to migrate to the x-agent-details subresource
func MatchAttrPattern(pattern ...string) {
	regexes = append(regexes, pattern...)
}

// MatchAttr matches attributes to migrate to the x-agent-details subresource
func MatchAttr(attr ...string) {
	oldAttrs = append(oldAttrs, attr...)
}

// RemoveTagPattern matches tags by a pattern for removal from the resource
func RemoveTagPattern(tags ...string) {
	tagRegexes = append(tagRegexes, tags...)
}

type client interface {
	ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	GetAPIV1ResourceInstancesWithPageSize(query map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error)
	UpdateAPIV1ResourceInstance(url string, ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
	CreateSubResourceScoped(rm v1.ResourceMeta, subs map[string]interface{}) error
	UpdateResourceInstance(ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
	CreateOrUpdateResource(data v1.Interface) (*v1.ResourceInstance, error)
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

	log.Debugf("migrating attributes for service: %s", ri.Name)

	funcs := []migrateFunc{
		m.updateSvc,
		m.updateRev,
		m.updateInst,
		m.updateCI,
	}

	errCh := make(chan error, len(funcs))
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

	log.Debugf("finished migrating attributes for service: %s", ri.Name)

	return ri, nil
}

// updateSvc updates the attributes on service in place, then updates on api server.
func (m *AttributeMigration) updateSvc(ri *v1.ResourceInstance) error {
	url := fmt.Sprintf("%s/%s", m.cfg.GetServicesURL(), ri.Name)
	r, err := m.getRI(url)
	if err != nil {
		return err
	}
	item := updateAttrs(r)
	if !item.update {
		return nil
	}

	// replace the address value so that the Migrate func can return the updated resource instance
	*ri = *item.ri

	return m.updateRI(url, item.ri)
}

// updateRev gets a list of revisions for the service and updates their attributes.
func (m *AttributeMigration) updateRev(ri *v1.ResourceInstance) error {
	url := m.cfg.GetRevisionsURL()
	q := map[string]string{
		"query": queryFunc(ri.Name),
	}

	return m.migrate(url, q)
}

// updateInst gets a list of instances for the service and updates their attributes.
func (m *AttributeMigration) updateInst(ri *v1.ResourceInstance) error {
	revURL := m.cfg.GetRevisionsURL()

	q := map[string]string{
		"query":  queryFunc(ri.Name),
		"fields": "name",
	}

	revs, err := m.client.GetAPIV1ResourceInstancesWithPageSize(q, revURL, 100)
	if err != nil {
		return err
	}

	errCh := make(chan error, len(revs))
	wg := &sync.WaitGroup{}

	// query for api service instances by reference to a revision name
	for _, rev := range revs {
		wg.Add(1)

		go func(r *v1.ResourceInstance) {
			defer wg.Done()

			q := map[string]string{
				"query": queryFunc(r.Name),
			}
			url := m.cfg.GetInstancesURL()
			err := m.migrate(url, q)
			errCh <- err
		}(rev)
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

// updateCI gets a list of consumer instances for the service and updates their attributes.
func (m *AttributeMigration) updateCI(ri *v1.ResourceInstance) error {
	url := m.cfg.GetConsumerInstancesURL()
	q := map[string]string{
		"query": queryFunc(ri.Name),
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

func (m *AttributeMigration) getRI(url string) (*v1.ResourceInstance, error) {
	response, err := m.client.ExecuteAPI(api.GET, url, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving ResourceInstance: %s", err)
	}

	resourceInstance := &v1.ResourceInstance{}
	err = json.Unmarshal(response, &resourceInstance)
	return resourceInstance, err
}

func (m *AttributeMigration) createSubResource(ri *v1.ResourceInstance) error {
	subResources := map[string]interface{}{
		defs.XAgentDetails: ri.SubResources[defs.XAgentDetails],
	}
	return m.client.CreateSubResourceScoped(ri.ResourceMeta, subResources)
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

	var tags []string

	for _, tag := range ri.Tags {
		for _, reg := range tagRegexes {
			if ok, _ := regexp.MatchString(reg, tag); ok {
				item.update = true
			} else {
				tags = append(tags, tag)
			}
		}
	}

	ri.Tags = tags

	util.SetAgentDetails(ri, details)

	return item
}

func queryFunc(name string) string {
	return fmt.Sprintf("metadata.references.name==%s", name)
}
