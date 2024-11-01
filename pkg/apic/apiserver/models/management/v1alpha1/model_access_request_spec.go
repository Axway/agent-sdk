package management

// GENERATE: All of the code below was replaced after code generation

// AccessRequestSpec  (management.v1alpha1.AccessRequest)
type AccessRequestSpec struct {
	// The name of an APIServiceInstance resource that specifies where the API is deployed.
	ApiServiceInstance string `json:"apiServiceInstance"`
	// The name of an ManagedApplication resource that groups set of credentials.
	ManagedApplication string `json:"managedApplication"`
	// The value that matches the AccessRequestDefinition schema linked to the referenced APIServiceInstance. (management.v1alpha1.AccessRequest)
	Data             map[string]interface{}              `json:"data"`
	Quota            *AccessRequestSpecQuota             `json:"quota,omitempty"`
	AdditionalQuotas []AccessRequestSpecAdditionalQuotas `json:"additionalQuotas,omitempty"`
}
