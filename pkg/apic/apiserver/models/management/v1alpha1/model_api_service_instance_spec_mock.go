/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// ApiServiceInstanceSpecMock Sets up the referenced API to be mocked by an Axway server. Can only be set upon creation. Requires an \"API Mocking\" entitlement.  (management.v1alpha1.APIServiceInstance)
type ApiServiceInstanceSpecMock struct {
	// Assigned to the mock server's URL base path. Must be unique for the organization.
	Name              string                                      `json:"name"`
	UseLatestRevision ApiServiceInstanceSpecMockUseLatestRevision `json:"useLatestRevision,omitempty"`
	// Optional property used to require authorization with mock-server.
	// GENERATE: The following code has been modified after code generation
	Auth interface{} `json:"auth,omitempty"`
}
