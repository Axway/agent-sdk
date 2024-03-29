/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// CredentialSpec struct for CredentialSpec
type CredentialSpec struct {
	// Reference to Credential Request Definition resource
	CredentialRequestDefinition string `json:"credentialRequestDefinition"`
	// data matching the credential request definition schema.
	Data  map[string]interface{} `json:"data"`
	State CredentialSpecState    `json:"state,omitempty"`
}
