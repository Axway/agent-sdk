package apic

import (
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
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
	RequestDefinitionsAllowed bool
	URL                       string
	Stage                     string
	StageDescriptor           string
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
	categoryTitles            []string //Titles will be set via the service body builder
	categoryNames             []string //Names will be determined based the Title
	credentialRequestPolicies []string
	ardName                   string
	uniqueARD                 bool
	specHash                  string
	accessRequestDefinition   *management.AccessRequestDefinition
	specHashes                map[string]interface{} // map of hash values to revision names
}

// SetAccessRequestDefinitionName - set the name of the access request definition for this service body
func (s *ServiceBody) SetAccessRequestDefinitionName(ardName string, isUnique bool) {
	s.ardName = ardName
	s.uniqueARD = isUnique
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
func (s *ServiceBody) GetCredentialRequestDefinitions() []string {
	if len(s.credentialRequestPolicies) > 0 {
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
			s.credentialRequestPolicies = append(s.credentialRequestPolicies, []string{provisioning.OAuthPublicKeyCRD, provisioning.OAuthSecretCRD}...)
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
