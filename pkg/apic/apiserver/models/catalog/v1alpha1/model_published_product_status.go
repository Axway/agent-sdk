/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// PublishedProductStatus  (catalog.v1alpha1.PublishedProduct)
type PublishedProductStatus struct {
	// The current status level, indicating progress towards consistency.
	Level string `json:"level"`
	// Reasons for the generated status.
	Reasons []PublishedProductStatusReasons `json:"reasons,omitempty"`
}