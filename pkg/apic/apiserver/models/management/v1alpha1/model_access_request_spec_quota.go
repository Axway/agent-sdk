/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// AccessRequestSpecQuota Quota information for accessing the api. (management.v1alpha1.AccessRequest)
type AccessRequestSpecQuota struct {
	// The limit of the allowed quota for the access request.
	// GENERATE: The following code has been modified after code generation
	Limit    float64 `json:"limit"`
	Interval string  `json:"interval"`
}