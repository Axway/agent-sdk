/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package management

import (
	// GENERATE: The following code has been modified after code generation
	//
	//	"time"
	time "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// ApiSpecLintingJobState  (management.v1alpha1.APISpecLintingJob)
type ApiSpecLintingJobState struct {
	// The current state, indicating progress towards consistency.
	Name string `json:"name"`
	// Details of the state.
	Message string `json:"message,omitempty"`
	// Time when the update occurred in ISO 8601 format with numeric timezone offset.
	Timestamp time.Time `json:"timestamp,omitempty"`
}
