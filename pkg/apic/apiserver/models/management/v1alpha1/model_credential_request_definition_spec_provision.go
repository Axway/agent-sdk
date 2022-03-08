package v1alpha1

// CredentialRequestDefinitionSpecProvision struct for CredentialRequestDefinitionSpecProvision
type CredentialRequestDefinitionSpecProvision struct {
	// JSON Schema draft \\#7 for defining the AccessRequest properties needed to get access to an APIServiceInstance.
	Schema map[string]interface{} `json:"schema"`
}
