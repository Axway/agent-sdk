package apic

import (
	"sync"

	cache2 "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/util/log"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
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
	GraphQL       = "graphql-sdl"
	Raml          = "RAML"

	SubscriptionSchemaNameSuffix      = ".authsubscription"
	DefaultSubscriptionWebhookName    = "subscriptionwebhook"
	DefaultSubscriptionWebhookAuthKey = "webhookAuthKey"

	FieldsKey = "fields"
	QueryKey  = "query"

	CreateTimestampQueryKey = "metadata.audit.createTimestamp"

	DefaultTeamKey = "DefaultTeam"
)

// consts for state
const (
	UnpublishedState     = "UNPUBLISHED"
	PublishedState       = "PUBLISHED"
	ApprovalPendingState = "PENDING"
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

// consts for RAML versions
const (
	Raml08 = "RAML 0.8"
	Raml10 = "RAML 1.0"
)

type serviceContext struct {
	serviceName          string
	serviceID            string
	serviceAction        actionType
	revisionName         string
	revisionCount        int
	instanceName         string
	consumerInstanceName string
}

// EndpointDefinition - holds the service endpoint definition
type EndpointDefinition struct {
	Host     string
	Port     int32
	Protocol string
	BasePath string
	Details  map[string]interface{}
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

// UnstructuredProperties -
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
	caches                             cache2.Manager
	subscriptionSchemaCache            cache.Cache
	subscriptionMgr                    SubscriptionManager
	DefaultSubscriptionApprovalWebhook corecfg.WebhookConfig
	subscriptionRegistrationLock       sync.Mutex
	logger                             log.FieldLogger
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
