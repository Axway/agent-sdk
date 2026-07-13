package management

// GENERATE: All of the code below was replaced after code generation

// CredentialRequestDefinitionSpec struct for CredentialRequestDefinitionSpec
type CredentialRequestDefinitionSpec struct {
	// JSON Schema draft \\#7 for describing the fields needed to provision ManagedApplicationProfiles.
	Type 				 				string                    										`json:"type,omitempty"`
	Schema       				map[string]interface{}                       	`json:"schema"`
	Provision    				*CredentialRequestDefinitionSpecProvision    	`json:"provision,omitempty"`
	Webhooks     				[]CredentialRequestDefinitionSpecWebhook     	`json:"webhooks,omitempty"`
	// The name of the IdentityProvider.
	IdentityProvider 		string 																				`json:"identityProvider,omitempty"`
}
