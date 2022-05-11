package migrate

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// MarketplaceMigrator interface for performing an marketplace provisioning migration
type MarketplaceMigrator interface {
	Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
}

// NewAttributeMigration creates a new MarketplaceMigration
func NewMarketplaceMigration(client client, cfg config.CentralConfig) *MarketplaceMigration {
	return &MarketplaceMigration{
		client: client,
		cfg:    cfg,
	}
}

// MarketplaceMigration - used for migrating attributes to subresource
type MarketplaceMigration struct {
	client client
	cfg    config.CentralConfig
}

func (m *MarketplaceMigration) Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	if ri.Kind != mv1a.APIServiceGVK().Kind {
		return ri, fmt.Errorf("expected resource instance kind to be api service")
	}

	log.Debugf("migrating marketplace provisioning for service: %s", ri.Name)

	funcs := []migrateFunc{
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

	log.Debugf("finished migrating marketplace provisioning for service: %s", ri.Name)

	return ri, nil
}

// updateInst gets a list of instances for the service and updates their request definitions.
func (m *MarketplaceMigration) updateInst(ri *v1.ResourceInstance) error {
	revURL := m.cfg.GetRevisionsURL()

	q := map[string]string{
		"query": queryFunc(ri.Name),
		// "fields": "spec",
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

			specDefintionType := r.Spec["definition"].(map[string]interface{})["type"]
			specDefinitionValue := r.Spec["definition"].(map[string]interface{})["value"].(string)

			q := map[string]string{
				"query": queryFunc(r.Name),
			}
			url := m.cfg.GetInstancesURL()
			err := m.migrate(url, q, specDefinitionValue, specDefintionType.(string))
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

func (m *MarketplaceMigration) migrate(resourceURL string, query map[string]string, specDefinitionValue, specDefintionType string) error {
	resources, err := m.client.GetAPIV1ResourceInstancesWithPageSize(query, resourceURL, 100)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error, len(resources))

	for _, resource := range resources {
		wg.Add(1)

		go func(ri *v1.ResourceInstance) {
			defer wg.Done()

			// Check if ARD exists
			_, ardExists := ri.Spec["accessRequestDefinition"]
			if !ardExists {
				fmt.Println("accessRequestDefinition doesn't exist")
			}

			// Check if CRD exists
			_, crdExists := ri.Spec["credentialRequestDefinitions"]
			if !crdExists {
				fmt.Println("credentialRequestDefinitions doesn't exist")
				credentialRequestPolicies, _ := m.parseSpec(specDefinitionValue, specDefintionType) //TODO - return error causes an issue
				fmt.Printf("adding credentialRequestDefinitions %s", credentialRequestPolicies)
				// if err != nil {
				// 	return err
				// }
			}

			url := fmt.Sprintf("%s/%s", resourceURL, ri.Name)
			err := m.updateRI(url, ri)
			errCh <- err
		}(resource)
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

func (m *MarketplaceMigration) parseSpec(specDefinitionValue, specDefintionType string) ([]string, error) {

	var credentialRequestPolicies []string
	specDefinition, _ := base64.StdEncoding.DecodeString(specDefinitionValue)

	specParser := apic.NewSpecResourceParser(specDefinition, specDefintionType)
	err := specParser.Parse()
	if err != nil {
		return nil, err
	}

	specProcessor := specParser.GetSpecProcessor()

	var i interface{} = specProcessor
	if val, ok := i.(apic.OasSpecProcessor); ok {
		val.ParseAuthInfo()

		// get the auth policy from the spec
		authPolicies := val.GetAuthPolicies()

		for _, policy := range authPolicies {
			if policy == apic.Apikey {
				credentialRequestPolicies = append(credentialRequestPolicies, provisioning.APIKeyCRD)
			}
			if policy == apic.Oauth {
				credentialRequestPolicies = append(credentialRequestPolicies, []string{provisioning.OAuthPublicKeyCRD, provisioning.OAuthSecretCRD}...)
			}
		}
	}
	return credentialRequestPolicies, nil
}

// updateRI updates the resource, and the sub resource
func (m *MarketplaceMigration) updateRI(url string, ri *v1.ResourceInstance) error {
	_, err := m.client.UpdateAPIV1ResourceInstance(url, ri)
	if err != nil {
		return err
	}

	// return m.createSubResource(ri)
	return nil
}
