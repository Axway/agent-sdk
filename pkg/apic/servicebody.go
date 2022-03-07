package apic

import (
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

//APIKeyInfo -
type APIKeyInfo struct {
	Name     string
	Location string
}

//ServiceBody -
type ServiceBody struct {
	NameToPush                string
	APIName                   string
	RestAPIID                 string
	PrimaryKey                string
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
	AgentMode                 corecfg.AgentMode
	Image                     string
	ImageContentType          string
	CreatedBy                 string
	ResourceType              string
	AltRevisionPrefix         string
	SubscriptionName          string
	APIUpdateSeverity         string
	State                     string
	Status                    string
	ServiceAttributes         map[string]string
	RevisionAttributes        map[string]string
	InstanceAttributes        map[string]string
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
}

//SetAccessRequestDefintionName - set the name of the access request definition for this service body
func (s *ServiceBody) SetAccessRequestDefintionName(ardName string, isUnique bool) {
	s.ardName = ardName
	s.uniqueARD = isUnique
}

//GetAuthPolicies - returns the array of all auth policies in the ServiceBody
func (s *ServiceBody) GetAuthPolicies() []string {
	return s.authPolicies
}

//GetAPIKeyInfo - returns the array of locations and argument names for the api key
func (s *ServiceBody) GetAPIKeyInfo() []APIKeyInfo {
	return s.apiKeyInfo
}

//GetScopes - returns the array of scopes for this service instance
func (s *ServiceBody) GetScopes() map[string]string {
	return s.scopes
}

//GetCredentialRequestDefinitions - returns the array of all credential request policies
func (s *ServiceBody) GetCredentialRequestDefinitions() []string {
	for _, policy := range s.authPolicies {
		if policy == Apikey {
			s.credentialRequestPolicies = append(s.credentialRequestPolicies, "api-key")
			s.ardName = "api-key"
		}
		if policy == Oauth {
			s.credentialRequestPolicies = append(s.credentialRequestPolicies, "oauth")
		}
	}
	return s.credentialRequestPolicies
}
