package provisioning

import (
	o "github.com/Axway/agent-sdk/pkg/authz/oauth"
)

// CredentialRequest - interface for agents to use to get necessary credential request details
type CredentialRequest interface {
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
	GetApplicationDetailsValue(key string) string
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetID returns the ID of the resource for the request
	GetID() string
	// GetName returns the name of the resource for the request
	GetName() string
	// GetCredentialDetailsValue returns a value found on the 'x-agent-details' sub resource of the Credential
	GetCredentialDetailsValue(key string) string
	// GetCredentialType returns the type of credential related to this request
	GetCredentialType() string
	// GetCredentialData returns the map[string]interface{} of data from the request
	GetCredentialData() map[string]interface{}
	// GetCredentialSchema returns the schema for the credential request.
	GetCredentialSchema() map[string]interface{}
	// GetCredentialProvisionSchema returns the provisioning schema for the credential request.
	GetCredentialProvisionSchema() map[string]interface{}
	// GetCredentialSchemaDetails returns a value found on the 'x-agent-details' sub resource of the crd.
	GetCredentialSchemaDetailsValue(key string) interface{}
	// IsIDPCredential returns boolean indicating if the credential request is for IDP provider
	IsIDPCredential() bool
	// GetIDPProvider returns the interface for IDP provider if the credential request is for IDP provider
	GetIDPProvider() o.Provider
	// GetIDPCredentialData returns the credential data for IDP from the request
	GetIDPCredentialData() IDPCredentialData
	// GetCredentialAction returns the action to be handled for this credential
	GetCredentialAction() CredentialAction
	// GetCredentialExpirationDays returns the number of days this credential has to live
	GetCredentialExpirationDays() int
}
