package migrate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	serviceName  = "service-name"
	instanceName = "instance-name"
	revisionName = "revision-name"
)

var apiserviceName string

// Migrator interface for performing a migration on a ResourceInstance
type Migrator interface {
	Migrate(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error)
}

type ardCache interface {
	GetCredentialRequestDefinitionByName(name string) (*apiv1.ResourceInstance, error)
	AddAccessRequestDefinition(resource *apiv1.ResourceInstance)
	GetAccessRequestDefinitionByName(name string) (*apiv1.ResourceInstance, error)
}

// MarketplaceMigration - used for migrating attributes to subresource
type MarketplaceMigration struct {
	migration
	logger log.FieldLogger
	cache  ardCache
}

// NewMarketplaceMigration - creates a new MarketplaceMigration
func NewMarketplaceMigration(client client, cfg config.CentralConfig, cache ardCache) *MarketplaceMigration {
	logger := log.NewFieldLogger().
		WithPackage("sdk.migrate").
		WithComponent("MarketplaceMigration")

	return &MarketplaceMigration{
		migration: migration{
			client: client,
			cfg:    cfg,
		},
		logger: logger,
		cache:  cache,
	}
}

// Migrate -
func (m *MarketplaceMigration) Migrate(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	if ri.Kind != management.APIServiceGVK().Kind || m.InstanceAlreadyMigrated(ri) {
		return ri, nil
	}

	err := m.UpdateService(ri)

	if err != nil {
		return ri, fmt.Errorf("migration marketplace provisioning failed: %s", err)
	}

	return ri, nil
}

