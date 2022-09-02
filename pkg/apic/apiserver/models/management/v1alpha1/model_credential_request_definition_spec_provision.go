package management

// GENERATE: All of the code below was replaced after code generation

// CredentialRequestDefinitionSpecProvision struct for CredentialRequestDefinitionSpecProvision
type CredentialRequestDefinitionSpecProvision struct {
	// JSON Schema draft \\#7 for defining the AccessRequest properties needed to get access to an APIServiceInstance.
	Schema   map[string]interface{}                            `json:"schema"`
	Policies *CredentialRequestDefinitionSpecProvisionPolicies `json:"policies,omitempty"`
}
