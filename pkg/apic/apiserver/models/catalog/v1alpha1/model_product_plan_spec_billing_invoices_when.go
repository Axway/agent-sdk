/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// ProductPlanSpecBillingInvoicesWhen Describes when to execute which action for a state of an invoice. (catalog.v1alpha1.ProductPlan)
type ProductPlanSpecBillingInvoicesWhen struct {
	State   string `json:"state,omitempty"`
	Trigger string `json:"trigger,omitempty"`
}
