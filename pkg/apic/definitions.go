package apic

import (
	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/auth"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
)

// Various consts for use
const (
	API           = "API"
	Wsdl          = "wsdl"
	SwaggerV2     = "swaggerv2"
	Oas2          = "oas2"
	Oas3          = "oas3"
	Specification = "specification"
	Swagger       = "swagger"

	SubscriptionSchemaNameSuffix      = ".authsubscription"
	DefaultSubscriptionWebhookName    = "subscriptionwebhook"
	DefaultSubscriptionWebhookAuthKey = "webhookAuthKey"
)

// Constants for attributes
const (
	AttrPreviousAPIServiceRevisionID = "prevAPIServiceRevisionID"
	AttrPreviousAPIServiceInstanceID = "prevAPIServiceInstanceID"
	AttrExternalAPIID                = "externalAPIID"
	AttrExternalAPIName              = "externalAPIName"
	AttrExternalAPIStage             = "externalAPIStage"
	AttrCreatedBy                    = "createdBy"
)

type apiErrorResponse map[string][]apiError

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// consts for state
const (
	UnpublishedState = "UNPUBLISHED"
	PublishedState   = "PUBLISHED"
)

// consts for status
const (
	DeprecatedStatus  = "DEPRECATED"
	PublishedStatus   = "PUBLISHED"
	UnpublishedStatus = "UNPUBLISHED"
)

// consts for update serverity
const (
	MajorChange = "MAJOR"
	MinorChange = "MINOR"
)

type serviceContext struct {
	serviceName      string
	serviceAction    actionType
	currentRevision  string
	revisionCount    int
	previousRevision *v1alpha1.APIServiceRevision
	revisionAction   actionType
	currentInstance  string
	instanceCount    int
	previousInstance *v1alpha1.APIServiceInstance
	instanceAction   actionType
	consumerInstance string
}

//ServiceBody -
type ServiceBody struct {
	NameToPush        string `json:",omitempty"`
	APIName           string `json:",omitempty"`
	RestAPIID         string `json:",omitempty"`
	URL               string `json:",omitempty"`
	Stage             string `json:",omitempty"`
	Description       string `json:",omitempty"`
	Version           string `json:",omitempty"`
	AuthPolicy        string `json:",omitempty"`
	Swagger           []byte `json:",omitempty"`
	Documentation     []byte `json:",omitempty"`
	Tags              map[string]interface{}
	AgentMode         corecfg.AgentMode `json:",omitempty"`
	Image             string
	ImageContentType  string
	CreatedBy         string
	ResourceType      string
	SubscriptionName  string
	APIUpdateSeverity string `json:",omitempty"`
	State             string
	Status            string
	ServiceAttributes map[string]string
	serviceContext    serviceContext
}

// ServiceClient -
type ServiceClient struct {
	tokenRequester                     auth.TokenGetter
	cfg                                corecfg.CentralConfig
	apiClient                          coreapi.Client
	DefaultSubscriptionSchema          SubscriptionSchema
	RegisteredSubscriptionSchema       SubscriptionSchema
	subscriptionMgr                    SubscriptionManager
	DefaultSubscriptionApprovalWebhook corecfg.WebhookConfig
}

// APIServerInfoProperty -
type APIServerInfoProperty struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// APIServerInfo -
type APIServerInfo struct {
	ConsumerInstance APIServerInfoProperty `json:"consumerInstance,omitempty"`
	Environment      APIServerInfoProperty `json:"environment,omitempty"`
}

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
