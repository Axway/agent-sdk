/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// DocumentSpec  (catalog.v1.Document)
type DocumentSpec struct {
	// Document description.
	Description string `json:"description,omitempty"`
	// Rank of document.
	// GENERATE: The following code has been modified after code generation
	Rank     float64                `json:"rank,omitempty"`
	Sections []DocumentSpecSections `json:"sections,omitempty"`
}
