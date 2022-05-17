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

// Migrator interface for performing a migration on a ResourceInstance
type Migrator interface {
	Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error)
}

// NewMarketplaceMigration - creates a new MarketplaceMigration
func NewMarketplaceMigration(client client, cfg config.CentralConfig, cache cache.Manager) *MarketplaceMigration {
	logger := log.NewFieldLogger().
		WithPackage("sdk.migrate").
		WithComponent("MarketplaceMigration")

	return &MarketplaceMigration{
		logger: logger,
		client: client,
		cfg:    cfg,
		cache:  cache,
	}
}

// MarketplaceMigration - used for migrating attributes to subresource
type MarketplaceMigration struct {
	logger                  log.FieldLogger
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

	err := m.updateService(ri)
	if err != nil {
		return nil, fmt.Errorf("migration marketplace provisioning failed")
	}

	return ri, nil
}

// updateService gets a list of instances for the service and updates their request definitions.
func (m *MarketplaceMigration) updateService(ri *v1.ResourceInstance) error {

	revURL := m.cfg.GetRevisionsURL()
	q := map[string]string{
		"query": queryFunc(ri.Name),
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

		go func(revision *v1.ResourceInstance) {
			defer wg.Done()

			q := map[string]string{
				"query": queryFunc(revision.Name),
			}
			url := m.cfg.GetInstancesURL()
			err := m.updateSvcInstance(url, q, revision)
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

func (m *MarketplaceMigration) updateSvcInstance(resourceURL string, query map[string]string, revision *v1.ResourceInstance) error {
	resources, err := m.client.GetAPIV1ResourceInstancesWithPageSize(query, resourceURL, 100)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error, len(resources))

	for _, resource := range resources {
		wg.Add(1)

		go func(svcInstance *v1.ResourceInstance) {
			defer wg.Done()

			apiSvcInst := mv1a.NewAPIServiceInstance(svcInstance.Name, svcInstance.Metadata.Scope.Name)
			apiSvcInst.FromInstance(svcInstance)

			// get spec definition type from apiservicerevision
			specDefintionType, ok := revision.Spec["definition"].(map[string]interface{})["type"].(string)
			if !ok {
				errCh <- fmt.Errorf("could not get the spec definition type from apiservicerevision %s", revision.Name)
			}

			// get spec definition value from apiservicerevision
			specDefinitionValue, ok := revision.Spec["definition"].(map[string]interface{})["value"].(string)
			if !ok {
				errCh <- fmt.Errorf("could not get the spec definition value from apiservicerevision %s", revision.Name)
			}

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
					m.logger.Debugf("apiserviceinstance %s has a spec definition type of %s", apiSvcInst.Name, "apiKey")
					ardRIName = "api-key"
				}

				// get oauth scopes
				oauthScopes := val.GetOAuthScopes()
				if len(oauthScopes) > 0 {
					m.logger.Debugf("apiserviceinstance %s has a spec definition type of %s", apiSvcInst.Name, "oauth")
				}

				var updateRequestDefinition = false

				// Check if ARD exists
				if apiSvcInst.Spec.AccessRequestDefinition == "" {
					// Only migrate resource with oauth scopes. Spec with type apiKey will be handled on startup
					if len(oauthScopes) > 0 {
						ardRIName, err = m.processAccessRequestDefinition(apiKeyInfo, oauthScopes)
						if err != nil {
							errCh <- err
						}
					}
				}

				// Check if CRD exists
				if len(apiSvcInst.Spec.CredentialRequestDefinitions) == 0 {
					credentialRequestPolicies, err = m.getCredentialRequestPolicies(authPolicies, svcInstance)

					// Find only the known CRD's
					credentialRequestPolicies = m.checkCredentialRequestDefinitions(credentialRequestPolicies)
					if len(credentialRequestPolicies) > 0 {
						m.logger.Debugf("adding the following credential request definitions %s, to apiserviceinstance %s", credentialRequestPolicies, apiSvcInst.Name)
						updateRequestDefinition = true
					}
				}

				existingARD, err := m.cache.GetAccessRequestDefinitionByName(ardRIName)
				if existingARD != nil && apiSvcInst.Spec.AccessRequestDefinition == "" {
					m.logger.Debugf("adding the following access request definition %s to apiserviceinstance %s", ardRIName, apiSvcInst.Name)
					updateRequestDefinition = true
				}

				if updateRequestDefinition {
					newSpec := mv1a.ApiServiceInstanceSpec{
						Endpoint:                     instanceSpecEndPoints,
						ApiServiceRevision:           revision.Name,
						CredentialRequestDefinitions: credentialRequestPolicies,
						AccessRequestDefinition:      ardRIName,
					}
					// convert to set ri.Spec
					var inInterface map[string]interface{}
					in, _ := json.Marshal(newSpec)
					json.Unmarshal(in, &inInterface)

					svcInstance.Spec = inInterface

					url := fmt.Sprintf("%s/%s", resourceURL, svcInstance.Name)
					err = m.updateRI(url, svcInstance)
					if err != nil {
						errCh <- err
						return
					} else {
						m.logger.Debugf("migrated %s with the necessary request definitions", apiSvcInst.Name)
					}
				} else {
					m.logger.Debugf("no request definitions migrated for apiserviceinstance %s done at this time", apiSvcInst.Name)
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

func (m *MarketplaceMigration) processAccessRequestDefinition(apiKeyInfo []apic.APIKeyInfo, oauthScopes map[string]string) (string, error) {
	err := m.registerAccessRequestDefintion(apiKeyInfo, oauthScopes)
	if err != nil {
		return "", err
	}

	newARD, err := m.client.CreateOrUpdateResource(m.getAccessRequestDefintion())
	if err != nil {
		return "", err
	}
	var ardRIName string
	if newARD != nil {
		ard, err := newARD.AsInstance()
		if err == nil {
			m.cache.AddAccessRequestDefinition(ard)
		} else {
			return "", err
		}
		ardRIName = ard.Name
	}
	return ardRIName, nil
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
		m.logger.Debug("Processing API service instance with no endpoint")
	}

	if err != nil {
		return nil, err
	}

	return endPoints, nil
}
