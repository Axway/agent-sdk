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
	AttrPreviousAPIServiceRevisionID = "prevAPIServiceRevisionID"
	AttrPreviousAPIServiceInstanceID = "prevAPIServiceInstanceID"
	AttrExternalAPIID                = "externalAPIID"
	AttrExternalAPIPrimaryKey        = "externalAPIPrimaryKey"
	AttrExternalAPIName              = "externalAPIName"
	AttrExternalAPIStage             = "externalAPIStage"
	AttrCreatedBy                    = "createdBy"
)
