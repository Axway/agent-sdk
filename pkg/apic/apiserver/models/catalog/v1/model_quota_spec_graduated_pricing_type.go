/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// QuotaSpecGraduatedPricingType struct for QuotaSpecGraduatedPricingType
type QuotaSpecGraduatedPricingType struct {
	Type string `json:"type"`
	// The tiered limits to set.
	// GENERATE: The following code has been modified after code generation
	Limit interface{} `json:"limit"`
}
