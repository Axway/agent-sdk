package migrate

import (
	"fmt"
	"regexp"
	"sync"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/hashicorp/go-version"
)

var oldAttrs = []string{
	defs.AttrPreviousAPIServiceRevisionID,
	defs.AttrExternalAPIID,
	defs.AttrExternalAPIPrimaryKey,
	defs.AttrExternalAPIName,
	defs.AttrExternalAPIStage,
	defs.AttrCreatedBy,
}

var agentAttrs = make([]string, 0)

var regexes = make([]string, 0)

// AddPattern saves a pattern to match against an attribute to migrate to a subresource
func AddPattern(pattern string) {
	regexes = append(regexes, pattern)
}

// AddAttr saves an attribute to migrate to a subresource
func AddAttr(attr string) {
	agentAttrs = append(agentAttrs, attr)
}

type client interface {
	GetAPIV1ResourceInstancesWithPageSize(query map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error)
	UpdateAPIV1ResourceInstance(url string, ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
	CreateSubResourceScoped(scopeKindPlural, scopeName, resKindPlural, name, group, version string, subs map[string]interface{}) error
}

type item struct {
	ri     *v1.ResourceInstance
	update bool
}

// AttributeMigration - used for migrating attributes to subresource
type AttributeMigration struct {
	client       client
	cfg          config.CentralConfig
	agentVersion string
	minVersion   string
}

// NewAttributeMigration creates a new AttributeMigration
func NewAttributeMigration(client client, cfg config.CentralConfig, agentVersion string) *AttributeMigration {
	return &AttributeMigration{
		client:       client,
		cfg:          cfg,
		agentVersion: agentVersion,
		minVersion:   "1.1.15",
	}
}

// Migrate - migrate attributes to subresource if the agent version is less than
// the minimum version needed for the migration
func (m *AttributeMigration) Migrate() error {
	ok := shouldMigrate(m.minVersion, m.agentVersion)
	if !ok {
		return nil
	}

	urls := []string{
		m.cfg.GetServicesURL(),
		m.cfg.GetInstancesURL(),
		m.cfg.GetRevisionsURL(),
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error, 4)

	for _, url := range urls {
		wg.Add(1)

		go func(url string) {
			defer wg.Done()

			err := m.migrate(url)
			errCh <- err
		}(url)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return fmt.Errorf("failed to perform attribute migration: %s", e)
		}
	}

	return nil
}

func (m *AttributeMigration) migrate(resourceURL string) error {
	resources, err := m.client.GetAPIV1ResourceInstancesWithPageSize(nil, resourceURL, 100)
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
		if item.update == false {
			continue
		}

		wg.Add(1)

		go func(ri *v1.ResourceInstance) {
			defer wg.Done()
			url := fmt.Sprintf("%s/%s", resourceURL, ri.Name)
			err := m.updateRes(url, ri)
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

func (m *AttributeMigration) updateRes(resUrl string, ri *v1.ResourceInstance) error {
	url := fmt.Sprintf("%s/%s", resUrl, ri.Name)
	_, err := m.client.UpdateAPIV1ResourceInstance(url, ri)
	if err != nil {
		return err
	}

	err = m.createSubResource(ri)
	if err != nil {
		return err
	}
	return nil
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

// shouldMigrate returns true if the current version is less than the minimum version
func shouldMigrate(min, current string) bool {
	minV, err := version.NewVersion(min)
	if err != nil {
		return false
	}

	currentV, err := version.NewVersion(current)
	if err != nil {
		return false
	}

	return currentV.LessThan(minV)
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

	for attr := range ri.Attributes {
		for _, reg := range regexes {
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
