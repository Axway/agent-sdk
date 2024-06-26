/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// CredentialSpecState Desired state of the Credential. (management.v1alpha1.Credential)
type CredentialSpecState struct {
	Name string `json:"name"`
	// Additional info on the desired state.
	Reason string `json:"reason,omitempty"`
	// Defines if credential needs to be rotated.
	Rotate bool `json:"rotate,omitempty"`
}
