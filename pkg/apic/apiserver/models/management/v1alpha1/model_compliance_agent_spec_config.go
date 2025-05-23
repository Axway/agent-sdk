/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// ComplianceAgentSpecConfig The compliance agent config (management.v1alpha1.ComplianceAgent)
type ComplianceAgentSpecConfig struct {
	// The list of referenced managed Environments
	ManagedEnvironments []string `json:"managedEnvironments,omitempty"`
}
