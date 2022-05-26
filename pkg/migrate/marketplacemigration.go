package migrate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

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

type ardCache interface {
	GetCredentialRequestDefinitionByName(name string) (*v1.ResourceInstance, error)
	AddAccessRequestDefinition(resource *v1.ResourceInstance)
	GetAccessRequestDefinitionByName(name string) (*v1.ResourceInstance, error)
}

// NewMarketplaceMigration - creates a new MarketplaceMigration
func NewMarketplaceMigration(client client, cfg config.CentralConfig, cache ardCache) *MarketplaceMigration {
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
	logger log.FieldLogger
	client client
	cfg    config.CentralConfig
	cache  ardCache
}

// Migrate -
func (m *MarketplaceMigration) Migrate(ri *v1.ResourceInstance) (*v1.ResourceInstance, error) {
	if ri.Kind != mv1a.APIServiceGVK().Kind {
		return ri, nil
	}

	err := m.updateService(ri)
	if err != nil {
		return ri, fmt.Errorf("migration marketplace provisioning failed: %s", err)
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

	m.logger.
		WithField("service-name", ri.Name).
		Debugf("found %d revisions for api", len(revs))

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

			// Passing down apiservice name (ri.Name) for logging purposes
			// Possible future refactor to send context through to get proper resources downstream
			err := m.updateSvcInstance(url, q, ri.Name, revision)
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

func (m *MarketplaceMigration) updateSvcInstance(
	resourceURL string, query map[string]string, apiservice string, revision *v1.ResourceInstance,
) error {
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

			err := m.handleSvcInstance(apiservice, svcInstance, revision)
			if err != nil {
				errCh <- err
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

func (m *MarketplaceMigration) processAccessRequestDefinition(oauthScopes map[string]string) (string, error) {
	ard, err := m.registerAccessRequestDefinition(oauthScopes)
	if err != nil {
		return "", err
	}

	newARD, err := m.client.CreateOrUpdateResource(ard)
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

func (m *MarketplaceMigration) getCredentialRequestPolicies(authPolicies []string) ([]string, error) {
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
	for _, policy := range credentialRequestPolicies {
		if def, err := m.cache.GetCredentialRequestDefinitionByName(policy); err == nil && def != nil {
			knownCRDs = append(knownCRDs, policy)
		}
	}

	return knownCRDs
}

func (m *MarketplaceMigration) registerAccessRequestDefinition(scopes map[string]string) (*mv1a.AccessRequestDefinition, error) {
	oauthScopes := make([]string, 0)
	for scope := range scopes {
		oauthScopes = append(oauthScopes, scope)
	}

	callback := func(ard *mv1a.AccessRequestDefinition) (*mv1a.AccessRequestDefinition, error) {
		return ard, nil
	}

	var ard *mv1a.AccessRequestDefinition
	var err error
	if len(scopes) > 0 {
		ard, err = provisioning.NewAccessRequestBuilder(callback).
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
			return nil, err
		}
	}
	return ard, nil
}

// updateRI updates the resource, and the sub resource
func (m *MarketplaceMigration) updateRI(ri *v1.ResourceInstance) error {
	_, err := m.client.UpdateResourceInstance(ri)
	if err != nil {
		return err
	}

	return nil
}

func (m *MarketplaceMigration) createInstanceEndpoint(endpoints []apic.EndpointDefinition) ([]mv1a.ApiServiceInstanceSpecEndpoint, error) {
	endPoints := make([]mv1a.ApiServiceInstanceSpecEndpoint, 0)
	var err error

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

func (m *MarketplaceMigration) handleSvcInstance(
	apiservice string, svcInstance *v1.ResourceInstance, revision *v1.ResourceInstance,
) error {
	logger := m.logger.
		WithField("service-name", apiservice).
		WithField("instance-name", svcInstance.Name).
		WithField("revision-name", revision.Name)

	apiSvcInst := mv1a.NewAPIServiceInstance(svcInstance.Name, svcInstance.Metadata.Scope.Name)
	apiSvcInst.FromInstance(svcInstance)

	specProcessor, err := m.getSpecParser(revision)
	if err != nil {
		return err
	}

	var i interface{} = specProcessor

	if processor, ok := i.(apic.OasSpecProcessor); ok {
		ardRIName := apiSvcInst.Spec.AccessRequestDefinition
		credentialRequestPolicies := apiSvcInst.Spec.CredentialRequestDefinitions

		processor.ParseAuthInfo()

		// get the auth policy from the spec
		authPolicies := processor.GetAuthPolicies()

		// get the apikey info
		apiKeyInfo := processor.GetAPIKeyInfo()
		if len(apiKeyInfo) > 0 {
			logger.Debug("instance has a spec definition type of apiKey")
			ardRIName = "api-key"
		}

		// get oauth scopes
		oauthScopes := processor.GetOAuthScopes()

		var updateRequestDefinition = false

		// Check if ARD exists
		if apiSvcInst.Spec.AccessRequestDefinition == "" && len(oauthScopes) > 0 {
			// Only migrate resource with oauth scopes. Spec with type apiKey will be handled on startup
			logger.Debug("instance has a spec definition type of oauth")
			ardRIName, err = m.processAccessRequestDefinition(oauthScopes)
			if err != nil {
				return err
			}
		}

		// Check if CRD exists
		credentialRequestPolicies, err = m.getCredentialRequestPolicies(authPolicies)
		if err != nil {
			return err
		}

		// Find only the known CRDs
		credentialRequestDefinitions := m.checkCredentialRequestDefinitions(credentialRequestPolicies)
		if len(credentialRequestDefinitions) > 0 && !m.sortCompare(apiSvcInst.Spec.CredentialRequestDefinitions, credentialRequestDefinitions) {
			log.Debugf("adding the following credential request definitions %s,", credentialRequestDefinitions)
			updateRequestDefinition = true
		}

		existingARD, _ := m.cache.GetAccessRequestDefinitionByName(ardRIName)
		if existingARD == nil {
			ardRIName = ""
		} else {
			if apiSvcInst.Spec.AccessRequestDefinition == "" {
				logger.Debugf("adding the following access request definition %s", ardRIName)
				updateRequestDefinition = true
			}
		}

		if updateRequestDefinition {
			inInterface := m.newInstanceSpec(apiSvcInst.Spec.Endpoint, revision.Name, ardRIName, credentialRequestDefinitions)
			svcInstance.Spec = inInterface

			err = m.updateRI(svcInstance)
			if err != nil {
				return err
			}

			logger.Debug("migrated instance with the necessary request definitions")

			return nil
		}

		logger.Debug("no request definitions migrated for instance done at this time")
	}

	return nil
}

func (m *MarketplaceMigration) newInstanceSpec(
	endpoints []mv1a.ApiServiceInstanceSpecEndpoint,
	revisionName,
	ardRIName string,
	credentialRequestDefinitions []string,
) map[string]interface{} {
	newSpec := mv1a.ApiServiceInstanceSpec{
		Endpoint:                     endpoints,
		ApiServiceRevision:           revisionName,
		CredentialRequestDefinitions: credentialRequestDefinitions,
		AccessRequestDefinition:      ardRIName,
	}
	// convert to set ri.Spec
	var inInterface map[string]interface{}
	in, _ := json.Marshal(newSpec)
	json.Unmarshal(in, &inInterface)
	return inInterface
}

func (m *MarketplaceMigration) sortCompare(apiSvcInstCRDs, knownCRDs []string) bool {
	if len(apiSvcInstCRDs) != len(knownCRDs) {
		return false
	}

	sort.Strings(apiSvcInstCRDs)
	sort.Strings(knownCRDs)

	for i, v := range apiSvcInstCRDs {
		if v != knownCRDs[i] {
			return false
		}
	}
	return true
}

func (m *MarketplaceMigration) getSpecParser(revision *v1.ResourceInstance) (apic.SpecProcessor, error) {
	specDefinitionType, ok := revision.Spec["definition"].(map[string]interface{})["type"].(string)
	if !ok {
		return nil, fmt.Errorf("could not get the spec definition type from apiservicerevision %s", revision.Name)
	}

	specDefinitionValue, ok := revision.Spec["definition"].(map[string]interface{})["value"].(string)
	if !ok {
		return nil, fmt.Errorf("could not get the spec definition value from apiservicerevision %s", revision.Name)
	}

	specDefinition, _ := base64.StdEncoding.DecodeString(specDefinitionValue)

	specParser := apic.NewSpecResourceParser(specDefinition, specDefinitionType)
	err := specParser.Parse()
	if err != nil {
		return nil, err
	}

	err = specParser.Parse()
	if err != nil {
		return nil, err
	}

	specProcessor := specParser.GetSpecProcessor()
	return specProcessor, nil
}
