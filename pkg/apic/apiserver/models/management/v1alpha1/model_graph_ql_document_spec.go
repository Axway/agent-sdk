/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// GraphQlDocumentSpec  (management.v1alpha1.GraphQLDocument)
type GraphQlDocumentSpec struct {
	VirtualService string `json:"virtualService"`
	//  (management.v1alpha1.GraphQLDocument)
	Graphql map[string]interface{} `json:"graphql"`
}