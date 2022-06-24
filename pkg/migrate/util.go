package migrate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
)

// UpdateService - gets a list of instances for the service and updates their request definitions.
func UpdateService(ctx context.Context, ri *v1.ResourceInstance, m *MarketplaceMigration) error {
	revURL := m.Cfg.GetRevisionsURL()
	q := map[string]string{
		"query": queryFunc(ri.Name),
	}

	revs, err := m.Client.GetAPIV1ResourceInstancesWithPageSize(q, revURL, 100)
	if err != nil {
		return err
	}

	m.Logger.
		WithField(string(serviceName), ctx.Value(serviceName)).
		Tracef("found %d revisions for api", len(revs))

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
			url := m.Cfg.GetInstancesURL()

			// Passing down apiservice name (ri.Name) for logging purposes
			// Possible future refactor to send context through to get proper resources downstream
			err := updateSvcInstance(ctx, url, q, revision, m)
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

func updateSvcInstance(
	ctx context.Context, resourceURL string, query map[string]string, revision *v1.ResourceInstance, m *MarketplaceMigration) error {
	resources, err := m.Client.GetAPIV1ResourceInstancesWithPageSize(query, resourceURL, 100)
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	errCh := make(chan error, len(resources))

	for _, resource := range resources {
		wg.Add(1)

		go func(svcInstance *v1.ResourceInstance) {
			defer wg.Done()

			err := handleSvcInstance(ctx, svcInstance, revision, m)
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

func processAccessRequestDefinition(oauthScopes map[string]string, m *MarketplaceMigration) (string, error) {
	ard, err := registerAccessRequestDefinition(oauthScopes)
	if err != nil {
		return "", err
	}

	newARD, err := m.Client.CreateOrUpdateResource(ard)
	if err != nil {
		return "", err
	}
	var ardRIName string
	if newARD != nil {
		ard, err := newARD.AsInstance()
		if err == nil {
			m.Cache.AddAccessRequestDefinition(ard)
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

func checkCredentialRequestDefinitions(credentialRequestPolicies []string, m *MarketplaceMigration) []string {
	// remove any crd not in the cache
	knownCRDs := make([]string, 0)
	for _, policy := range credentialRequestPolicies {
		if def, err := m.Cache.GetCredentialRequestDefinitionByName(policy); err == nil && def != nil {
			knownCRDs = append(knownCRDs, policy)
		}
	}

	return knownCRDs
}

func registerAccessRequestDefinition(scopes map[string]string) (*mv1a.AccessRequestDefinition, error) {
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
		ard, err = provisioning.NewAccessRequestBuilder(callback).Register()
		if err != nil {
			return nil, err
		}
	}
	return ard, nil
}

// updateRI updates the resource, and the sub resource
func updateRI(ri *v1.ResourceInstance, m *MarketplaceMigration) error {
	_, err := m.Client.UpdateResourceInstance(ri)
	if err != nil {
		return err
	}

	return nil
}

func handleSvcInstance(
	ctx context.Context, svcInstance *v1.ResourceInstance, revision *v1.ResourceInstance, m *MarketplaceMigration) error {
	logger := m.Logger.
		WithField(string(serviceName), ctx.Value(serviceName)).
		WithField(instanceName, svcInstance.Name).
		WithField(revisionName, revision.Name)

	apiSvcInst := mv1a.NewAPIServiceInstance(svcInstance.Name, svcInstance.Metadata.Scope.Name)
	apiSvcInst.FromInstance(svcInstance)

	specProcessor, err := getSpecParser(revision)
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
			ardRIName = provisioning.APIKeyARD
		}

		// get oauth scopes
		oauthScopes := processor.GetOAuthScopes()

		var updateRequestDefinition = false

		// Check if ARD exists
		if apiSvcInst.Spec.AccessRequestDefinition == "" && len(oauthScopes) > 0 {
			// Only migrate resource with oauth scopes. Spec with type apiKey will be handled on startup
			logger.Debug("instance has a spec definition type of oauth")
			ardRIName, err = processAccessRequestDefinition(oauthScopes, m)
			if err != nil {
				return err
			}
		}

		// Check if CRD exists
		credentialRequestPolicies, err = getCredentialRequestPolicies(authPolicies)
		if err != nil {
			return err
		}

		// Find only the known CRDs
		credentialRequestDefinitions := checkCredentialRequestDefinitions(credentialRequestPolicies, m)
		if len(credentialRequestDefinitions) > 0 && !sortCompare(apiSvcInst.Spec.CredentialRequestDefinitions, credentialRequestDefinitions) {
			logger.Debugf("adding the following credential request definitions %s,", credentialRequestDefinitions)
			updateRequestDefinition = true
		}

		existingARD, _ := m.Cache.GetAccessRequestDefinitionByName(ardRIName)
		if existingARD == nil {
			ardRIName = ""
		} else {
			if apiSvcInst.Spec.AccessRequestDefinition == "" {
				logger.Debugf("adding the following access request definition %s", ardRIName)
				updateRequestDefinition = true
			}
		}

		if updateRequestDefinition {
			inInterface := newInstanceSpec(apiSvcInst.Spec.Endpoint, revision.Name, ardRIName, credentialRequestDefinitions)
			svcInstance.Spec = inInterface

			err = updateRI(svcInstance, m)
			if err != nil {
				return err
			}

			logger.Debugf("migrated instance %s with the necessary request definitions", apiSvcInst.Name)
		}
	}

	return nil
}

func newInstanceSpec(
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

func getSpecParser(revision *v1.ResourceInstance) (apic.SpecProcessor, error) {
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
