package migrate

import (
	"encoding/base64"
	"encoding/json"
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

// NewMarketplaceMigration - creates a new MarketplaceMigration
func NewMarketplaceMigration(client client, cfg config.CentralConfig) *MarketplaceMigration {
	return &MarketplaceMigration{
		client: client,
		cfg:    cfg,
	}
}

// MarketplaceMigration - used for migrating attributes to subresource
type MarketplaceMigration struct {
	client                  client
	cfg                     config.CentralConfig
	accessRequestDefinition *mv1a.AccessRequestDefinition
}

// Migrate -
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

	q := map[string]string{}

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

			specDefinition, _ := base64.StdEncoding.DecodeString(specDefinitionValue)

			specParser := apic.NewSpecResourceParser(specDefinition, specDefintionType)
			err := specParser.Parse()
			if err != nil {
				errCh <- err
				return
			}

			specProcessor := specParser.GetSpecProcessor()

			var ardRIName string
			var credentialRequestPolicies []string

			var i interface{} = specProcessor

			if val, ok := i.(apic.OasSpecProcessor); ok {
				val.ParseAuthInfo()

				// get the auth policy from the spec
				authPolicies := val.GetAuthPolicies()

				// get the apikey info
				apiKeyInfo := val.GetAPIKeyInfo()

				// get oauth scopes
				oauthScopes := val.GetOAuthScopes()

				// Check if ARD exists
				_, ardExists := ri.Spec["accessRequestDefinitions"]
				if !ardExists {
					log.Debug("accessRequestDefinitions doesn't exist")
					ardRIName, err = m.migrateAccessRequestDefinitions(apiKeyInfo, oauthScopes, ri)
					if err != nil {
						errCh <- err
						return
					}

					if ardRIName == "" {
						ardRIName = "api-key"
					}

					log.Debugf("adding the following access request definition %s", ardRIName)
				}

				// Check if CRD exists
				_, crdExists := ri.Spec["credentialRequestDefinitions"]
				if !crdExists {
					log.Debug("credentialRequestDefinitions doesn't exist")
					credentialRequestPolicies, err = m.migrateCredentialRequestDefinitions(authPolicies, ri)
					if err != nil {
						errCh <- err
						return
					}

					log.Debugf("adding the following credential request policies %s", credentialRequestPolicies)
				}
				newSpec := mv1a.ApiServiceInstanceSpec{
					ApiServiceRevision:           ri.Name,
					CredentialRequestDefinitions: credentialRequestPolicies,
					AccessRequestDefinition:      ardRIName,
				}

				// convert to set ri.Spec
				var inInterface map[string]interface{}
				in, _ := json.Marshal(newSpec)
				json.Unmarshal(in, &inInterface)

				ri.Spec = inInterface
			}

			url := fmt.Sprintf("%s/%s", resourceURL, ri.Name)
			err = m.updateRI(url, ri)
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

func (m *MarketplaceMigration) migrateCredentialRequestDefinitions(authPolicies []string, ri *v1.ResourceInstance) ([]string, error) {
	var credentialRequestPolicies []string

	fmt.Println("credentialRequestDefinitions doesn't exist")
	for _, policy := range authPolicies {
		if policy == apic.Apikey {
			credentialRequestPolicies = append(credentialRequestPolicies, provisioning.APIKeyCRD)
		}
		if policy == apic.Oauth {
			credentialRequestPolicies = append(credentialRequestPolicies, []string{provisioning.OAuthPublicKeyCRD, provisioning.OAuthSecretCRD}...)
		}
	}

	return credentialRequestPolicies, nil

}

func (m *MarketplaceMigration) migrateAccessRequestDefinitions(apiKeyInfo []apic.APIKeyInfo, oauthScopes map[string]string, ri *v1.ResourceInstance) (string, error) {

	scopes := make([]string, 0)
	for scope := range oauthScopes {
		scopes = append(scopes, scope)
	}

	if len(scopes) > 0 {
		ardRI, err := provisioning.NewAccessRequestBuilder(m.setAccessRequestDefintion).
			SetSchema(
				provisioning.NewSchemaBuilder().
					AddProperty(
						provisioning.NewSchemaPropertyBuilder().
							SetName("scopes").
							SetLabel("Scopes").
							IsArray().
							AddItem(
								provisioning.NewSchemaPropertyBuilder().
									SetName("scope").
									IsString().
									SetEnumValues(scopes)))).Register()
		if err != nil {
			return "", err
		}
		return ardRI.Name, nil
	}
	return "", nil
}

func (m *MarketplaceMigration) setAccessRequestDefintion(accessRequestDefinition *mv1a.AccessRequestDefinition) (*mv1a.AccessRequestDefinition, error) {
	m.accessRequestDefinition = accessRequestDefinition
	return m.accessRequestDefinition, nil
}

// updateRI updates the resource, and the sub resource
func (m *MarketplaceMigration) updateRI(url string, ri *v1.ResourceInstance) error {
	_, err := m.client.UpdateAPIV1ResourceInstance(url, ri)
	if err != nil {
		return err
	}

	return nil
}
