
package management

// AccessRequestDefinitionSpec  (management.v1alpha1.AccessRequestDefinition)
type AccessRequestDefinitionSpec struct {
	// JSON Schema draft \\#7 for defining the AccessRequest properties needed to get access to an APIServiceInstance. (management.v1alpha1.AccessRequestDefinition)
	Schema    map[string]interface{}                `json:"schema"`
	Provision *AccessRequestDefinitionSpecProvision `json:"provision,omitempty"`
}