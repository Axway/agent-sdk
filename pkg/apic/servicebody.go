package apic

import (
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// APIKeyInfo -
type APIKeyInfo struct {
	Name     string
	Location string
}

// ServiceBody - details about a service to create
type ServiceBody struct {
	NameToPush                string
	APIName                   string
	RestAPIID                 string
	PrimaryKey                string
	URL                       string
	Stage                     string
	StageDescriptor           string
	StageDisplayName          string
	Description               string
	Version                   string
	AuthPolicy                string
	authPolicies              []string
	apiKeyInfo                []APIKeyInfo
	scopes                    map[string]string
	SpecDefinition            []byte
	Documentation             []byte
	Tags                      map[string]interface{}
	Image                     string
	ImageContentType          string
	CreatedBy                 string
	ResourceContentType       string
	ResourceType              string
	SubscriptionName          string
	APIUpdateSeverity         string
	State                     string
	Status                    string
	ServiceAttributes         map[string]string
	RevisionAttributes        map[string]string
	InstanceAttributes        map[string]string
	ServiceAgentDetails       map[string]interface{}
	InstanceAgentDetails      map[string]interface{}
	RevisionAgentDetails      map[string]interface{}
	serviceContext            serviceContext
	Endpoints                 []EndpointDefinition
	UnstructuredProps         *UnstructuredProperties
	TeamName                  string
	teamID                    string
	credentialRequestPolicies []string
	ardName                   string
	uniqueARD                 bool
	ignoreSpecBasesCreds      bool
	specHash                  string
	specVersion               string
	accessRequestDefinition   *management.AccessRequestDefinition
	specHashes                map[string]interface{} // map of hash values to revision names
	requestDefinitionsAllowed bool                   // used to validate if the instance can have request definitions or not. Use case example - v7 unpublished, remove request definitions
	dataplaneType             DataplaneType
	isDesignDataplane         bool
	referencedServiceName     string
	referencedInstanceName    string
	logger                    log.FieldLogger
	instanceLifecycle         *management.ApiServiceInstanceLifecycle
}

// SetAccessRequestDefinitionName - set the name of the access request definition for this service body
func (s *ServiceBody) SetAccessRequestDefinitionName(ardName string, isUnique bool) {
	s.ardName = ardName
	s.uniqueARD = isUnique
}

func (s *ServiceBody) SetIgnoreSpecBasedCreds(ignore bool) {
	s.ignoreSpecBasesCreds = ignore
}

// GetAuthPolicies - returns the array of all auth policies in the ServiceBody
func (s *ServiceBody) GetAuthPolicies() []string {
	return s.authPolicies
}

// GetAPIKeyInfo - returns the array of locations and argument names for the api key
func (s *ServiceBody) GetAPIKeyInfo() []APIKeyInfo {
	return s.apiKeyInfo
}

// GetScopes - returns the array of scopes for this service instance
func (s *ServiceBody) GetScopes() map[string]string {
	return s.scopes
}

// GetCredentialRequestDefinitions - returns the array of all credential request policies
func (s *ServiceBody) GetCredentialRequestDefinitions(allowedOAuthMethods []string) []string {
	if len(s.credentialRequestPolicies) > 0 || s.ignoreSpecBasesCreds {
		return s.credentialRequestPolicies
	}
	for _, policy := range s.authPolicies {
		if policy == Basic {
			s.credentialRequestPolicies = append(s.credentialRequestPolicies, provisioning.BasicAuthCRD)
		}
		if policy == Apikey {
			s.credentialRequestPolicies = append(s.credentialRequestPolicies, provisioning.APIKeyCRD)
		}
		if policy == Oauth {
			oauthCRDs := []string{provisioning.OAuthPublicKeyCRD, provisioning.OAuthSecretCRD}
			if len(allowedOAuthMethods) > 0 {
				oauthCRDs = allowedOAuthMethods
			}
			s.credentialRequestPolicies = append(s.credentialRequestPolicies, oauthCRDs...)
		}
	}
	return s.credentialRequestPolicies
}

func (s *ServiceBody) setAccessRequestDefinition(accessRequestDefinition *management.AccessRequestDefinition) (*management.AccessRequestDefinition, error) {
	s.accessRequestDefinition = accessRequestDefinition
	return s.accessRequestDefinition, nil
}

// GetAccessRequestDefinition -
func (s *ServiceBody) GetAccessRequestDefinition() *management.AccessRequestDefinition {
	return s.accessRequestDefinition
}

func (s *ServiceBody) createAccessRequestDefinition() error {
	if s.ignoreSpecBasesCreds {
		s.logger.WithField("ardName", s.ardName).Debug("skipping registering new ARD")
		return nil
	}
	oauthScopes := make([]string, 0)
	for scope := range s.GetScopes() {
		oauthScopes = append(oauthScopes, scope)
	}
	if len(oauthScopes) > 0 {
		// sort the strings for consistent specs
		_, err := provisioning.NewAccessRequestBuilder(s.setAccessRequestDefinition).Register()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetSpecVersion - returns version parsed from the spec
func (s *ServiceBody) GetSpecVersion() string {
	return s.specVersion
}

// GetDataplaneType - returns dataplane type
func (s *ServiceBody) GetDataplaneType() DataplaneType {
	return s.dataplaneType
}

// IsDesignDataplane - returns true for design dataplane
func (s *ServiceBody) IsDesignDataplane() bool {
	return s.isDesignDataplane
}

func (s *ServiceBody) GetReferencedServiceName() string {
	return s.referencedServiceName
}

func (s *ServiceBody) GetReferenceInstanceName() string {
	return s.referencedInstanceName
}

func (s *ServiceBody) GetInstanceLifeCycle() *management.ApiServiceInstanceLifecycle {
	return s.instanceLifecycle
}
