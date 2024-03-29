/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// AssetMappingStatusSourceResource The resource that triggered the Asset Mapping.
type AssetMappingStatusSourceResource struct {
	ApiService           AssetMappingStatusSourceResourceApiService           `json:"apiService,omitempty"`
	ApiServiceRevision   AssetMappingStatusSourceResourceApiServiceRevision   `json:"apiServiceRevision,omitempty"`
	ApiServiceInstance   AssetMappingStatusSourceResourceApiServiceInstance   `json:"apiServiceInstance,omitempty"`
	AssetMappingTemplate AssetMappingStatusSourceResourceAssetMappingTemplate `json:"assetMappingTemplate,omitempty"`
}
