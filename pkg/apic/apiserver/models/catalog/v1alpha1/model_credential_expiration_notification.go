/*
 * API Server specification.
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: SNAPSHOT
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package catalog

// CredentialExpirationNotification Details about the scheduled notification job. (catalog.v1alpha1.Credential)
type CredentialExpirationNotification struct {
	// Latest scheduled command id for sending notifications.
	CommandId string `json:"commandId,omitempty"`
	// Expiration command action. Set to 'notify' to trigger a credential expiration notification.
	Action string `json:"action,omitempty"`
}
