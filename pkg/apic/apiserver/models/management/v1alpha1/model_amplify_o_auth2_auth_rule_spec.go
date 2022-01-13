/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package v1alpha1

// AmplifyOAuth2AuthRuleSpec struct for AmplifyOAuth2AuthRuleSpec
type AmplifyOAuth2AuthRuleSpec struct {
	AuthorizationUrl string `json:"authorizationUrl"`
	TokenUrl         string `json:"tokenUrl"`
	RefreshUrl       string `json:"refreshUrl,omitempty"`
}