/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// ProductStatus  (catalog.v1.Product)
type ProductStatus struct {
	// The current status level, indicating progress towards consistency.
	Level string `json:"level"`
	// Reasons for the generated status.
	Reasons []ProductStatusReasons `json:"reasons,omitempty"`
}
