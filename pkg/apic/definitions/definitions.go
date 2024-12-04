package definitions

// PlatformUserInfo - Represents user resource from platform
type PlatformUserInfo struct {
	Success bool `json:"success"`
	Result  struct {
		ID        string `json:"_id"`
		GUID      string `json:"guid"`
		UserID    int64  `json:"user_id"`
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
		Active    bool   `json:"active"`
		Email     string `json:"email"`
	} `json:"result"`
}

// PlatformTeam - represents team from Central Client registry
type PlatformTeam struct {
	ID      string `json:"guid"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

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
	AttrCreatedBy                    = "createdBy"
	AttrSpecHash                     = "specHash"
	Spec                             = "spec"
	MarketplaceSubResource           = "marketplace"
	ReferencesSubResource            = "references"
	Subscription                     = "Subscription"
	MarketplaceMigration             = "marketplace-migration"
	InstanceMigration                = "instance-migration"
)

// market place provisioning migration
const (
	MigrationCompleted = "completed"
)
