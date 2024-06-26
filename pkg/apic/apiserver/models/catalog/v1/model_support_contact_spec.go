/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// SupportContactSpec  (catalog.v1.SupportContact)
type SupportContactSpec struct {
	// Email address of the Support Contact.
	Email string `json:"email"`
	// String of the E.164 format of the phone number, e.g. +11234567899
	PhoneNumber         string                                `json:"phoneNumber,omitempty"`
	OfficeHours         SupportContactSpecOfficeHours         `json:"officeHours,omitempty"`
	AlternativeContacts SupportContactSpecAlternativeContacts `json:"alternativeContacts,omitempty"`
}
