/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

import (
	// GENERATE: The following code has been modified after code generation
	//
	//	"time"
	time "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
)

// ApplicationProfileStatusReasons  (catalog.v1.ApplicationProfile)
type ApplicationProfileStatusReasons struct {
	Type string `json:"type"`
	// Details of the error.
	Detail string `json:"detail"`
	// Time when the update occurred in ISO 8601 format with numeric timezone offset.
	Timestamp time.Time `json:"timestamp"`
	//  (catalog.v1.ApplicationProfile)
	Meta map[string]map[string]interface{} `json:"meta,omitempty"`
}
