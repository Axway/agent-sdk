package management

// GENERATE: All of the code below was replaced after code generation

// CredentialRequestDefinitionSpecProvisionPolicies Defines the policies applied on the Credentials generated using this definition. (management.v1alpha1.CredentialRequestDefinition)
type CredentialRequestDefinitionSpecProvisionPolicies struct {
	// Defines on if Credentials using this definitions can be renewed.
	Renewable bool `json:"renewable,omitempty"`
	// Defines on if Credentials can be suspended.
	Suspendable bool                                                    `json:"suspendable,omitempty"`
	Expiry      *CredentialRequestDefinitionSpecProvisionPoliciesExpiry `json:"expiry,omitempty"`
}
