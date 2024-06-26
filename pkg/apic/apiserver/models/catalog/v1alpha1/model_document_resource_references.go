/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// DocumentResourceReferences  (catalog.v1alpha1.DocumentResource)
type DocumentResourceReferences struct {
	// The marketplaces this DocumentResource is being used in as part of the marketplace settings.
	MarketplaceSettings []DocumentResourceReferencesMarketplaceSettings `json:"marketplaceSettings,omitempty"`
	PlatformSettings    DocumentResourceReferencesPlatformSettings      `json:"platformSettings,omitempty"`
}
