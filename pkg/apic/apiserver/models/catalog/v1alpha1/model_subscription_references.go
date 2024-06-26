/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// SubscriptionReferences  (catalog.v1alpha1.Subscription)
type SubscriptionReferences struct {
	// Reference a source Subscription if the Subscription was generated from a Subscription migration to a new Product Plan.
	Subscription string `json:"subscription,omitempty"`
}
