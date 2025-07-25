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

// SubscriptionInvoiceBillingPaymentTypeStripe Defines Stripe type invoice payment details.
type SubscriptionInvoiceBillingPaymentTypeStripe struct {
	Type string `json:"type"`
	// Stripe Invoice id.
	Id string `json:"id,omitempty"`
	// Stripe Invoice number.
	Number string `json:"number,omitempty"`
	// Due date of the invoice in ISO 8601 format with numeric timezone offset.
	DueDate time.Time `json:"dueDate,omitempty"`
	// Issue date of the invoice in ISO 8601 format with numeric timezone offset.
	IssueDate time.Time                                         `json:"issueDate,omitempty"`
	Amount    SubscriptionInvoiceBillingPaymentTypeStripeAmount `json:"amount,omitempty"`
	// Link where the payment can be done.
	Link string `json:"link,omitempty"`
	// Link from where the invoice can be downloaded.
	DocumentLink string                                              `json:"documentLink,omitempty"`
	Customer     SubscriptionInvoiceBillingPaymentTypeStripeCustomer `json:"customer,omitempty"`
}
