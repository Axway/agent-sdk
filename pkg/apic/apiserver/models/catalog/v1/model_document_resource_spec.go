/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// DocumentResourceSpec  (catalog.v1.DocumentResource)
type DocumentResourceSpec struct {
	// Document description.
	Description string `json:"description,omitempty"`
	// Version of the DocumentResource.
	Version string                    `json:"version"`
	Usage   DocumentResourceSpecUsage `json:"usage"`
	// GENERATE: The following code has been modified after code generation
	Data interface{} `json:"data"`
}