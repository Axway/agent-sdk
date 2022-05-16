package migrate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
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
func NewMarketplaceMigration(client client, cfg config.CentralConfig, cache cache.Manager) *MarketplaceMigration {
	return &MarketplaceMigration{
		client: client,
		cfg:    cfg,
		cache:  cache,
	}
}

// MarketplaceMigration - used for migrating attributes to subresource
type MarketplaceMigration struct {
	client                  client
	cfg                     config.CentralConfig
	cache                   agentcache.Manager
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

			q := map[string]string{
				"query": queryFunc(r.Name),
			}
			url := m.cfg.GetInstancesURL()
			err := m.updateInstResources(url, q, r)
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

func (m *MarketplaceMigration) updateInstResources(resourceURL string, query map[string]string, resourceInstance *v1.ResourceInstance) error {
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

			apiSvcInst := mv1a.NewAPIServiceInstance(ri.Name, ri.Metadata.Scope.Name)
			apiSvcInst.FromInstance(ri)

			specDefintionType := resourceInstance.Spec["definition"].(map[string]interface{})["type"].(string)
			specDefinitionValue := resourceInstance.Spec["definition"].(map[string]interface{})["value"].(string)

			specDefinition, _ := base64.StdEncoding.DecodeString(specDefinitionValue)

			specParser := apic.NewSpecResourceParser(specDefinition, specDefintionType)
			err := specParser.Parse()
			if err != nil {
				errCh <- err
				return
			}

			specProcessor := specParser.GetSpecProcessor()
			endPoints, err := specProcessor.GetEndpoints()
			instanceSpecEndPoints, err := m.createInstanceEndpoint(endPoints)
			if err != nil {
				errCh <- err
				return
			}

			var i interface{} = specProcessor

			if val, ok := i.(apic.OasSpecProcessor); ok {

				ardRIName := apiSvcInst.Spec.AccessRequestDefinition
				credentialRequestPolicies := apiSvcInst.Spec.CredentialRequestDefinitions

				val.ParseAuthInfo()

				// get the auth policy from the spec
				authPolicies := val.GetAuthPolicies()

				// get the apikey info
				apiKeyInfo := val.GetAPIKeyInfo()
				if len(apiKeyInfo) > 0 {
					log.Debugf("apiserviceinstance %s has a spec definition type of %s", apiSvcInst.Name, "apiKey")
					ardRIName = "api-key"
				}

				// get oauth scopes
				oauthScopes := val.GetOAuthScopes()
				if len(oauthScopes) > 0 {
					log.Debugf("apiserviceinstance %s has a spec definition type of %s", apiSvcInst.Name, "oauth")
				}

				// Check if ARD exists
				if apiSvcInst.Spec.AccessRequestDefinition == "" {
					// Only migrate resource with oauth scopes. Spec with type apiKey will be handled on startup
					if len(oauthScopes) > 0 {
						err = m.registerAccessRequestDefintion(apiKeyInfo, oauthScopes)
						if err != nil {
							errCh <- err
							return
						}

						newARD, err := m.client.CreateOrUpdateResource(m.getAccessRequestDefintion())
						if err != nil {
							errCh <- err
							return
						}

						if newARD != nil {
							ard, err := newARD.AsInstance()
							if err == nil {
								m.cache.AddAccessRequestDefinition(ard)
							} else {
								errCh <- err
								return
							}
							ardRIName = ard.Name
						}
					}
				}

				// Check if CRD exists
				if len(apiSvcInst.Spec.CredentialRequestDefinitions) == 0 {
					credentialRequestPolicies, err = m.getCredentialRequestPolicies(authPolicies, ri)

					// Find only the known CRD's
					credentialRequestPolicies = m.checkCredentialRequestDefinitions(credentialRequestPolicies)
					if len(credentialRequestPolicies) > 0 {
						log.Debugf("attempt to add the following credential request definitions %s, to apiservice %s", credentialRequestPolicies, apiSvcInst.Name)
					}
				}

				existingARD, err := m.cache.GetAccessRequestDefinitionByName(ardRIName)
				if existingARD != nil {
					log.Debugf("adding the following access request definition %s to apiserviceinstance %s", ardRIName, apiSvcInst.Name)

					newSpec := mv1a.ApiServiceInstanceSpec{
						Endpoint:                     instanceSpecEndPoints,
						ApiServiceRevision:           ri.Name,
						CredentialRequestDefinitions: credentialRequestPolicies,
						AccessRequestDefinition:      ardRIName,
					}
					// convert to set ri.Spec
					var inInterface map[string]interface{}
					in, _ := json.Marshal(newSpec)
					json.Unmarshal(in, &inInterface)

					ri.Spec = inInterface

					url := fmt.Sprintf("%s/%s", resourceURL, ri.Name)
					err = m.updateRI(url, ri)
					errCh <- err
				}
			}

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

func (m *MarketplaceMigration) getCredentialRequestPolicies(authPolicies []string, ri *v1.ResourceInstance) ([]string, error) {
	var credentialRequestPolicies []string

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

func (m *MarketplaceMigration) checkCredentialRequestDefinitions(credentialRequestPolicies []string) []string {
	// remove any crd not in the cache
	knownCRDs := make([]string, 0)
	for _, credentialRequestPolicy := range credentialRequestPolicies {
		if def, err := m.cache.GetCredentialRequestDefinitionByName(credentialRequestPolicy); err == nil && def != nil {
			knownCRDs = append(knownCRDs, credentialRequestPolicy)
		}
	}

	return knownCRDs
}

func (m *MarketplaceMigration) registerAccessRequestDefintion(apiKeyInfo []apic.APIKeyInfo, scopes map[string]string) error {

	oauthScopes := make([]string, 0)
	for scope := range scopes {
		oauthScopes = append(oauthScopes, scope)
	}

	if len(scopes) > 0 {
		_, err := provisioning.NewAccessRequestBuilder(m.setAccessRequestDefintion).
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
									SetEnumValues(oauthScopes)))).Register()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MarketplaceMigration) setAccessRequestDefintion(accessRequestDefinition *mv1a.AccessRequestDefinition) (*mv1a.AccessRequestDefinition, error) {
	m.accessRequestDefinition = accessRequestDefinition
	return m.accessRequestDefinition, nil
}

// getAccessRequestDefintion -
func (m *MarketplaceMigration) getAccessRequestDefintion() *mv1a.AccessRequestDefinition {
	return m.accessRequestDefinition
}

// updateRI updates the resource, and the sub resource
func (m *MarketplaceMigration) updateRI(url string, ri *v1.ResourceInstance) error {
	_, err := m.client.UpdateResourceInstance(ri)
	if err != nil {
		return err
	}

	return nil
}

func (m *MarketplaceMigration) createInstanceEndpoint(endpoints []apic.EndpointDefinition) ([]mv1a.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := make([]mv1a.ApiServiceInstanceSpecEndpoint, 0)
	var err error

	// To set your own endpoints call AddServiceEndpoint/SetServiceEndpoint on the ServiceBodyBuilder.
	// Any endpoints provided from the ServiceBodyBuilder will override the endpoints found in the spec.
	if len(endpoints) > 0 {
		for _, endpointDef := range endpoints {
			ep := mv1a.ApiServiceInstanceSpecEndpoint{
				Host:     endpointDef.Host,
				Port:     endpointDef.Port,
				Protocol: endpointDef.Protocol,
				Routing: mv1a.ApiServiceInstanceSpecRouting{
					BasePath: endpointDef.BasePath,
				},
			}
			endPoints = append(endPoints, ep)
		}
	} else {
		log.Debug("Processing API service instance with no endpoint")
	}

	if err != nil {
		return nil, err
	}

	return endPoints, nil
}
