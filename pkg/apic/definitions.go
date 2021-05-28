package apic

import (
	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

// Various consts for use
const (
	API           = "API"
	Wsdl          = "wsdl"
	SwaggerV2     = "swaggerv2"
	Oas2          = "oas2"
	Oas3          = "oas3"
	Protobuf      = "protobuf"
	AsyncAPI      = "asyncapi"
	Unstructured  = "unstructured"
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
	AttrExternalAPIPrimaryKey        = "externalAPIPrimaryKey"
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
	DeprecatedStatus          = "DEPRECATED"
	PublishedStatus           = "PUBLISHED"
	UnpublishedStatus         = "UNPUBLISHED"
	UnidentifiedInboundPolicy = "UNIDENTIFIED INBOUND POLICY"
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

// EndpointDefinition - holds the service endpoint definition
type EndpointDefinition struct {
	Host     string
	Port     int32
	Protocol string
	BasePath string
}

//ServiceBody -
type ServiceBody struct {
	NameToPush        string `json:",omitempty"`
	APIName           string `json:",omitempty"`
	RestAPIID         string `json:",omitempty"`
	PrimaryKey        string `json:",omitempty"`
	URL               string `json:",omitempty"`
	Stage             string `json:",omitempty"`
	Description       string `json:",omitempty"`
	Version           string `json:",omitempty"`
	AuthPolicy        string `json:",omitempty"`
	SpecDefinition    []byte `json:",omitempty"`
	Documentation     []byte `json:",omitempty"`
	Tags              map[string]interface{}
	AgentMode         corecfg.AgentMode `json:",omitempty"`
	Image             string
	ImageContentType  string
	CreatedBy         string
	ResourceType      string
	AltRevisionPrefix string
	SubscriptionName  string
	APIUpdateSeverity string `json:",omitempty"`
	State             string
	Status            string
	ServiceAttributes map[string]string
	serviceContext    serviceContext
	Endpoints         []EndpointDefinition
	UnstructuredProps *UnstructuredProperties
}

// APIError - api response error
type APIError struct {
	Status int    `json:"status,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// ResponseError - api response errors
type ResponseError struct {
	Errors []APIError `json:"errors,omitempty"`
}

//UnstructuredProperties -
type UnstructuredProperties struct {
	AssetType   string
	ContentType string
	Label       string
	Filename    string
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
