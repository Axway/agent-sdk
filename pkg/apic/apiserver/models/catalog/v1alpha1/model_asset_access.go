/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// AssetAccess Defines how an asset should handle access requests received from marketplace consumers. (catalog.v1alpha1.Asset)
type AssetAccess struct {
	Approval string `json:"approval"`
}
