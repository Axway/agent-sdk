/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// AssetMappingTemplateSpecFilters struct for AssetMappingTemplateSpecFilters
type AssetMappingTemplateSpecFilters struct {
	ApiService []AssetMappingTemplateSpecApiService `json:"apiService,omitempty"`
	// name of the stage
	Stage string `json:"stage,omitempty"`
	// list of categories for the asset.
	Categories []string                      `json:"categories,omitempty"`
	Asset      AssetMappingTemplateSpecAsset `json:"asset,omitempty"`
}
