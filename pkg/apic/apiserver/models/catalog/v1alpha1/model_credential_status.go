/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package v1alpha1

// CredentialStatus  (catalog.v1alpha1.Credential)
type CredentialStatus struct {
	// The current status level, indicating progress towards consistency.
	Level string `json:"level"`
	// Reasons for the generated credential status.
	Reasons []CredentialStatusReasons `json:"reasons,omitempty"`
}