/*
 * Amplify Unified Catalog APIs
 *
 * APIs for Amplify Unified Catalog
 *
 * API version: 1.43.0
 * Contact: support@axway.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package unifiedcatalog
// CatalogItemSubscriptionDefinition struct for CatalogItemSubscriptionDefinition
type CatalogItemSubscriptionDefinition struct {
	Enabled bool `json:"enabled,omitempty"`
	AutoSubscribe bool `json:"autoSubscribe,omitempty"`
	AutoUnsubscribe bool `json:"autoUnsubscribe,omitempty"`
	Metadata AuditMetadata `json:"metadata,omitempty"`
	Properties []CatalogItemProperty `json:"properties,omitempty"`
}
