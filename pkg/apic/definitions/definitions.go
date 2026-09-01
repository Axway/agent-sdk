package definitions

// Constants for attributes
const (
	XAgentDetails                    = "x-agent-details"
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
