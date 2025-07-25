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

// SubscriptionInvoiceSpecIntervals  (catalog.v1alpha1.SubscriptionInvoice)
type SubscriptionInvoiceSpecIntervals struct {
	// The start of the interval in ISO 8601 format with numeric timezone offset.
	From time.Time `json:"from"`
	// Number of consumed units in the interval.
	Units int32 `json:"units"`
	// In case the item is from a prior invoice for the same quota. Prior invoice items are included if the first item is not for the complete quota interval.
	PreviousInvoiceItem bool `json:"previousInvoiceItem,omitempty"`
}
