/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// SecretSpec struct for SecretSpec
type SecretSpec struct {
	// Key value pairs. The value will be stored encrypted.
	Data map[string]string `json:"data,omitempty"`
}
