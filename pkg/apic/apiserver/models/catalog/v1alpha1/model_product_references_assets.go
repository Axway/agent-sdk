/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// ProductReferencesAssets  (catalog.v1alpha1.Product)
type ProductReferencesAssets struct {
	// The Asset reference.
	Name    string                   `json:"name,omitempty"`
	Release ProductReferencesRelease `json:"release,omitempty"`
}
