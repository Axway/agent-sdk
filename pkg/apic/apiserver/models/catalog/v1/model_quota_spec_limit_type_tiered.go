/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// QuotaSpecLimitTypeTiered struct for QuotaSpecLimitTypeTiered
type QuotaSpecLimitTypeTiered struct {
	Type  string                          `json:"type"`
	Tiers []QuotaSpecLimitTypeTieredTiers `json:"tiers"`
}