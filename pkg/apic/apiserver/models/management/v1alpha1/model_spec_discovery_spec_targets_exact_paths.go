/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// SpecDiscoverySpecTargetsExactPaths struct for SpecDiscoverySpecTargetsExactPaths
type SpecDiscoverySpecTargetsExactPaths struct {
	// path to api definition
	Path string `json:"path,omitempty"`
	// headers to add to the query
	Headers map[string]string `json:"headers,omitempty"`
}
