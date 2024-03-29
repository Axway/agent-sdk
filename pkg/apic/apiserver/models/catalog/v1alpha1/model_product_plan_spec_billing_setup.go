/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// ProductPlanSpecBillingSetup Defines the properties for the setup of the plan's subscriptions.
type ProductPlanSpecBillingSetup struct {
	// One time charge for the setup of the subscription.
	Price float64 `json:"price"`
}
