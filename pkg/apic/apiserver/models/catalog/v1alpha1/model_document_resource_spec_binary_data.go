/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// DocumentResourceSpecBinaryData struct for DocumentResourceSpecBinaryData
type DocumentResourceSpecBinaryData struct {
	Type string `json:"type"`
	// Base64 encoded value of the file.
	Content string `json:"content"`
	// The name of the file.
	FileName string `json:"fileName,omitempty"`
	// The type of the resource, example: pdf, markdown
	FileType string `json:"fileType"`
	// The content type
	ContentType string `json:"contentType"`
}