// UpdateService - gets a list of instances for the service and updates their request definitions.
func (m *MarketplaceMigration) UpdateService(ri *apiv1.ResourceInstance) error {
	revURL := m.cfg.GetRevisionsURL()
	q := map[string]string{
		"query": queryFunc(ri.Name),
	}

	revs, err := m.client.GetAPIV1ResourceInstancesWithPageSize(q, revURL, 100)
	if err != nil {
		return err
	}

	apiserviceName = ri.Name
	m.logger.
		WithField(serviceName, apiserviceName).
		Tracef("found %d revisions for api", len(revs))

	errCh := make(chan error, len(revs))
	wg := &sync.WaitGroup{}

	// query for api service instances by reference to a revision name
	for _, rev := range revs {
		wg.Add(1)

		go func(revision *apiv1.ResourceInstance) {
			defer wg.Done()

			q := map[string]string{
				"query": queryFunc(revision.Name),
			}
			url := m.cfg.GetInstancesURL()

			// Passing down apiservice name (ri.Name) for logging purposes
			// Possible future refactor to send context through to get proper resources downstream
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

// InstanceAlreadyMigrated - check to see if apiservice already migrated
func (m *MarketplaceMigration) InstanceAlreadyMigrated(ri *apiv1.ResourceInstance) bool {

	// get x-agent-details and determine if we need to process this apiservice for marketplace provisioning
	if isMigrationCompleted(ri, definitions.MarketplaceMigration) {
		return true
	}

	m.logger.
		WithField(serviceName, ri.Name).
		Tracef("perform marketplace provision")

	return false
}

func (m *MarketplaceMigration) updateSvcInstance(
	resourceURL string, query map[string]string, revision *apiv1.ResourceInstance) error {
	resources, err := m.client.GetAPIV1ResourceInstancesWithPageSize(query, resourceURL, 100)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error, len(resources))

	for _, resource := range resources {
		wg.Add(1)

		go func(svcInstance *apiv1.ResourceInstance) {
			defer wg.Done()

			err := m.handleSvcInstance(svcInstance, revision)
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

func getCredentialRequestPolicies(authPolicies []string) ([]string, error) {
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

func (m *MarketplaceMigration) registerAccessRequestDefinition(scopes map[string]string) (*management.AccessRequestDefinition, error) {

	callback := func(ard *management.AccessRequestDefinition) (*management.AccessRequestDefinition, error) {
		return ard, nil
	}

	var ard *management.AccessRequestDefinition
	var err error
	if len(scopes) > 0 {
		ard, err = provisioning.NewAccessRequestBuilder(callback).Register()
		if err != nil {
			return nil, err
		}
	}
	return ard, nil
}

func (m *MarketplaceMigration) handleSvcInstance(
	svcInstance *apiv1.ResourceInstance, revision *apiv1.ResourceInstance) error {
	logger := m.logger.
		WithField(serviceName, apiserviceName).
		WithField(instanceName, svcInstance.Name).
		WithField(revisionName, revision.Name)

	apiSvcInst := management.NewAPIServiceInstance(svcInstance.Name, svcInstance.Metadata.Scope.Name)
	apiSvcInst.FromInstance(svcInstance)

	specProcessor, err := getSpecParser(revision)
	if err != nil {
		return err
	}

	var i interface{} = specProcessor

	if processor, ok := i.(apic.OasSpecProcessor); ok {
		ardRIName := apiSvcInst.Spec.AccessRequestDefinition

		processor.ParseAuthInfo()

		// get the auth policy from the spec
		authPolicies := processor.GetAuthPolicies()

		// get the apikey info
		apiKeyInfo := processor.GetAPIKeyInfo()
		if len(apiKeyInfo) > 0 {
			logger.Trace("instance has a spec definition type of apiKey")
			ardRIName = provisioning.APIKeyARD
		}

		// get oauth scopes
		oauthScopes := processor.GetOAuthScopes()

		// Check if ARD exists
		if apiSvcInst.Spec.AccessRequestDefinition == "" && len(oauthScopes) > 0 {
			// Only migrate resource with oauth scopes. Spec with type apiKey will be handled on startup
			logger.Trace("instance has a spec definition type of oauth")
			ardRIName, err = m.processAccessRequestDefinition(oauthScopes)
			if err != nil {
				return err
			}
		}

		// Check if CRD exists
		credentialRequestPolicies, err := getCredentialRequestPolicies(authPolicies)
		if err != nil {
			return err
		}

		// Find only the known CRDs
		credentialRequestDefinitions := m.checkCredentialRequestDefinitions(credentialRequestPolicies)
		m.updateRequestDefinition(credentialRequestDefinitions, ardRIName, revision, apiSvcInst, svcInstance, logger)

	}

	return nil
}

func (m *MarketplaceMigration) updateRequestDefinition(credentialRequestDefinitions []string, ardRIName string, revision *apiv1.ResourceInstance,
	apiSvcInst *management.APIServiceInstance, svcInstance *apiv1.ResourceInstance, logger log.FieldLogger) error {
	var updateRequestDefinition = false

	if len(credentialRequestDefinitions) > 0 && !sortCompare(apiSvcInst.Spec.CredentialRequestDefinitions, credentialRequestDefinitions) {
		logger.Debugf("adding the following credential request definitions %s to apiserviceinstance %s", credentialRequestDefinitions, apiSvcInst.Name)
		updateRequestDefinition = true
	}

	existingARD, _ := m.cache.GetAccessRequestDefinitionByName(ardRIName)
	if existingARD == nil {
		ardRIName = ""
	} else {
		if apiSvcInst.Spec.AccessRequestDefinition == "" {
			logger.Debugf("adding the following access request definition %s to apiserviceinstance %s", ardRIName, apiSvcInst.Name)
			updateRequestDefinition = true
		}
	}

	// update apiserivceinstane spec with necessary request definitions
	if updateRequestDefinition {
		inInterface := m.newInstanceSpec(apiSvcInst.Spec.Endpoint, revision.Name, ardRIName, credentialRequestDefinitions)
		svcInstance.Spec = inInterface

		err := m.updateRI(svcInstance)
		if err != nil {
			return err
		}

		logger.Debugf("migrated instance %s with the necessary request definitions", apiSvcInst.Name)
	}
	return nil
}

func (m *MarketplaceMigration) newInstanceSpec(
	endpoints []management.ApiServiceInstanceSpecEndpoint,
	revisionName,
	ardRIName string,
	credentialRequestDefinitions []string,
) map[string]interface{} {
	newSpec := management.ApiServiceInstanceSpec{
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

func sortCompare(apiSvcInstCRDs, knownCRDs []string) bool {
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

func getSpecParser(revision *apiv1.ResourceInstance) (apic.SpecProcessor, error) {
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
