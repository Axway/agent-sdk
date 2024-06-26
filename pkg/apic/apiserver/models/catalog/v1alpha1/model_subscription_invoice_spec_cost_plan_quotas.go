/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// SubscriptionInvoiceSpecCostPlanQuotas  (catalog.v1alpha1.SubscriptionInvoice)
type SubscriptionInvoiceSpecCostPlanQuotas struct {
	Name string `json:"name,omitempty"`
	// The cost associated with the quota.
	Cost float64 `json:"cost,omitempty"`
	// The items included in the quota cost.
	Items []SubscriptionInvoiceSpecCostPlanItems `json:"items,omitempty"`
}
