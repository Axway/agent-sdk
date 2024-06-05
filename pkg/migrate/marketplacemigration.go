package migrate

import (
	"context"
	"encoding/base64"
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

// TODO - this file should be able to be removed once Unified Catalog support has been removed
// Migrator interface for performing a migration on a ResourceInstance
type Migrator interface {
	Migrate(ctx context.Context, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error)
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
func (m *MarketplaceMigration) Migrate(ctx context.Context, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	if ri.Kind != management.APIServiceGVK().Kind || isMigrationCompleted(ri, definitions.MarketplaceMigration) {
		return ri, nil
	}
	ctx = context.WithValue(ctx, management.APIServiceCtx, ri.Name)

	logger := log.UpdateLoggerWithContext(ctx, m.logger)
	logger.Tracef("handling marketplace migration")
	err := m.UpdateService(ctx, ri)

	if err != nil {
		return ri, fmt.Errorf("migration marketplace provisioning failed: %s", err)
	}

	return ri, nil
}

// UpdateService - gets a list of instances for the service and updates their request definitions.
func (m *MarketplaceMigration) UpdateService(ctx context.Context, ri *apiv1.ResourceInstance) error {
	ctx = context.WithValue(ctx, management.APIServiceCtx, ri.Name)
	logger := log.UpdateLoggerWithContext(ctx, m.logger)

	instURL := management.NewAPIServiceInstance("", m.cfg.GetEnvironmentName()).GetKindLink()
	q := map[string]string{
		"query": queryFuncByMetadataID(ri.Metadata.ID),
	}
	apiSvcInsts, err := m.client.GetAPIV1ResourceInstancesWithPageSize(q, instURL, 100)
	if err != nil {
		return err
	}
	logger.WithField("instances", apiSvcInsts).Debug("handling api service instances")

	errCh := make(chan error, len(apiSvcInsts))
	wg := &sync.WaitGroup{}
	wg.Add(len(apiSvcInsts))

	// handle each api service instance
	for _, inst := range apiSvcInsts {
		go func(instance *apiv1.ResourceInstance) {
			defer wg.Done()

			ctx := context.WithValue(ctx, management.APIServiceInstanceCtx, instance.Name)
			err := m.handleSvcInstance(ctx, instance)
			errCh <- err
		}(inst)
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

func (m *MarketplaceMigration) handleSvcInstance(ctx context.Context, svcInstance *apiv1.ResourceInstance) error {
	logger := log.UpdateLoggerWithContext(ctx, m.logger)

	logger.Trace("handling service instance")
	apiSvcInst := management.NewAPIServiceInstance(svcInstance.Name, svcInstance.Metadata.Scope.Name)
	apiSvcInst.FromInstance(svcInstance)

	revision, err := m.client.GetResource(management.NewAPIServiceRevision(apiSvcInst.Spec.ApiServiceRevision, apiSvcInst.Metadata.Scope.Name).GetSelfLink())
	if err != nil {
		logger.WithError(err).Error("error retrieving revision")
		return err
	}
	specProcessor, err := getSpecParser(revision)
	if err != nil {
		logger.WithError(err).Error("error parsing revision spec")
		return err
	}

	var i interface{} = specProcessor

	if processor, ok := i.(apic.OasSpecProcessor); ok {
		return m.processInstance(ctx, apiSvcInst, revision, processor)
	}

	return nil
}

func (m *MarketplaceMigration) processInstance(ctx context.Context, apiSvcInst *management.APIServiceInstance, revision *apiv1.ResourceInstance, processor apic.OasSpecProcessor) error {
	logger := log.UpdateLoggerWithContext(ctx, m.logger)

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

	var updateRequestDefinition = false
	var err error

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
	if len(credentialRequestDefinitions) > 0 && !sortCompare(apiSvcInst.Spec.CredentialRequestDefinitions, credentialRequestDefinitions) {
		logger.Debugf("adding the following credential request definitions %s to apiserviceinstance %s", credentialRequestDefinitions, apiSvcInst.Name)
		apiSvcInst.Spec.CredentialRequestDefinitions = credentialRequestDefinitions
		updateRequestDefinition = true
	}

	existingARD, _ := m.cache.GetAccessRequestDefinitionByName(ardRIName)
	if existingARD == nil {
		ardRIName = ""
	} else {
		if apiSvcInst.Spec.AccessRequestDefinition == "" {
			logger.Debugf("adding the following access request definition %s to apiserviceinstance %s", ardRIName, apiSvcInst.Name)
			apiSvcInst.Spec.AccessRequestDefinition = ardRIName
			updateRequestDefinition = true
		}
	}

	// update apiserivceinstance spec with necessary request definitions
	if updateRequestDefinition {
		err = m.updateRI(apiSvcInst)
		if err != nil {
			return err
		}

		logger.Debugf("migrated instance %s with the necessary request definitions", apiSvcInst.Name)
	}
	return nil
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
