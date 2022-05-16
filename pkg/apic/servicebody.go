package apic

import (
	"sort"

	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
	AltRevisionPrefix         string
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
	accessRequestDefinition   *mv1a.AccessRequestDefinition
}

//SetAccessRequestDefintionName - set the name of the access request definition for this service body
func (s *ServiceBody) SetAccessRequestDefintionName(ardName string, isUnique bool) {
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

//GetScopes - returns the array of scopes for this service instance
func (s *ServiceBody) GetScopes() map[string]string {
	return s.scopes
}

//GetCredentialRequestDefinitions - returns the array of all credential request policies
func (s *ServiceBody) GetCredentialRequestDefinitions() []string {
	if len(s.credentialRequestPolicies) > 0 {
		return s.credentialRequestPolicies
	}
	for _, policy := range s.authPolicies {
		if policy == Apikey {
			s.credentialRequestPolicies = append(s.credentialRequestPolicies, provisioning.APIKeyCRD)
		}
		if policy == Oauth {
			s.credentialRequestPolicies = append(s.credentialRequestPolicies, []string{provisioning.OAuthPublicKeyCRD, provisioning.OAuthSecretCRD}...)
		}
	}
	return s.credentialRequestPolicies
}

func (s *ServiceBody) setAccessRequestDefintion(accessRequestDefinition *mv1a.AccessRequestDefinition) (*mv1a.AccessRequestDefinition, error) {
	s.accessRequestDefinition = accessRequestDefinition
	return s.accessRequestDefinition, nil
}

// GetAccessRequestDefintion -
func (s *ServiceBody) GetAccessRequestDefintion() *mv1a.AccessRequestDefinition {
	return s.accessRequestDefinition
}

func (s *ServiceBody) createAccessRequestDefintion() error {
	oauthScopes := make([]string, 0)
	for scope := range s.GetScopes() {
		oauthScopes = append(oauthScopes, scope)
	}
	if len(oauthScopes) > 0 {
		// sort the strings for consistent specs
		sort.Strings(oauthScopes)
		_, err := provisioning.NewAccessRequestBuilder(s.setAccessRequestDefintion).
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
