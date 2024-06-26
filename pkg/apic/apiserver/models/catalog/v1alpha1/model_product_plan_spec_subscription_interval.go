/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// ProductPlanSpecSubscriptionInterval The subscription interval (catalog.v1alpha1.ProductPlan)
type ProductPlanSpecSubscriptionInterval struct {
	// The type of the interval
	Type string `json:"type,omitempty"`
	// The subscription interval length
	// GENERATE: The following code has been modified after code generation
	Length float64 `json:"length,omitempty"`
}
