/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// SubscriptionRequestDefinitionSpec  (catalog.v1alpha1.SubscriptionRequestDefinition)
type SubscriptionRequestDefinitionSpec struct {
	// JSON Schema draft \\#7 for defining the properties needed from a consumer to subscribe to a plan. (catalog.v1alpha1.SubscriptionRequestDefinition)
	Schema map[string]interface{} `json:"schema"`
}
