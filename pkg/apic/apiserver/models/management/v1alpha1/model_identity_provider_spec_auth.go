/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// IdentityProviderSpecAuth The authentication config to communicate with identity provider (management.v1alpha1.IdentityProvider)
type IdentityProviderSpecAuth struct {
	// The authentication type
	Type string `json:"type"`
	// GENERATE: The following code has been modified after code generation
	Config interface{} `json:"config"`
}