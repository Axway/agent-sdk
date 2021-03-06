/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package v1alpha1

// AssetReleaseReferences struct for AssetReleaseReferences
type AssetReleaseReferences struct {
	ApiServices           []string `json:"apiServices,omitempty"`
	AssetMappings         []string `json:"assetMappings,omitempty"`
	AssetMappingTemplates []string `json:"assetMappingTemplates,omitempty"`
}
