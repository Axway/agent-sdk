package migrate

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
	GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*apiv1.ResourceInstance, error)
	UpdateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	CreateOrUpdateResource(data apiv1.Interface) (*apiv1.ResourceInstance, error)
	CreateSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) error
	DeleteResourceInstance(ri apiv1.Interface) error
	GetResource(url string) (*apiv1.ResourceInstance, error)
}

type item struct {
	ri     *apiv1.ResourceInstance
	update bool
}

type migrateFunc func(ri *apiv1.ResourceInstance) error

// AttributeMigration - used for migrating attributes to subresource
type AttributeMigration struct {
	migration
	riMutex sync.Mutex
}

// NewAttributeMigration creates a new AttributeMigration
func NewAttributeMigration(client client, cfg config.CentralConfig) *AttributeMigration {
	return &AttributeMigration{
		migration: migration{
			client: client,
			cfg:    cfg,
		},
		riMutex: sync.Mutex{},
	}
}

// Migrate - receives an APIService as a ResourceInstance, and checks if an attribute migration should be performed.
// If a migration should occur, then the APIService, Instances, and Revisions, that refer to the APIService will all have their attributes updated.
func (m *AttributeMigration) Migrate(_ context.Context, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	if ri.Kind != management.APIServiceGVK().Kind {
		return ri, nil
	}

	// skip migration if x-agent-details is found for the service.
	details := util.GetAgentDetails(ri)
	if len(details) > 0 {
		return ri, nil
	}

	log.Debugf("migrating attributes for service: %s", ri.Name)

	funcs := []migrateFunc{
		m.updateSvc,
		m.updateInst,
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
func (m *AttributeMigration) updateSvc(ri *apiv1.ResourceInstance) error {
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
	m.riMutex.Lock()
	defer m.riMutex.Unlock()
	*ri = *item.ri

	return m.updateRI(item.ri)
}

// updateInst gets a list of instances for the service and updates their attributes.
func (m *AttributeMigration) updateInst(ri *apiv1.ResourceInstance) error {
	m.riMutex.Lock()
	defer m.riMutex.Unlock()

	q := map[string]string{
		"query": queryFuncByMetadataID(ri.Metadata.ID),
	}
	url := m.cfg.GetInstancesURL()
	if err := m.migrate(url, q); err != nil {
		return err
	}

	return nil
}

func (m *AttributeMigration) migrate(resourceURL string, query map[string]string) error {
	resources, err := m.client.GetAPIV1ResourceInstances(query, resourceURL)
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

		go func(ri *apiv1.ResourceInstance) {
			defer wg.Done()

			err := m.updateRI(ri)
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

func updateAttrs(ri *apiv1.ResourceInstance) item {
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
