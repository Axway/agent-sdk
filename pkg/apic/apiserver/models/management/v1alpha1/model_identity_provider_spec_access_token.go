/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// IdentityProviderSpecAccessToken The access token config
type IdentityProviderSpecAccessToken struct {
	Type string `json:"type"`
	// The access token
	Token string `json:"token"`
}