/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package v1alpha1

// ProductSpecAssetFiltersReleaseVersions The asset releases to use. Once the asset has new releases, the Product will follow just the specified version.
type ProductSpecAssetFiltersReleaseVersions struct {
	Major int32 `json:"major"`
	Minor int32 `json:"minor,omitempty"`
	Patch int32 `json:"patch,omitempty"`
}
