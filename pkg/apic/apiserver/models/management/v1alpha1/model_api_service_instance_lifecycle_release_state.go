/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// ApiServiceInstanceLifecycleReleaseState  (management.v1alpha1.APIServiceInstance)
type ApiServiceInstanceLifecycleReleaseState struct {
	// Current release state of the API endpoint(s) such as stable or deprecated.
	Name string `json:"name"`
	// Optional info to be associated with the current state. If state is deprecated, then this is intended to indicate when the servers will become decommissioned.
	Message string `json:"message,omitempty"`
}
