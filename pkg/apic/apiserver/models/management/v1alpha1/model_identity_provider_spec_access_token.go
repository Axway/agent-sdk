/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// IdentityProviderSpecAccessToken The configuration for access token based authentication with Identity provider
type IdentityProviderSpecAccessToken struct {
	Type string `json:"type"`
	// The access token to be used for authentication with Identity provider
	Token string `json:"token"`
}