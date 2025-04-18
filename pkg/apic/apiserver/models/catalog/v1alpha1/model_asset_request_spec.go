/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// AssetRequestSpec  (catalog.v1alpha1.AssetRequest)
type AssetRequestSpec struct {
	//  (catalog.v1alpha1.AssetRequest)
	Data          map[string]interface{} `json:"data"`
	AssetResource string                 `json:"assetResource"`
	// reference to the Subscription to be used to access the Asset Resource.
	Subscription string `json:"subscription,omitempty"`
	// The AssetRequest from which this resource is being migrated from. Reference must be in the same Application.
	AssetRequest string `json:"assetRequest,omitempty"`
	// A reference to the Product for which the request was done.
	Product string `json:"product,omitempty"`
	// A reference to the ProductRelease that contained the asset resource for which the request was done.
	ProductRelease string `json:"productRelease,omitempty"`
}
