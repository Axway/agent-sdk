/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// CredentialRequestDefinitionSpecProvisionPoliciesExpiry Expiry properties for Credentials linked to this definition. (management.v1alpha1.CredentialRequestDefinition)
type CredentialRequestDefinitionSpecProvisionPoliciesExpiry struct {
	// The number of days after the Credentials are considered to be expired.
	Period int32 `json:"period"`
	// The actions taken when the Credentials expire.
	Actions []CredentialRequestDefinitionSpecProvisionPoliciesExpiryActions `json:"actions,omitempty"`
}