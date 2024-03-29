/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// ProductSpecAutoRelease Defines if a product should create releases everytime there is an update to the product references.
type ProductSpecAutoRelease struct {
	// Description of the generated release tag.
	Description              string                                         `json:"description,omitempty"`
	ReleaseType              string                                         `json:"releaseType"`
	ReleaseVersionProperties ProductSpecAutoReleaseReleaseVersionProperties `json:"releaseVersionProperties,omitempty"`
	PreviousReleases         ProductSpecAutoReleasePreviousReleases         `json:"previousReleases,omitempty"`
}
