package definitions

// Constants for attributes
const (
	XAgentDetails = "x-agent-details"
	// XWebhookDetails is the subresource an on-prem provisioning webhook is scoped to write directly
	// (see docs/discovery/provisioning-webhook.md). agent-sdk mirrors it into XAgentDetails whenever it
	// changes, so existing traceability code (which only reads XAgentDetails) needs no changes.
	XWebhookDetails                  = "x-webhook-details"
	XSubResourceHashes               = "x-subresource-hashes"
	AttrPreviousAPIServiceRevisionID = "prevAPIServiceRevisionID"
	AttrPreviousAPIServiceInstanceID = "prevAPIServiceInstanceID"
	AttrExternalAPIID                = "externalAPIID"
	AttrExternalAPIPrimaryKey        = "externalAPIPrimaryKey"
	AttrExternalAPIName              = "externalAPIName"
	AttrExternalAPIStage             = "externalAPIStage"
	AttrExternalAPIVersion           = "externalAPIVersion"
	AttrExternalAppID                = "applicationID"
	AttrExternalAppName              = "applicationName"
	AttrExternalAPISyncWarning       = "externalAPISyncWarning"
	AttrCreatedBy                    = "createdBy"
	AttrSpecHash                     = "specHash"
	Spec                             = "spec"
	MarketplaceSubResource           = "marketplace"
	ReferencesSubResource            = "references"
	Subscription                     = "Subscription"
	MarketplaceMigration             = "marketplace-migration"
	InstanceMigration                = "instance-migration"
	ComplianceAgentTrigger           = "triggerProcessing"
	TriggerTeamUpdate                = "triggerTeamUpdate"
)

// market place provisioning migration
const (
	MigrationCompleted = "completed"
)

type ExternalAppData struct {
	Key          string
	ResourceType string
}
