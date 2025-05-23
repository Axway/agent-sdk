/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

// ComplianceRuntimeResultSpecTraceableRuntime struct for ComplianceRuntimeResultSpecTraceableRuntime
type ComplianceRuntimeResultSpecTraceableRuntime struct {
	// Grade result from the compliance runtime result.
	Grade string `json:"grade,omitempty"`
	// The average risk score in the compliance runtime result.
	// GENERATE: The following code has been modified after code generation
	RiskScore float64 `json:"riskScore,omitempty"`
}
