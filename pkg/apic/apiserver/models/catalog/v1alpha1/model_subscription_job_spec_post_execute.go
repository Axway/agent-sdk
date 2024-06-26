/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// SubscriptionJobSpecPostExecute  (catalog.v1alpha1.SubscriptionJob)
type SubscriptionJobSpecPostExecute struct {
	// Actions to be executed after the new Subscription was created.
	OnSuccess []map[string]interface{} `json:"onSuccess,omitempty"`
}
