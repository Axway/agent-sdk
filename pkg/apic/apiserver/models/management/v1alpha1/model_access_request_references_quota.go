/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// AccessRequestReferencesQuota struct for AccessRequestReferencesQuota
type AccessRequestReferencesQuota struct {
	Kind string `json:"kind"`
	Name string `json:"name,omitempty"`
	// The name of the unit for the included quota.
	Unit string `json:"unit,omitempty"`
}
