/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package v1alpha1

// DocumentSpec struct for DocumentSpec
type DocumentSpec struct {
	// Markdown content.
	Content string   `json:"content"`
	Stages  []string `json:"stages"`
}
