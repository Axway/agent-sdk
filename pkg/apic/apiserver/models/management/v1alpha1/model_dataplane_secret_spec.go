/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// DataplaneSecretSpec  (management.v1alpha1.DataplaneSecret)
type DataplaneSecretSpec struct {
	Dataplane string `json:"dataplane"`
	// Key value pairs for accessing the dataplane. The value will be stored encrypted.
	Data string `json:"data"`
}
